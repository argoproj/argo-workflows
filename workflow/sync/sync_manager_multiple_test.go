//go:build !windows

package sync

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

const wfWithDatabaseSemaphore = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-db-semaphore
  namespace: default
spec:
  entrypoint: whalesay
  synchronization:
    semaphores:
      - database:
          key: my-db-semaphore
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

func checkCanAcquire(ctx context.Context, t *testing.T, syncMgr *Manager, wf *wfv1.Workflow) {
	status, _, _, _, err := syncMgr.TryAcquire(ctx, wf, wf.Name, wf.Spec.Synchronization)
	require.NoError(t, err)
	assert.True(t, status, "wf should acquire lock")
}

func checkCannotAcquire(ctx context.Context, t *testing.T, syncMgr *Manager, wf *wfv1.Workflow) {
	status, _, msg, _, err := syncMgr.TryAcquire(ctx, wf, wf.Name, wf.Spec.Synchronization)
	require.NoError(t, err)
	assert.False(t, status, "wf should not acquire lock")
	assert.Contains(t, msg, "Waiting for", "wf should be waiting")
}

func setupMultipleLockManagers(t *testing.T, dbType sqldb.DBType, semaphoreSize int) (context.Context, func(), *Manager, *Manager) {
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	// Create a database session for the semaphore
	info, deferfn, cfg, err := createTestDBSession(ctx, t, dbType)
	deferfn2 := func() {
		deferfn()
		cancel()
	}
	require.NoError(t, err)

	// Set up the semaphore limit in the database
	dbKey := "default/my-db-semaphore"
	_, err = info.Session.SQL().Exec("INSERT INTO sync_limit (name, sizelimit) VALUES (?, ?)", dbKey, semaphoreSize)
	require.NoError(t, err)

	// Create two sync managers with the same database session
	syncMgr1 := createLockManager(ctx, info.Session, &cfg, func(_ context.Context, _ string) (int, error) { return 2, nil }, func(key string) {}, WorkflowExistenceFunc)
	require.NotNil(t, syncMgr1)
	require.NotNil(t, syncMgr1.dbInfo.Session)
	// Second controller
	cfg.ControllerName = "test2"
	syncMgr2 := createLockManager(ctx, info.Session, &cfg, func(_ context.Context, _ string) (int, error) { return 2, nil }, func(key string) {}, WorkflowExistenceFunc)
	require.NotNil(t, syncMgr2)
	require.NotNil(t, syncMgr2.dbInfo.Session)
	return ctx, deferfn2, syncMgr1, syncMgr2
}

func testSyncManagersSemaphoreAcquisitionForDB(t *testing.T, dbType sqldb.DBType) {
	ctx, deferfn, syncMgr1, syncMgr2 := setupMultipleLockManagers(t, dbType, 2)
	defer deferfn()

	// Create 4 workflows
	wf01 := wfv1.MustUnmarshalWorkflow(wfWithDatabaseSemaphore)
	wf01.CreationTimestamp = metav1.Time{Time: time.Now().Add(-4 * time.Second)}
	wf01.Name = "wf-01"
	wf02 := wf01.DeepCopy()
	wf02.CreationTimestamp = metav1.Time{Time: time.Now().Add(-3 * time.Second)}
	wf02.Name = "wf-02"
	wf03 := wf01.DeepCopy()
	wf03.CreationTimestamp = metav1.Time{Time: time.Now().Add(-2 * time.Second)}
	wf03.Name = "wf-03"
	wf04 := wf01.DeepCopy()
	wf04.CreationTimestamp = metav1.Time{Time: time.Now().Add(-1 * time.Second)}
	wf04.Name = "wf-04"

	checkCanAcquire(ctx, t, syncMgr1, wf01)
	checkCanAcquire(ctx, t, syncMgr2, wf02)
	checkCannotAcquire(ctx, t, syncMgr1, wf03)
	checkCannotAcquire(ctx, t, syncMgr2, wf04)

	// wf-01 releases lock
	syncMgr1.Release(ctx, wf01, wf01.Name, wf01.Spec.Synchronization)
	checkCannotAcquire(ctx, t, syncMgr2, wf04)
	checkCanAcquire(ctx, t, syncMgr1, wf03)
	// wf-03 releases lock
	syncMgr1.Release(ctx, wf03, wf03.Name, wf03.Spec.Synchronization)
	checkCanAcquire(ctx, t, syncMgr2, wf04)
}

func TestSyncManagersSemaphoreAcquisition(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			testSyncManagersSemaphoreAcquisitionForDB(t, dbType)
		})
	}
}

func testSyncManagersSemaphoreAcquisitionWithPriorityForDB(t *testing.T, dbType sqldb.DBType) {
	ctx, deferfn, syncMgr1, syncMgr2 := setupMultipleLockManagers(t, dbType, 1)
	defer deferfn()

	// Create 4 workflows
	wf01 := wfv1.MustUnmarshalWorkflow(wfWithDatabaseSemaphore)
	wf01.CreationTimestamp = metav1.Time{Time: time.Now().Add(-4 * time.Second)}
	wf01.Spec.Priority = ptr.To(int32(1))
	wf01.Name = "wf-01"
	wf02 := wf01.DeepCopy()
	wf02.CreationTimestamp = metav1.Time{Time: time.Now().Add(-3 * time.Second)}
	wf02.Name = "wf-02"
	wf02.Spec.Priority = ptr.To(int32(2))
	wf03 := wf01.DeepCopy()
	wf03.CreationTimestamp = metav1.Time{Time: time.Now().Add(-2 * time.Second)}
	wf03.Name = "wf-03"
	wf03.Spec.Priority = ptr.To(int32(3))
	wf04 := wf01.DeepCopy()
	wf04.CreationTimestamp = metav1.Time{Time: time.Now().Add(-1 * time.Second)}
	wf04.Name = "wf-04"
	wf04.Spec.Priority = ptr.To(int32(4))

	// wf-01 acquires lock as first to appear
	checkCanAcquire(ctx, t, syncMgr1, wf01)
	checkCannotAcquire(ctx, t, syncMgr2, wf02)
	checkCannotAcquire(ctx, t, syncMgr1, wf03)
	checkCannotAcquire(ctx, t, syncMgr2, wf04)
	// wf-01 releases lock
	syncMgr1.Release(ctx, wf01, wf01.Name, wf01.Spec.Synchronization)
	// wf-04 has highest priority, so should be next
	checkCannotAcquire(ctx, t, syncMgr1, wf03)
	checkCannotAcquire(ctx, t, syncMgr2, wf02)
	checkCanAcquire(ctx, t, syncMgr2, wf04)
	// recheck the others
	checkCannotAcquire(ctx, t, syncMgr1, wf03)
	checkCannotAcquire(ctx, t, syncMgr2, wf02)
	// wf-04 releases lock
	syncMgr2.Release(ctx, wf04, wf04.Name, wf04.Spec.Synchronization)
	checkCannotAcquire(ctx, t, syncMgr2, wf02)
	checkCanAcquire(ctx, t, syncMgr1, wf03)
	// wf-03 releases lock
	syncMgr1.Release(ctx, wf03, wf03.Name, wf03.Spec.Synchronization)
	checkCanAcquire(ctx, t, syncMgr2, wf02)
}

func TestSyncManagersSemaphoreAcquisitionWithPriority(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			testSyncManagersSemaphoreAcquisitionWithPriorityForDB(t, dbType)
		})
	}
}

func testSyncManagersContendingForSemaphore(t *testing.T, dbType sqldb.DBType) {
	ctx, deferfn, syncMgr1, syncMgr2 := setupMultipleLockManagers(t, dbType, 1)
	defer deferfn()

	// Create 4 workflows
	wfbase := wfv1.MustUnmarshalWorkflow(wfWithDatabaseSemaphore)
	var wg sync.WaitGroup
	lockCount := 0
	maxLockCount := 0
	testMtx := sync.Mutex{}

	// Function to run workflows for a sync manager
	runWorkflows := func(sm *Manager, name string, count int) {
		for testCounter := range count {
			wfCopy := wfbase.DeepCopy()
			wfName := fmt.Sprintf("%s-%d", name, testCounter)
			t.Log(wfName)
			wfCopy.Name = wfName
			wfCopy.CreationTimestamp = metav1.Time{Time: time.Now()}
			// Try to acquire lock
			var acquired bool
			for !acquired {
				var err error
				acquired, _, _, _, err = sm.TryAcquire(ctx, wfCopy, wfCopy.Name, wfCopy.Spec.Synchronization)
				if err != nil {
					t.Errorf("Error acquiring lock: %v", err)
					return
				}

				if acquired {
					testMtx.Lock()
					lockCount++
					t.Log(lockCount)
					if lockCount >= maxLockCount {
						maxLockCount = lockCount
					}
					testMtx.Unlock()

					// Simulate work
					time.Sleep(time.Millisecond * 10)

					// Release lock with a mutex to ensure we won't have lockCount at +1 after release
					testMtx.Lock()
					sm.Release(ctx, wfCopy, wfCopy.Name, wfCopy.Spec.Synchronization)
					lockCount--
					testMtx.Unlock()
				}
			}
		}
	}

	const iterationCount = 5
	// Start two goroutines
	wg.Go(func() { runWorkflows(syncMgr1, "wf1", iterationCount) })
	wg.Go(func() { runWorkflows(syncMgr2, "wf2", iterationCount) })
	wg.Wait()

	// Verify that at no point were multiple locks held
	if maxLockCount > 1 {
		t.Errorf("Multiple locks were held simultaneously: %d", maxLockCount)
	}

}

func TestSyncManagersContendingForSemaphore(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			testSyncManagersContendingForSemaphore(t, dbType)
		})
	}
}
