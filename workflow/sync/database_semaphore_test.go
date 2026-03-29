package sync

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
	syncdb "github.com/argoproj/argo-workflows/v4/util/sync/db"
)

var testDBTypes []sqldb.DBType

// Don't test databases on windows as testcontainers don't work there
func init() {
	switch runtime.GOOS {
	case "windows":
		// Can't test these on windows
		testDBTypes = []sqldb.DBType{}
	default:
		testDBTypes = []sqldb.DBType{sqldb.Postgres, sqldb.MySQL}
	}
}

// TestInactiveControllerDBSemaphore tests that a semaphore can't be acquired if the controller is not marked as responding
func TestInactiveControllerDBSemaphore(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			// Only run this test for the database semaphore
			nextWorkflow := func(key string) {}

			// Create the database semaphore
			s, info, deferfunc := createTestDatabaseSemaphore(ctx, t, "bar", "foo", 1, 0, nextWorkflow, dbType)
			defer deferfunc()

			// Update the controller heartbeat to be older than the inactive controller timeout
			staleTime := time.Now().Add(-info.Config.InactiveControllerTimeout * 2)
			_, err := info.Session.SQL().Update(info.Config.ControllerTable).
				Set("time", staleTime).
				Where(db.Cond{"controller": info.Config.ControllerName}).
				Exec()
			require.NoError(t, err)

			// Add items to the queue
			now := time.Now()
			require.NoError(t, s.addToQueue(ctx, "foo/wf-01", 0, now))
			require.NoError(t, s.addToQueue(ctx, "foo/wf-02", 0, now.Add(time.Second)))

			// Try to acquire - this should fail because the controller is considered inactive
			tx := &transaction{db: &info.Session}
			acquired, _ := s.tryAcquire(ctx, "foo/wf-01", tx)
			assert.False(t, acquired, "Semaphore should not be acquired when controller is marked as inactive")

			// Now update the controller heartbeat to be current
			_, err = info.Session.SQL().Update(info.Config.ControllerTable).
				Set("time", time.Now()).
				Where(db.Cond{"controller": info.Config.ControllerName}).
				Exec()
			require.NoError(t, err)

			// Try again - now it should work
			acquired, _ = s.tryAcquire(ctx, "foo/wf-01", tx)
			assert.True(t, acquired, "Semaphore should be acquired when controller is alive")
		})
	}
}

// TestOtherControllerDBSemaphore tests semaphore behavior when items from other controllers are in the queue
func TestOtherControllerDBSemaphore(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			// Create a semaphore with limit 1
			nextWorkflow := func(key string) {}
			s, info, deferfunc := createTestDatabaseSemaphore(ctx, t, "bar", "foo", 1, 0, nextWorkflow, dbType)
			defer deferfunc()

			// Add an entry for another controller
			otherController := "otherController"
			controllerRecord := &syncdb.ControllerHealthRecord{
				Controller: otherController,
				Time:       time.Now(),
			}
			_, err := info.Session.Collection(info.Config.ControllerTable).
				Insert(controllerRecord)
			require.NoError(t, err)

			// Add an item to the queue from the other controller
			semaphoreRecord := &syncdb.StateRecord{
				Name:       s.longDBKey(),
				Key:        "foo/other-wf-01",
				Controller: otherController,
				Held:       false,
				Time:       time.Now(),
			}
			_, err = info.Session.Collection(info.Config.StateTable).
				Insert(semaphoreRecord)
			require.NoError(t, err)

			// Add our own item to the queue
			now := time.Now()
			require.NoError(t, s.addToQueue(ctx, "foo/our-wf-01", 0, now.Add(time.Second)))

			// Try to acquire - this should fail because the other controller's item is first in line
			tx := &transaction{db: &info.Session}
			acquired, _ := s.tryAcquire(ctx, "foo/our-wf-01", tx)
			assert.False(t, acquired, "Semaphore should not be acquired when another controller's item is first in queue")

			// Now mark the other controller as inactive by setting its timestamp to be old
			staleTime := time.Now().Add(-info.Config.InactiveControllerTimeout * 2)
			_, err = info.Session.SQL().Update(info.Config.ControllerTable).
				Set("time", staleTime).
				Where(db.Cond{"controller": otherController}).
				Exec()
			require.NoError(t, err)

			// Try again - now it should work because the other controller is considered inactive
			acquired, _ = s.tryAcquire(ctx, "foo/our-wf-01", tx)
			assert.True(t, acquired, "Semaphore should be acquired when other controller is marked as inactive")

			// Verify the semaphore is now held by our workflow
			holders, err := s.getCurrentHolders(ctx)
			require.NoError(t, err)
			require.Len(t, holders, 1, "Should have one holder")
			assert.Equal(t, "foo/our-wf-01", holders[0], "Our workflow should be the holder")
		})
	}
}

// TestDifferentSemaphoreDBSemaphore tests that semaphores with different names don't block each other
func TestDifferentSemaphoreDBSemaphore(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			// Create a semaphore with limit 1
			nextWorkflow := func(key string) {}
			s, info, deferfunc := createTestDatabaseSemaphore(ctx, t, "bar", "foo", 1, 0, nextWorkflow, dbType)
			defer deferfunc()

			// Add an entry for another controller
			otherController := "otherController"
			controllerRecord := &syncdb.ControllerHealthRecord{
				Controller: otherController,
				Time:       time.Now(),
			}
			_, err := info.Session.Collection(info.Config.ControllerTable).
				Insert(controllerRecord)
			require.NoError(t, err)

			// Add an item to the queue from the other cluster with a DIFFERENT semaphore name
			semaphoreRecord := &syncdb.StateRecord{
				Name:       "sem/different/semaphore",
				Key:        "foo/other-wf-01",
				Controller: otherController,
				Held:       false,
				Time:       time.Now(),
			}
			_, err = info.Session.Collection(info.Config.StateTable).
				Insert(semaphoreRecord)
			require.NoError(t, err)

			// Add our own item to the queue
			now := time.Now()
			require.NoError(t, s.addToQueue(ctx, "foo/our-wf-01", 0, now.Add(time.Second)))

			// Try to acquire - this should succeed because the other cluster's item is for a different semaphore
			tx := &transaction{db: &info.Session}
			acquired, _ := s.tryAcquire(ctx, "foo/our-wf-01", tx)
			assert.True(t, acquired, "Semaphore should be acquired when another cluster's item is for a different semaphore")

			// Verify the semaphore is now held by our workflow
			holders, err := s.getCurrentHolders(ctx)
			require.NoError(t, err)
			assert.Len(t, holders, 1, "Should have one holder")
			assert.Equal(t, "foo/our-wf-01", holders[0], "Our workflow should be the holder")
		})
	}
}

// TestMutexAndSemaphoreWithSameName tests that a mutex and semaphore with the same name don't interfere with each other
func TestMutexAndSemaphoreWithSameName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			// Setup the same key name for both
			sharedKey := "foo/shared-name"

			nextWorkflow := func(key string) {}

			// Create a semaphore with limit 2 using the helper function
			semaphore, info, deferfunc := createTestDatabaseSemaphore(ctx, t, "shared-name", "foo", 2, 0, nextWorkflow, dbType)
			defer deferfunc()

			// Create a mutex using that key
			mutex := newDatabaseMutex("foo/shared-name", sharedKey, nextWorkflow, info)

			// Add entries to queue and acquire for both
			now := time.Now()

			// Mutex workflow 1
			tx := &transaction{db: &info.Session}
			require.NoError(t, mutex.addToQueue(ctx, "foo/wf-mutex-1", 0, now))
			mutexAcquired1, _ := mutex.tryAcquire(ctx, "foo/wf-mutex-1", tx)
			assert.True(t, mutexAcquired1, "Mutex should be acquired by first workflow")

			// Semaphore workflow 1
			require.NoError(t, semaphore.addToQueue(ctx, "foo/wf-sem-1", 0, now))
			semAcquired1, _ := semaphore.tryAcquire(ctx, "foo/wf-sem-1", tx)
			assert.True(t, semAcquired1, "Semaphore should be acquired by first workflow")

			// Verify the mutex can't be acquired again
			require.NoError(t, mutex.addToQueue(ctx, "foo/wf-mutex-2", 0, now))
			mutexAcquired2, _ := mutex.tryAcquire(ctx, "foo/wf-mutex-2", tx)
			assert.False(t, mutexAcquired2, "Mutex should not be acquired by second workflow")

			// But the semaphore can still be acquired (limit=2)
			require.NoError(t, semaphore.addToQueue(ctx, "foo/wf-sem-2", 0, now))
			semAcquired2, _ := semaphore.tryAcquire(ctx, "foo/wf-sem-2", tx)
			assert.True(t, semAcquired2, "Semaphore should be acquired by second workflow")

			// But not a third time (because limit=2)
			require.NoError(t, semaphore.addToQueue(ctx, "foo/wf-sem-3", 0, now))
			semAcquired3, _ := semaphore.tryAcquire(ctx, "foo/wf-sem-3", tx)
			assert.False(t, semAcquired3, "Semaphore should not be acquired by third workflow (at capacity)")

			// Now release the mutex
			mutex.release(ctx, "foo/wf-mutex-1")

			// The mutex should be acquirable now
			mutexAcquired2Again, _ := mutex.tryAcquire(ctx, "foo/wf-mutex-2", tx)
			assert.True(t, mutexAcquired2Again, "Mutex should be acquired after release")

			// But this shouldn't affect the semaphore's capacity
			semAcquired3Again, _ := semaphore.tryAcquire(ctx, "foo/wf-sem-3", tx)
			assert.False(t, semAcquired3Again, "Semaphore should still be at capacity")

			// Now release one of the semaphore holders
			released := semaphore.release(ctx, "foo/wf-sem-1")
			assert.True(t, released, "Semaphore should be released successfully")

			// Now we should be able to acquire the semaphore once
			semAcquired3Again, _ = semaphore.tryAcquire(ctx, "foo/wf-sem-3", tx)
			assert.True(t, semAcquired3Again, "Semaphore should be acquired after release")

			// But not a fourth time (still at capacity with 2 holders)
			require.NoError(t, semaphore.addToQueue(ctx, "foo/wf-sem-4", 0, now))
			semAcquired4, _ := semaphore.tryAcquire(ctx, "foo/wf-sem-4", tx)
			assert.False(t, semAcquired4, "Semaphore should not be acquired fourth time (at capacity again)")

			// The mutex should still be held
			mutexAcquired3, _ := mutex.tryAcquire(ctx, "foo/wf-mutex-3", tx)
			assert.False(t, mutexAcquired3, "Mutex should still be held by another workflow")

			// Verify by checking the database directly
			var allHolders []syncdb.StateRecord
			err := info.Session.SQL().
				Select("*").
				From(info.Config.StateTable).
				Where(db.Cond{"held": true}).
				All(&allHolders)
			require.NoError(t, err)
			assert.Len(t, allHolders, 3, "Should have three total holders (1 mutex + 2 semaphore)")

			// Check that we have the correct holders
			holderKeys := []string{}
			for _, holder := range allHolders {
				holderKeys = append(holderKeys, holder.Key)
			}
			t.Logf("holderKeys: %v", holderKeys)
			assert.Contains(t, holderKeys, "foo/wf-mutex-2", "wf-mutex-2 should be a holder")
			assert.Contains(t, holderKeys, "foo/wf-sem-2", "wf-sem-2 should be a holder")
			assert.Contains(t, holderKeys, "foo/wf-sem-3", "wf-sem-3 should be a holder")
		})
	}
}

// TestSyncLimitCacheDB tests the caching of semaphore limit values in database semaphores
func TestSyncLimitCacheDB(t *testing.T) {
	ctx := logging.TestContext(t.Context())

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

	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			t.Run("RefreshesAfterTTL", func(t *testing.T) {
				nextWorkflow := func(key string) {}

				// Create a semaphore with initial limit of 5 and a 10 second TTL
				cacheTTL := 10 * time.Second
				s, info, deferfunc := createTestDatabaseSemaphore(ctx, t, "test-semaphore", "foo", 5, cacheTTL, nextWorkflow, dbType)
				defer deferfunc()

				// First call to getLimit() should return the initial limit
				initialLimit := s.getLimit(ctx)
				assert.Equal(t, 5, initialLimit, "Initial limit should be 5")

				// Call getLimit() again immediately - should use cached value and not update timestamp
				limit := s.getLimit(ctx)
				assert.Equal(t, 5, limit, "Limit should still be 5")

				// Update the semaphore limit in the database
				_, err := info.Session.SQL().
					Update(info.Config.LimitTable).
					Set(syncdb.LimitSizeField, 10).
					Where(db.Cond{syncdb.LimitNameField: s.shortDBKey}).
					Exec()
				require.NoError(t, err)

				// Call getLimit() again - should still use cached value
				limit = s.getLimit(ctx)
				assert.Equal(t, 5, limit, "Limit should still be cached at 5")

				// Advance time just before TTL expires
				mockNow = mockNow.Add(cacheTTL - time.Second)

				// Call getLimit() again - should still use cached value
				limit = s.getLimit(ctx)
				assert.Equal(t, 5, limit, "Limit should still be cached at 5")

				// Advance time past TTL
				mockNow = mockNow.Add(2 * time.Second) // Now we're past the TTL

				// Call getLimit() again - should refresh from database
				limit = s.getLimit(ctx)
				assert.Equal(t, 10, limit, "Limit should be updated to 10")
			})

			t.Run("ZeroTTLAlwaysRefreshes", func(t *testing.T) {
				nextWorkflow := func(key string) {}

				// Create a semaphore with initial limit of 5 and a 0 second TTL
				s, info, deferfunc := createTestDatabaseSemaphore(ctx, t, "test-semaphore-zero", "foo", 5, 0, nextWorkflow, dbType)
				defer deferfunc()

				// First call to getLimit() should return the initial limit
				initialLimit := s.getLimit(ctx)
				assert.Equal(t, 5, initialLimit, "Initial limit should be 5")

				// As we've a stopped clock we need to advance time to test the refresh
				mockNow = mockNow.Add(1 * time.Millisecond)

				// Update the semaphore limit in the database
				_, err := info.Session.SQL().
					Update(info.Config.LimitTable).
					Set(syncdb.LimitSizeField, 7).
					Where(db.Cond{syncdb.LimitNameField: s.shortDBKey}).
					Exec()
				require.NoError(t, err)

				// Call getLimit() again - should immediately refresh with zero TTL
				limit := s.getLimit(ctx)
				assert.Equal(t, 7, limit, "Limit should be updated to 7")
			})
		})
	}
}
