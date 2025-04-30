package sync

import (
	"context"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"

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

func setupMultipleLockManagers(t *testing.T, dbType sqldb.DBType, semaphoreSize int) (context.Context, *Manager, *Manager) {
	ctx := context.Background()
	kube := fake.NewSimpleClientset()
	// Create a database session for the semaphore
	info, deferfn, cfg, err := createTestDBSession(t, dbType)
	require.NoError(t, err)
	defer deferfn()

	// Set up the semaphore limit in the database
	dbKey := "default/my-db-semaphore"
	_, err = info.session.SQL().Exec("INSERT INTO sync_limit (name, sizelimit) VALUES (?, ?)", dbKey, semaphoreSize)
	require.NoError(t, err)

	// Create two sync managers with the same database session
	syncMgr1 := createLockManager(ctx, kube, "default", info.session, &cfg, func(string) (int, error) { return 2, nil }, func(key string) {}, WorkflowExistenceFunc)
	require.NotNil(t, syncMgr1)
	require.NotNil(t, syncMgr1.dbInfo.session)
	// Second controller
	cfg.ControllerName = "test2"
	syncMgr2 := createLockManager(ctx, kube, "default", info.session, &cfg, func(string) (int, error) { return 2, nil }, func(key string) {}, WorkflowExistenceFunc)
	require.NotNil(t, syncMgr2)
	require.NotNil(t, syncMgr2.dbInfo.session)
	return ctx, syncMgr1, syncMgr2
}

func testSyncManagersSemaphoreAcquisitionForDB(t *testing.T, dbType sqldb.DBType) {
	ctx, syncMgr1, syncMgr2 := setupMultipleLockManagers(t, dbType, 2)

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
	ctx, syncMgr1, syncMgr2 := setupMultipleLockManagers(t, dbType, 1)

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

	checkCanAcquire(ctx, t, syncMgr1, wf01)
	checkCannotAcquire(ctx, t, syncMgr2, wf02)
	checkCannotAcquire(ctx, t, syncMgr1, wf03)
	checkCannotAcquire(ctx, t, syncMgr2, wf04)
	// wf-01 releases lock
	syncMgr1.Release(ctx, wf01, wf01.Name, wf01.Spec.Synchronization)
	checkCannotAcquire(ctx, t, syncMgr1, wf03)
	checkCannotAcquire(ctx, t, syncMgr2, wf02)
	checkCanAcquire(ctx, t, syncMgr2, wf04)
	// recheck this time
	checkCannotAcquire(ctx, t, syncMgr1, wf03)
	checkCannotAcquire(ctx, t, syncMgr2, wf02)
	// wf-04 releases lock
	syncMgr2.Release(ctx, wf04, wf04.Name, wf04.Spec.Synchronization)
	checkCannotAcquire(ctx, t, syncMgr1, wf02)
	checkCanAcquire(ctx, t, syncMgr2, wf03)
	// wf-03 releases lock
	syncMgr2.Release(ctx, wf03, wf03.Name, wf03.Spec.Synchronization)
	checkCanAcquire(ctx, t, syncMgr1, wf02)
}

func TestSyncManagersSemaphoreAcquisitionWithPriority(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			testSyncManagersSemaphoreAcquisitionWithPriorityForDB(t, dbType)
		})
	}
}

// func testSyncManagersContendingForSemaphore(t *testing.T, dbType sqldb.DBType) {
// 	ctx, syncMgr1, syncMgr2 := setupMultipleLockManagers(t, dbType, 1)

// }

// func TestSyncManagersContendingForSemaphore(t *testing.T) {
// 	for _, dbType := range testDBTypes {
// 		t.Run(string(dbType), func(t *testing.T) {
// 			testSyncManagersContendingForSemaphore(t, dbType)
// 		})
// 	}
// }
