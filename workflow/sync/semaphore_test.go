package sync

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
	syncdb "github.com/argoproj/argo-workflows/v4/util/sync/db"
)

// semaphoreFactory is a function that creates a semaphore for testing
type semaphoreFactory func(ctx context.Context, t *testing.T, name, namespace string, limit int, nextWorkflow NextWorkflow) (semaphore, *sqldb.SessionProxy, func())

// createTestInternalSemaphore creates an in-memory semaphore for testing
func createTestInternalSemaphore(ctx context.Context, t *testing.T, name, namespace string, limit int, nextWorkflow NextWorkflow) (semaphore, *sqldb.SessionProxy, func()) {
	t.Helper()
	sem, err := newInternalSemaphore(ctx, name, nextWorkflow, func(ctx context.Context, _ string) (int, error) { return limit, nil }, 0)
	require.NoError(t, err)
	return sem, nil, func() {}
}

// createTestDatabaseSemaphore creates a database-backed semaphore for testing, used elsewhere
func createTestDatabaseSemaphore(ctx context.Context, t *testing.T, name, namespace string, limit int, cacheTTL time.Duration, nextWorkflow NextWorkflow, dbType sqldb.DBType) (*databaseSemaphore, syncdb.DBInfo, func()) {
	t.Helper()
	info, deferfunc, _, err := createTestDBSession(ctx, t, dbType)
	require.NoError(t, err)

	dbKey := fmt.Sprintf("%s/%s", namespace, name)
	_, err = info.SessionProxy.Session().SQL().Exec("INSERT INTO sync_limit (name, sizelimit) VALUES (?, ?)", dbKey, limit)
	require.NoError(t, err)

	s, err := newDatabaseSemaphore(ctx, name, dbKey, nextWorkflow, info, cacheTTL)
	require.NoError(t, err)
	require.NotNil(t, s)

	return s, info, deferfunc
}

// createTestDatabaseSemaphorePostgres creates a database-backed semaphore that conforms to the factory
func createTestDatabaseSemaphorePostgres(ctx context.Context, t *testing.T, name, namespace string, limit int, nextWorkflow NextWorkflow) (semaphore, *sqldb.SessionProxy, func()) {
	t.Helper()
	s, info, deferfunc := createTestDatabaseSemaphore(ctx, t, name, namespace, limit, 0, nextWorkflow, sqldb.Postgres)
	return s, info.SessionProxy, deferfunc
}

// createTestDatabaseSemaphoreMySQL creates a database-backed semaphore that conforms to the factory
func createTestDatabaseSemaphoreMySQL(ctx context.Context, t *testing.T, name, namespace string, limit int, nextWorkflow NextWorkflow) (semaphore, *sqldb.SessionProxy, func()) {
	t.Helper()
	s, info, deferfunc := createTestDatabaseSemaphore(ctx, t, name, namespace, limit, 0, nextWorkflow, sqldb.MySQL)
	return s, info.SessionProxy, deferfunc
}

// semaphoreFactories defines the available semaphore implementations for testing
var semaphoreFactories map[string]semaphoreFactory

// Don't test databases on windows as testcontainers don't work there
func init() {
	switch runtime.GOOS {
	case "windows":
		semaphoreFactories = map[string]semaphoreFactory{
			"InternalSemaphore": createTestInternalSemaphore,
		}
	default:
		semaphoreFactories = map[string]semaphoreFactory{
			"InternalSemaphore": createTestInternalSemaphore,
			"PostgresSemaphore": createTestDatabaseSemaphorePostgres,
			"MySQLSemaphore":    createTestDatabaseSemaphoreMySQL,
		}
	}
}

// TestIsSameWorkflowNodeKeys tests the isSameWorkflowNodeKeys function
func TestIsSameWorkflowNodeKeys(t *testing.T) {
	wfkey1 := "default/wf-1"
	wfkey2 := "default/wf-2"
	nodeWf1key1 := "default/wf-1/node-1"
	nodeWf1key2 := "default/wf-1/node-2"
	nodeWf2key1 := "default/wf-2/node-1"
	nodeWf2key2 := "default/wf-2/node-2"
	assert.True(t, isSameWorkflowNodeKeys(nodeWf1key1, nodeWf1key2))
	assert.True(t, isSameWorkflowNodeKeys(wfkey1, wfkey1))
	assert.False(t, isSameWorkflowNodeKeys(nodeWf1key1, nodeWf2key1))
	assert.False(t, isSameWorkflowNodeKeys(wfkey1, wfkey2))
	assert.True(t, isSameWorkflowNodeKeys(nodeWf2key1, nodeWf2key2))
}

// testTryAcquireSemaphore tests the tryAcquire method for one semaphore implementation
func testTryAcquireSemaphore(t *testing.T, factory semaphoreFactory) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	nextWorkflow := func(key string) {}

	s, sessionProxy, cleanup := factory(ctx, t, "bar", "default", 2, nextWorkflow)
	defer cleanup()

	now := time.Now()
	tx := &transaction{sessionProxy: sessionProxy}
	require.NoError(t, s.addToQueue(ctx, "default/wf-01", 0, now))
	require.NoError(t, s.addToQueue(ctx, "default/wf-02", 0, now.Add(time.Second)))
	require.NoError(t, s.addToQueue(ctx, "default/wf-03", 0, now.Add(2*time.Second)))
	require.NoError(t, s.addToQueue(ctx, "default/wf-04", 0, now.Add(3*time.Second)))
	// verify only the first in line is allowed to acquired the semaphore
	var acquired bool
	acquired, _ = s.tryAcquire(ctx, "default/wf-04", tx)
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire(ctx, "default/wf-03", tx)
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire(ctx, "default/wf-02", tx)
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire(ctx, "default/wf-01", tx)
	assert.True(t, acquired)
	// now that wf-01 obtained it, wf-02 can
	acquired, _ = s.tryAcquire(ctx, "default/wf-02", tx)
	assert.True(t, acquired)
	acquired, _ = s.tryAcquire(ctx, "default/wf-03", tx)
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire(ctx, "default/wf-04", tx)
	assert.False(t, acquired)
}

// TestTryAcquireSemaphore runs the tryAcquire test for all semaphore implementations
func TestTryAcquireSemaphore(t *testing.T) {
	for name, factory := range semaphoreFactories {
		t.Run(name, func(t *testing.T) {
			testTryAcquireSemaphore(t, factory)
		})
	}
}

// testNotifyWaitersAcquire tests the notifyWaiters method for one semaphore implementation
func testNotifyWaitersAcquire(t *testing.T, factory semaphoreFactory) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	notified := make(map[string]bool)
	nextWorkflow := func(key string) {
		notified[key] = true
	}

	s, sessionProxy, cleanup := factory(ctx, t, "bar", "default", 3, nextWorkflow)
	defer cleanup()

	now := time.Now()
	// The ordering here is important and perhaps counterintuitive.
	require.NoError(t, s.addToQueue(ctx, "default/wf-04", 0, now.Add(3*time.Second)))
	require.NoError(t, s.addToQueue(ctx, "default/wf-02", 0, now.Add(time.Second)))
	require.NoError(t, s.addToQueue(ctx, "default/wf-01", 0, now))
	require.NoError(t, s.addToQueue(ctx, "default/wf-05", 0, now.Add(4*time.Second)))
	require.NoError(t, s.addToQueue(ctx, "default/wf-03", 0, now.Add(2*time.Second)))

	tx := &transaction{sessionProxy: sessionProxy}
	acquired, _ := s.tryAcquire(ctx, "default/wf-01", tx)
	assert.True(t, acquired)

	assert.Len(t, notified, 2)
	assert.True(t, notified["default/wf-02"])
	assert.True(t, notified["default/wf-03"])
	assert.False(t, notified["default/wf-04"])
	assert.False(t, notified["default/wf-05"])

	notified = make(map[string]bool)
	released := s.release(ctx, "default/wf-01")
	assert.True(t, released)

	assert.Len(t, notified, 3)
	assert.True(t, notified["default/wf-02"])
	assert.True(t, notified["default/wf-03"])
	assert.True(t, notified["default/wf-04"])
	assert.False(t, notified["default/wf-05"])
}

// TestNotifyWaitersAcquire runs the notifyWaiters test for all semaphore implementations
func TestNotifyWaitersAcquire(t *testing.T) {
	for name, factory := range semaphoreFactories {
		t.Run(name, func(t *testing.T) {
			testNotifyWaitersAcquire(t, factory)
		})
	}
}

// testNotifyWorkflowFromTemplateSemaphore tests the template semaphore behavior for one semaphore` implementation
func testNotifyWorkflowFromTemplateSemaphore(t *testing.T, factory semaphoreFactory) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	notified := make(map[string]bool)
	nextWorkflow := func(key string) {
		notified[key] = true
	}

	s, sessionProxy, cleanup := factory(ctx, t, "bar", "foo", 2, nextWorkflow)
	defer cleanup()

	now := time.Now()
	require.NoError(t, s.addToQueue(ctx, "foo/wf-01/nodeid-123", 0, now))
	require.NoError(t, s.addToQueue(ctx, "foo/wf-02/nodeid-456", 0, now.Add(time.Second)))

	tx := &transaction{sessionProxy: sessionProxy}
	acquired, _ := s.tryAcquire(ctx, "foo/wf-01/nodeid-123", tx)
	assert.True(t, acquired)

	assert.Len(t, notified, 1)
	assert.True(t, notified["foo/wf-02"])
}

// TestNotifyWorkflowFromTemplateSemaphore runs the template semaphore test for all implementations
func TestNotifyWorkflowFromTemplateSemaphore(t *testing.T) {
	for name, factory := range semaphoreFactories {
		t.Run(name, func(t *testing.T) {
			testNotifyWorkflowFromTemplateSemaphore(t, factory)
		})
	}
}

// testCheckAcquireNotifiesCorrectKeyForTemplateSemaphore verifies that when a non-front workflow
// calls checkAcquire, the front workflow is re-queued using the workflow-level key
// (namespace/workflow), not the raw template-level key (namespace/workflow/nodeid).
func testCheckAcquireNotifiesCorrectKeyForTemplateSemaphore(t *testing.T, factory semaphoreFactory) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	notified := make(map[string]bool)
	nextWorkflow := func(key string) {
		notified[key] = true
	}

	s, sessionProxy, cleanup := factory(ctx, t, "bar", "foo", 2, nextWorkflow)
	defer cleanup()

	now := time.Now()
	require.NoError(t, s.addToQueue(ctx, "foo/wf-01/node-aaa", 0, now))
	require.NoError(t, s.addToQueue(ctx, "foo/wf-02/node-bbb", 0, now.Add(time.Second)))

	tx := &transaction{sessionProxy: sessionProxy}
	// wf-02 is not first in queue, so checkAcquire should notify the front (wf-01)
	// via nextWorkflow with the workflow-level key, not the template-level key
	acquired, _, _ := s.checkAcquire(ctx, "foo/wf-02/node-bbb", tx)
	assert.False(t, acquired)

	assert.True(t, notified["foo/wf-01"], "nextWorkflow should receive workflow key 'foo/wf-01', not template key 'foo/wf-01/node-aaa'")
	assert.False(t, notified["foo/wf-01/node-aaa"], "nextWorkflow should not receive raw template-level key")
}

// TestCheckAcquireNotifiesCorrectKeyForTemplateSemaphore runs the checkAcquire template key test for all implementations
func TestCheckAcquireNotifiesCorrectKeyForTemplateSemaphore(t *testing.T) {
	for name, factory := range semaphoreFactories {
		t.Run(name, func(t *testing.T) {
			testCheckAcquireNotifiesCorrectKeyForTemplateSemaphore(t, factory)
		})
	}
}
