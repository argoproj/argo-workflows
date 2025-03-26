package sync

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upper/db/v4"
)

// TestInactiveControllerDBSemaphore tests that a semaphore can't be acquired if the controller is not marked as responding
func TestInactiveControllerDBSemaphore(t *testing.T) {
	// Only run this test for the database semaphore
	nextWorkflow := func(key string) {}

	// Create the database semaphore
	s, dbSession, _ := createTestDatabaseSemaphore(t, "bar", "foo", 1, 0, nextWorkflow)
	defer dbSession.Close()

	// Get the underlying database semaphore to access its session
	dbSemaphore, ok := s.(*databaseSemaphore)
	require.True(t, ok, "Expected a database semaphore")

	// Update the controller heartbeat to be older than the inactive controller timeout
	staleTime := time.Now().Add(-dbSemaphore.info.config.inactiveControllerTimeout * 2)
	_, err := dbSemaphore.info.session.SQL().Update(dbSemaphore.info.config.controllerTable).
		Set("time", staleTime).
		Where(db.Cond{"controller": dbSemaphore.info.config.controllerName}).
		Exec()
	require.NoError(t, err)

	// Add items to the queue
	now := time.Now()
	s.addToQueue("foo/wf-01", 0, now)
	s.addToQueue("foo/wf-02", 0, now.Add(time.Second))

	// Try to acquire - this should fail because the controller is considered inactive
	acquired, _ := s.tryAcquire("foo/wf-01")
	assert.False(t, acquired, "Semaphore should not be acquired when controller is marked as inactive")

	// Now update the controller heartbeat to be current
	_, err = dbSemaphore.info.session.SQL().Update(dbSemaphore.info.config.controllerTable).
		Set("time", time.Now()).
		Where(db.Cond{"controller": dbSemaphore.info.config.controllerName}).
		Exec()
	require.NoError(t, err)

	// Try again - now it should work
	acquired, _ = s.tryAcquire("foo/wf-01")
	assert.True(t, acquired, "Semaphore should be acquired when controller is alive")
}

// TestOtherControllerDBSemaphore tests semaphore behavior when items from other controllers are in the queue
func TestOtherControllerDBSemaphore(t *testing.T) {
	// Create a semaphore with limit 1
	nextWorkflow := func(key string) {}
	s, dbSession, _ := createTestDatabaseSemaphore(t, "bar", "foo", 1, 0, nextWorkflow)
	defer dbSession.Close()

	// Get the underlying database semaphore to access its session
	dbSemaphore, ok := s.(*databaseSemaphore)
	require.True(t, ok, "Expected a database semaphore")

	// Add an entry for another controller
	otherController := "otherController"
	_, err := dbSemaphore.info.session.SQL().InsertInto(dbSemaphore.info.config.controllerTable).
		Values(otherController, time.Now()).
		Exec()
	require.NoError(t, err)

	// Add an item to the queue from the other controller
	_, err = dbSemaphore.info.session.SQL().InsertInto(dbSemaphore.info.config.stateTable).
		Values(dbSemaphore.dbKey, "foo/other-wf-01", otherController, false, false, 0, time.Now()).
		Exec()
	require.NoError(t, err)

	// Add our own item to the queue
	now := time.Now()
	s.addToQueue("foo/our-wf-01", 0, now.Add(time.Second))

	// Try to acquire - this should fail because the other controller's item is first in line
	acquired, _ := s.tryAcquire("foo/our-wf-01")
	assert.False(t, acquired, "Semaphore should not be acquired when another controller's item is first in queue")

	// Now mark the other controller as inactive by setting its timestamp to be old
	staleTime := time.Now().Add(-dbSemaphore.info.config.inactiveControllerTimeout * 2)
	_, err = dbSemaphore.info.session.SQL().Update(dbSemaphore.info.config.controllerTable).
		Set("time", staleTime).
		Where(db.Cond{"controller": otherController}).
		Exec()
	require.NoError(t, err)

	// Try again - now it should work because the other controller is considered inactive
	acquired, _ = s.tryAcquire("foo/our-wf-01")
	assert.True(t, acquired, "Semaphore should be acquired when other controller is marked as inactive")

	// Verify the semaphore is now held by our workflow
	holders := dbSemaphore.getCurrentHolders()
	require.Len(t, holders, 1, "Should have one holder")
	assert.Equal(t, "foo/our-wf-01", holders[0], "Our workflow should be the holder")
}

// TestDifferentSemaphoreDBSemaphore tests that semaphores with different names don't block each other
func TestDifferentSemaphoreDBSemaphore(t *testing.T) {
	// Create a semaphore with limit 1
	nextWorkflow := func(key string) {}
	s, dbSession, _ := createTestDatabaseSemaphore(t, "bar", "foo", 1, 0, nextWorkflow)
	defer dbSession.Close()

	// Get the underlying database semaphore to access its session
	dbSemaphore, ok := s.(*databaseSemaphore)
	require.True(t, ok, "Expected a database semaphore")

	// Add an entry for another controller
	otherController := "otherController"
	_, err := dbSemaphore.info.session.SQL().InsertInto(dbSemaphore.info.config.controllerTable).
		Values(otherController, time.Now()).
		Exec()
	require.NoError(t, err)

	// Add an item to the queue from the other cluster with a DIFFERENT semaphore name
	_, err = dbSemaphore.info.session.SQL().InsertInto(dbSemaphore.info.config.stateTable).
		Values("different/semaphore", "foo/other-wf-01", otherController, false, false, 0, time.Now()).
		Exec()
	require.NoError(t, err)

	// Add our own item to the queue
	now := time.Now()
	s.addToQueue("foo/our-wf-01", 0, now.Add(time.Second))

	// Try to acquire - this should succeed because the other cluster's item is for a different semaphore
	acquired, _ := s.tryAcquire("foo/our-wf-01")
	assert.True(t, acquired, "Semaphore should be acquired when another cluster's item is for a different semaphore")

	// Verify the semaphore is now held by our workflow
	holders := dbSemaphore.getCurrentHolders()
	assert.Len(t, holders, 1, "Should have one holder")
	assert.Equal(t, "foo/our-wf-01", holders[0], "Our workflow should be the holder")
}

// TestMutexAndSemaphoreWithSameName tests that a mutex and semaphore with the same name don't interfere with each other
func TestMutexAndSemaphoreWithSameName(t *testing.T) {
	// Setup the same key name for both
	sharedKey := "foo/shared-name"

	nextWorkflow := func(key string) {}

	// Create a semaphore with limit 2 using the helper function
	semaphore, dbSession, info := createTestDatabaseSemaphore(t, "shared-name", "foo", 2, 0, nextWorkflow)
	defer dbSession.Close()

	// Create a mutex using that key
	mutex := newDatabaseMutex("foo/shared-name", sharedKey, nextWorkflow, info)

	// Add entries to queue and acquire for both
	now := time.Now()

	// Mutex workflow 1
	mutex.addToQueue("foo/wf-mutex-1", 0, now)
	mutexAcquired1, _ := mutex.tryAcquire("foo/wf-mutex-1")
	assert.True(t, mutexAcquired1, "Mutex should be acquired by first workflow")

	// Semaphore workflow 1
	semaphore.addToQueue("foo/wf-sem-1", 0, now)
	semAcquired1, _ := semaphore.tryAcquire("foo/wf-sem-1")
	assert.True(t, semAcquired1, "Semaphore should be acquired by first workflow")

	// Verify the mutex can't be acquired again
	mutex.addToQueue("foo/wf-mutex-2", 0, now)
	mutexAcquired2, _ := mutex.tryAcquire("foo/wf-mutex-2")
	assert.False(t, mutexAcquired2, "Mutex should not be acquired by second workflow")

	// But the semaphore can still be acquired (limit=2)
	semaphore.addToQueue("foo/wf-sem-2", 0, now)
	semAcquired2, _ := semaphore.tryAcquire("foo/wf-sem-2")
	assert.True(t, semAcquired2, "Semaphore should be acquired by second workflow")

	// But not a third time (because limit=2)
	semaphore.addToQueue("foo/wf-sem-3", 0, now)
	semAcquired3, _ := semaphore.tryAcquire("foo/wf-sem-3")
	assert.False(t, semAcquired3, "Semaphore should not be acquired by third workflow (at capacity)")

	// Now release the mutex
	mutex.release("foo/wf-mutex-1")

	// The mutex should be acquirable now
	mutexAcquired2Again, _ := mutex.tryAcquire("foo/wf-mutex-2")
	assert.True(t, mutexAcquired2Again, "Mutex should be acquired after release")

	// But this shouldn't affect the semaphore's capacity
	semAcquired3Again, _ := semaphore.tryAcquire("foo/wf-sem-3")
	assert.False(t, semAcquired3Again, "Semaphore should still be at capacity")

	// Now release one of the semaphore holders
	released := semaphore.release("foo/wf-sem-1")
	assert.True(t, released, "Semaphore should be released successfully")

	// Now we should be able to acquire the semaphore once
	semAcquired3Again, _ = semaphore.tryAcquire("foo/wf-sem-3")
	assert.True(t, semAcquired3Again, "Semaphore should be acquired after release")

	// But not a fourth time (still at capacity with 2 holders)
	semaphore.addToQueue("foo/wf-sem-4", 0, now)
	semAcquired4, _ := semaphore.tryAcquire("foo/wf-sem-4")
	assert.False(t, semAcquired4, "Semaphore should not be acquired fourth time (at capacity again)")

	// The mutex should still be held
	mutexAcquired3, _ := mutex.tryAcquire("foo/wf-mutex-3")
	assert.False(t, mutexAcquired3, "Mutex should still be held by another workflow")

	// Verify by checking the database directly
	var allHolders []stateRecord
	err := dbSession.SQL().
		Select("*").
		From(info.config.stateTable).
		Where(db.Cond{"name": sharedKey, "held": true}).
		All(&allHolders)
	require.NoError(t, err)
	assert.Len(t, allHolders, 3, "Should have three total holders (1 mutex + 2 semaphore)")

	// Check that we have the correct holders
	holderKeys := []string{}
	for _, holder := range allHolders {
		holderKeys = append(holderKeys, holder.Key)
	}
	assert.Contains(t, holderKeys, "foo/wf-mutex-2", "wf-mutex-2 should be a holder")
	assert.Contains(t, holderKeys, "foo/wf-sem-2", "wf-sem-2 should be a holder")
	assert.Contains(t, holderKeys, "foo/wf-sem-3", "wf-sem-3 should be a holder")
}

// TestSyncLimitCacheDB tests the caching of semaphore limit values in database semaphores
func TestSyncLimitCacheDB(t *testing.T) {
	// Keep track of the original nowFn and restore it after the test
	originalNowFn := nowFn
	defer func() {
		nowFn = originalNowFn
	}()

	// Mock time for consistent testing
	mockNow := time.Now()
	nowFn = func() time.Time {
		return mockNow
	}

	t.Run("RefreshesAfterTTL", func(t *testing.T) {
		nextWorkflow := func(key string) {}

		// Create a semaphore with initial limit of 5 and a 10 second TTL
		cacheTTL := 10 * time.Second
		s, dbSession, _ := createTestDatabaseSemaphore(t, "test-semaphore", "foo", 5, cacheTTL, nextWorkflow)
		defer dbSession.Close()

		// Cast to access internal fields
		dbSemaphore, ok := s.(*databaseSemaphore)
		require.True(t, ok, "Expected a database semaphore")

		// First call to getLimit() should return the initial limit
		initialLimit := dbSemaphore.getLimit()
		assert.Equal(t, 5, initialLimit, "Initial limit should be 5")

		// Get the initial timestamp
		initialTimestamp := dbSemaphore.getLimitTimestamp()

		// Call getLimit() again immediately - should use cached value and not update timestamp
		limit := dbSemaphore.getLimit()
		assert.Equal(t, 5, limit, "Limit should still be 5")
		assert.Equal(t, initialTimestamp, dbSemaphore.getLimitTimestamp(), "Timestamp should not change")

		// Update the semaphore limit in the database
		_, err := dbSemaphore.info.session.SQL().
			Update(dbSemaphore.info.config.limitTable).
			Set(limitSizeField, 10).
			Where(db.Cond{limitNameField: dbSemaphore.dbKey}).
			Exec()
		require.NoError(t, err)

		// Call getLimit() again - should still use cached value
		limit = dbSemaphore.getLimit()
		assert.Equal(t, 5, limit, "Limit should still be cached at 5")

		// Advance time just before TTL expires
		mockNow = mockNow.Add(cacheTTL - time.Second)

		// Call getLimit() again - should still use cached value
		limit = dbSemaphore.getLimit()
		assert.Equal(t, 5, limit, "Limit should still be cached at 5")

		// Advance time past TTL
		mockNow = mockNow.Add(2 * time.Second) // Now we're past the TTL

		// Call getLimit() again - should refresh from database
		limit = dbSemaphore.getLimit()
		assert.Equal(t, 10, limit, "Limit should be updated to 10")

		// Timestamp should be updated
		assert.NotEqual(t, initialTimestamp, dbSemaphore.getLimitTimestamp(), "Timestamp should be updated")
	})

	t.Run("ZeroTTLAlwaysRefreshes", func(t *testing.T) {
		nextWorkflow := func(key string) {}

		// Create a semaphore with initial limit of 5 and a 0 second TTL
		s, dbSession, _ := createTestDatabaseSemaphore(t, "test-semaphore-zero", "foo", 5, 0, nextWorkflow)
		defer dbSession.Close()

		// Cast to access internal fields
		dbSemaphore, ok := s.(*databaseSemaphore)
		require.True(t, ok, "Expected a database semaphore")

		// First call to getLimit() should return the initial limit
		initialLimit := dbSemaphore.getLimit()
		assert.Equal(t, 5, initialLimit, "Initial limit should be 5")

		// Get the initial timestamp
		initialTimestamp := dbSemaphore.getLimitTimestamp()

		// As we've a stopped clock we need to advance time to test the refresh
		mockNow = mockNow.Add(1 * time.Millisecond)

		// Update the semaphore limit in the database
		_, err := dbSemaphore.info.session.SQL().
			Update(dbSemaphore.info.config.limitTable).
			Set(limitSizeField, 7).
			Where(db.Cond{limitNameField: dbSemaphore.dbKey}).
			Exec()
		require.NoError(t, err)

		// Call getLimit() again - should immediately refresh with zero TTL
		limit := dbSemaphore.getLimit()
		assert.Equal(t, 7, limit, "Limit should be updated to 7")
		assert.NotEqual(t, initialTimestamp, dbSemaphore.getLimitTimestamp(), "Timestamp should be updated")
	})
}
