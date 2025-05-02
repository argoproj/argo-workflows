package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

// createTestDatabaseMutex creates a database-backed mutex for testing
func createTestDatabaseMutex(t *testing.T, name, namespace string, nextWorkflow NextWorkflow, dbType sqldb.DBType) (*databaseSemaphore, *transaction, func()) {
	t.Helper()
	info, deferfunc, _, err := createTestDBSession(t, dbType)
	require.NoError(t, err)

	dbKey := namespace + "/" + name

	// Create a mutex (which is a semaphore with limit=1)
	mutex := newDatabaseMutex(name, dbKey, nextWorkflow, info)
	require.NotNil(t, mutex)
	tx := &transaction{db: &info.session}
	return mutex, tx, deferfunc
}

// TestDatabaseMutexAcquireRelease tests basic acquire and release functionality
func TestDatabaseMutexAcquireRelease(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			nextWorkflow := func(key string) {}

			mutex, tx, deferfunc := createTestDatabaseMutex(t, "test-mutex", "default", nextWorkflow, dbType)
			defer deferfunc()

			now := time.Now()
			mutex.addToQueue("default/workflow1", 0, now)
			mutex.addToQueue("default/workflow2", 0, now.Add(time.Second))

			// First acquisition should succeed
			acquired, _ := mutex.tryAcquire("default/workflow1", tx)
			assert.True(t, acquired, "First acquisition should succeed")

			// Second acquisition should fail
			acquired, _ = mutex.tryAcquire("default/workflow2", tx)
			assert.False(t, acquired, "Second acquisition should fail")

			// Release the mutex
			mutex.release("default/workflow1")

			// Now acquisition should succeed again
			acquired, _ = mutex.tryAcquire("default/workflow2", tx)
			assert.True(t, acquired, "Acquisition after release should succeed")
		})
	}
}

// TestDatabaseMutexQueueOrder tests that workflows are processed in order
func TestDatabaseMutexQueueOrder(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			notified := make(map[string]bool)
			nextWorkflow := func(key string) {
				notified[key] = true
			}

			mutex, tx, deferfunc := createTestDatabaseMutex(t, "test-mutex", "default", nextWorkflow, dbType)
			defer deferfunc()

			// Add items to the queue
			now := time.Now()
			mutex.addToQueue("default/workflow1", 0, now)
			mutex.addToQueue("default/workflow2", 0, now.Add(time.Second))

			acquired, _ := mutex.tryAcquire("default/workflow2", tx)
			assert.False(t, acquired, "Second workflow should not acquire the mutex")

			// Acquire the first one
			acquired, _ = mutex.tryAcquire("default/workflow1", tx)
			assert.True(t, acquired, "First workflow should acquire the mutex")

			// Release it - this should notify the next one
			mutex.release("default/workflow1")

			// Check that workflow2 was notified
			assert.True(t, notified["default/workflow2"], "workflow2 should be notified")
		})
	}
}
