package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestDatabaseMutex creates a database-backed mutex for testing
func createTestDatabaseMutex(t *testing.T, name, namespace string, nextWorkflow NextWorkflow) (*databaseSemaphore, func()) {
	t.Helper()
	dbSession, info, err := createTestDBSession(t)
	require.NoError(t, err)

	dbKey := namespace + "/" + name

	// Create a mutex (which is a semaphore with limit=1)
	mutex := newDatabaseMutex(name, dbKey, nextWorkflow, info)
	require.NotNil(t, mutex)

	return mutex, func() {
		dbSession.Close()
	}
}

// TestDatabaseMutexAcquireRelease tests basic acquire and release functionality
func TestDatabaseMutexAcquireRelease(t *testing.T) {
	nextWorkflow := func(key string) {}

	mutex, cleanup := createTestDatabaseMutex(t, "test-mutex", "default", nextWorkflow)
	defer cleanup()

	now := time.Now()
	mutex.addToQueue("default/workflow1", 0, now)
	mutex.addToQueue("default/workflow2", 0, now.Add(time.Second))

	// First acquisition should succeed
	acquired, _ := mutex.tryAcquire("default/workflow1")
	assert.True(t, acquired, "First acquisition should succeed")

	// Second acquisition should fail
	acquired, _ = mutex.tryAcquire("default/workflow2")
	assert.False(t, acquired, "Second acquisition should fail")

	// Release the mutex
	mutex.release("default/workflow1")

	// Now acquisition should succeed again
	acquired, _ = mutex.tryAcquire("default/workflow2")
	assert.True(t, acquired, "Acquisition after release should succeed")
}

// TestDatabaseMutexQueueOrder tests that workflows are processed in order
func TestDatabaseMutexQueueOrder(t *testing.T) {
	notified := make(map[string]bool)
	nextWorkflow := func(key string) {
		notified[key] = true
	}

	mutex, cleanup := createTestDatabaseMutex(t, "test-mutex", "default", nextWorkflow)
	defer cleanup()

	// Add items to the queue
	now := time.Now()
	mutex.addToQueue("default/workflow1", 0, now)
	mutex.addToQueue("default/workflow2", 0, now.Add(time.Second))

	acquired, _ := mutex.tryAcquire("default/workflow2")
	assert.False(t, acquired, "Second workflow should not acquire the mutex")

	// Acquire the first one
	acquired, _ = mutex.tryAcquire("default/workflow1")
	assert.True(t, acquired, "First workflow should acquire the mutex")

	// Release it - this should notify the next one
	mutex.release("default/workflow1")

	// Check that workflow2 was notified
	assert.True(t, notified["default/workflow2"], "workflow2 should be notified")
}
