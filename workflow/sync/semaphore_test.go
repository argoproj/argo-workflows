package sync

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upper/db/v4"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

// semaphoreFactory is a function that creates a semaphore for testing
type semaphoreFactory func(t *testing.T, name, namespace string, limit int, nextWorkflow NextWorkflow) (semaphore, db.Session, func())

// createTestInternalSemaphore creates an in-memory semaphore for testing
func createTestInternalSemaphore(t *testing.T, name, namespace string, limit int, nextWorkflow NextWorkflow) (semaphore, db.Session, func()) {
	t.Helper()
	sem, err := newInternalSemaphore(name, nextWorkflow, func(_ string) (int, error) { return limit, nil }, 0)
	require.NoError(t, err)
	return sem, nil, func() {}
}

// createTestDatabaseSemaphore creates a database-backed semaphore for testing, used elsewhere
func createTestDatabaseSemaphore(t *testing.T, name, namespace string, limit int, cacheTTL time.Duration, nextWorkflow NextWorkflow, dbType sqldb.DBType) (*databaseSemaphore, dbInfo, func()) {
	t.Helper()
	info, deferfunc, _, err := createTestDBSession(t, dbType)
	require.NoError(t, err)

	dbKey := fmt.Sprintf("%s/%s", namespace, name)
	_, err = info.session.SQL().Exec("INSERT INTO sync_limit (name, sizelimit) VALUES (?, ?)", dbKey, limit)
	require.NoError(t, err)

	s := newDatabaseSemaphore(name, dbKey, nextWorkflow, info, cacheTTL)
	require.NotNil(t, s)

	return s, info, deferfunc
}

// createTestDatabaseSemaphorePostgres creates a database-backed semaphore that conforms to the factory
func createTestDatabaseSemaphorePostgres(t *testing.T, name, namespace string, limit int, nextWorkflow NextWorkflow) (semaphore, db.Session, func()) {
	t.Helper()
	s, info, deferfunc := createTestDatabaseSemaphore(t, name, namespace, limit, 0, nextWorkflow, sqldb.Postgres)
	return s, info.session, deferfunc
}

// createTestDatabaseSemaphoreMySQL creates a database-backed semaphore that conforms to the factory
func createTestDatabaseSemaphoreMySQL(t *testing.T, name, namespace string, limit int, nextWorkflow NextWorkflow) (semaphore, db.Session, func()) {
	t.Helper()
	s, info, deferfunc := createTestDatabaseSemaphore(t, name, namespace, limit, 0, nextWorkflow, sqldb.MySQL)
	return s, info.session, deferfunc
}

// semaphoreFactories defines the available semaphore implementations for testing
var semaphoreFactories = map[string]semaphoreFactory{
	"InternalSemaphore": createTestInternalSemaphore,
	"PostgresSemaphore": createTestDatabaseSemaphorePostgres,
	"MySQLSemaphore":    createTestDatabaseSemaphoreMySQL,
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

// testTryAcquireSemaphore tests the tryAcquire method for both semaphore implementations
func testTryAcquireSemaphore(t *testing.T, factory semaphoreFactory) {
	t.Helper()
	nextWorkflow := func(key string) {}

	s, dbSession, cleanup := factory(t, "bar", "default", 2, nextWorkflow)
	defer cleanup()

	now := time.Now()
	tx := &transaction{db: &dbSession}
	s.addToQueue("default/wf-01", 0, now, tx)
	s.addToQueue("default/wf-02", 0, now.Add(time.Second), tx)
	s.addToQueue("default/wf-03", 0, now.Add(2*time.Second), tx)
	s.addToQueue("default/wf-04", 0, now.Add(3*time.Second), tx)

	// verify only the first in line is allowed to acquired the semaphore
	var acquired bool
	acquired, _ = s.tryAcquire("default/wf-04", tx)
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire("default/wf-03", tx)
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire("default/wf-02", tx)
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire("default/wf-01", tx)
	assert.True(t, acquired)
	// now that wf-01 obtained it, wf-02 can
	acquired, _ = s.tryAcquire("default/wf-02", tx)
	assert.True(t, acquired)
	acquired, _ = s.tryAcquire("default/wf-03", tx)
	assert.False(t, acquired)
	acquired, _ = s.tryAcquire("default/wf-04", tx)
	assert.False(t, acquired)
}

// TestTryAcquireSemaphore runs the tryAcquire test for both semaphore implementations
func TestTryAcquireSemaphore(t *testing.T) {
	for name, factory := range semaphoreFactories {
		t.Run(name, func(t *testing.T) {
			testTryAcquireSemaphore(t, factory)
		})
	}
}

// testNotifyWaitersAcquire tests the notifyWaiters method for both semaphore implementations
func testNotifyWaitersAcquire(t *testing.T, factory semaphoreFactory) {
	t.Helper()
	notified := make(map[string]bool)
	nextWorkflow := func(key string) {
		notified[key] = true
	}

	s, dbSession, cleanup := factory(t, "bar", "default", 3, nextWorkflow)
	defer cleanup()

	now := time.Now()
	tx := &transaction{db: &dbSession}
	s.addToQueue("default/wf-04", 0, now.Add(3*time.Second), tx)
	s.addToQueue("default/wf-02", 0, now.Add(time.Second), tx)
	s.addToQueue("default/wf-01", 0, now, tx)
	s.addToQueue("default/wf-05", 0, now.Add(4*time.Second), tx)
	s.addToQueue("default/wf-03", 0, now.Add(2*time.Second), tx)

	acquired, _ := s.tryAcquire("default/wf-01", tx)
	assert.True(t, acquired)

	assert.Len(t, notified, 2)
	assert.True(t, notified["default/wf-02"])
	assert.True(t, notified["default/wf-03"])
	assert.False(t, notified["default/wf-04"])
	assert.False(t, notified["default/wf-05"])

	notified = make(map[string]bool)
	released := s.release("default/wf-01")
	assert.True(t, released)

	assert.Len(t, notified, 3)
	assert.True(t, notified["default/wf-02"])
	assert.True(t, notified["default/wf-03"])
	assert.True(t, notified["default/wf-04"])
	assert.False(t, notified["default/wf-05"])
}

// TestNotifyWaitersAcquire runs the notifyWaiters test for both semaphore implementations
func TestNotifyWaitersAcquire(t *testing.T) {
	for name, factory := range semaphoreFactories {
		t.Run(name, func(t *testing.T) {
			testNotifyWaitersAcquire(t, factory)
		})
	}
}

// testNotifyWorkflowFromTemplateSemaphore tests the template semaphore behavior for both implementations
func testNotifyWorkflowFromTemplateSemaphore(t *testing.T, factory semaphoreFactory) {
	t.Helper()
	notified := make(map[string]bool)
	nextWorkflow := func(key string) {
		notified[key] = true
	}

	s, dbSession, cleanup := factory(t, "bar", "foo", 2, nextWorkflow)
	defer cleanup()

	now := time.Now()
	tx := &transaction{db: &dbSession}
	s.addToQueue("foo/wf-01/nodeid-123", 0, now, tx)
	s.addToQueue("foo/wf-02/nodeid-456", 0, now.Add(time.Second), tx)

	acquired, _ := s.tryAcquire("foo/wf-01/nodeid-123", tx)
	assert.True(t, acquired)

	assert.Len(t, notified, 1)
	assert.True(t, notified["foo/wf-02"])
}

// TestNotifyWorkflowFromTemplateSemaphore runs the template semaphore test for both implementations
func TestNotifyWorkflowFromTemplateSemaphore(t *testing.T) {
	for name, factory := range semaphoreFactories {
		t.Run(name, func(t *testing.T) {
			testNotifyWorkflowFromTemplateSemaphore(t, factory)
		})
	}
}
