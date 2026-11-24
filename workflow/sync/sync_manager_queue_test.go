package sync

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
	syncdb "github.com/argoproj/argo-workflows/v4/util/sync/db"
	syncdbmocks "github.com/argoproj/argo-workflows/v4/util/sync/db/mocks"
)

// These tests cover https://github.com/argoproj/argo-workflows/issues/17013:
// TryAcquire queues a holder on every lock it asks for, so a holder that will
// never acquire its locks must be removed from all of those queues, or its
// stale entry blocks every later holder of a free lock.

const (
	queueTestSemaphoreKey = "default/ConfigMap/my-config/template"
	queueTestMutexKey     = "default/Mutex/my-mutex"
)

var (
	queueTestSemaphore = &wfv1.SemaphoreRef{ConfigMapKeyRef: &v1.ConfigMapKeySelector{
		LocalObjectReference: v1.LocalObjectReference{Name: "my-config"},
		Key:                  "template",
	}}
	queueTestMutex = &wfv1.Mutex{Name: "my-mutex"}
	// Both locks: the semaphore first, so that it is the lock a waiter records.
	queueTestBoth = &wfv1.Synchronization{
		Semaphores: []*wfv1.SemaphoreRef{queueTestSemaphore},
		Mutexes:    []*wfv1.Mutex{queueTestMutex},
	}
	queueTestCreated = time.Now().Add(-time.Hour)
)

// newQueueTestManager returns a lock manager whose ConfigMap semaphore has a
// limit of 1, and the workflow keys it has notified, in order.
func newQueueTestManager(ctx context.Context, t *testing.T) (*Manager, *[]string) {
	t.Helper()
	kube := fake.NewClientset()
	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "my-config", Namespace: "default"},
		Data:       map[string]string{"template": "1"},
	}
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)
	var notified []string
	syncManager, err := NewLockManager(ctx, kube, "", nil, GetSyncLimitFunc(kube), func(key string) {
		notified = append(notified, key)
	}, WorkflowExistenceFunc, false)
	require.NoError(t, err)
	return syncManager, &notified
}

func newQueueTestWorkflow(name string, created time.Time) *wfv1.Workflow {
	return &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{
		Name:              name,
		Namespace:         "default",
		CreationTimestamp: metav1.Time{Time: created},
	}}
}

// queueVictim fills the semaphore with another workflow, then queues node
// victim-1 of workflow victim on both the semaphore and the free mutex.
func queueVictim(ctx context.Context, t *testing.T, syncManager *Manager) *wfv1.Workflow {
	t.Helper()
	acquired, _, _, _, err := syncManager.TryAcquire(ctx, newQueueTestWorkflow("holder", queueTestCreated), "holder-1", &wfv1.Synchronization{
		Semaphores: []*wfv1.SemaphoreRef{queueTestSemaphore},
	})
	require.NoError(t, err)
	require.True(t, acquired)

	victimWf := newQueueTestWorkflow("victim", queueTestCreated)
	acquired, _, _, failedLockName, err := syncManager.TryAcquire(ctx, victimWf, "victim-1", queueTestBoth)
	require.NoError(t, err)
	require.False(t, acquired)
	require.Equal(t, queueTestSemaphoreKey, failedLockName)
	return victimWf
}

// tryAcquireMutex tries to acquire the mutex for node <name>-1 of wf.
func tryAcquireMutex(ctx context.Context, t *testing.T, syncManager *Manager, wf *wfv1.Workflow) (bool, string) {
	t.Helper()
	acquired, _, msg, _, err := syncManager.TryAcquire(ctx, wf, wf.Name+"-1", &wfv1.Synchronization{
		Mutexes: []*wfv1.Mutex{queueTestMutex},
	})
	require.NoError(t, err)
	return acquired, msg
}

func pendingOn(ctx context.Context, t *testing.T, syncManager *Manager, lockKey string) []string {
	t.Helper()
	pending, err := syncManager.syncLockMap[lockKey].getCurrentPending(ctx)
	require.NoError(t, err)
	return pending
}

// A node that will never acquire its locks is removed from every queue it
// joined, not only the one it records, and the next waiter is notified.
func TestDequeueRemovesNodeFromEveryWaitQueue(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, notified := newQueueTestManager(ctx, t)
	victimWf := queueVictim(ctx, t, syncManager)

	// The other workflow waits behind the victim for the free mutex.
	otherWf := newQueueTestWorkflow("other", queueTestCreated.Add(time.Minute))
	acquired, msg := tryAcquireMutex(ctx, t, syncManager, otherWf)
	require.False(t, acquired)
	require.Contains(t, msg, "Lock status: 1/1")

	*notified = nil
	syncManager.Dequeue(ctx, victimWf, "victim-1")

	assert.NotContains(t, pendingOn(ctx, t, syncManager, queueTestSemaphoreKey), "default/victim/victim-1")
	assert.NotContains(t, pendingOn(ctx, t, syncManager, queueTestMutexKey), "default/victim/victim-1")
	assert.Contains(t, *notified, "default/other", "the next waiter for the mutex must be requeued")
	acquired, msg = tryAcquireMutex(ctx, t, syncManager, otherWf)
	assert.True(t, acquired, "got %q", msg)
}

// Dequeue only leaves queues: a node that acquired its locks keeps them.
func TestDequeueKeepsAcquiredLocks(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, _ := newQueueTestManager(ctx, t)

	wf := newQueueTestWorkflow("running", queueTestCreated)
	acquired, _, _, _, err := syncManager.TryAcquire(ctx, wf, "running-1", queueTestBoth)
	require.NoError(t, err)
	require.True(t, acquired)

	syncManager.Dequeue(ctx, wf, "running-1")

	for _, lockKey := range []string{queueTestSemaphoreKey, queueTestMutexKey} {
		holders, err := syncManager.syncLockMap[lockKey].getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"default/running/running-1"}, holders, lockKey)
	}
	acquired, _ = tryAcquireMutex(ctx, t, syncManager, newQueueTestWorkflow("other", queueTestCreated.Add(time.Minute)))
	assert.False(t, acquired, "the mutex is still held")
}

// Removing a holder from its queues more than once, by Dequeue and then by
// ReleaseAll, is safe.
func TestDequeueIsIdempotent(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, _ := newQueueTestManager(ctx, t)
	victimWf := queueVictim(ctx, t, syncManager)

	syncManager.Dequeue(ctx, victimWf, "victim-1")
	syncManager.Dequeue(ctx, victimWf, "victim-1")
	syncManager.ReleaseAll(ctx, victimWf)
	syncManager.ReleaseAll(ctx, victimWf)

	assert.Empty(t, pendingOn(ctx, t, syncManager, queueTestMutexKey))
	assert.Empty(t, pendingOn(ctx, t, syncManager, queueTestSemaphoreKey))
	acquired, msg := tryAcquireMutex(ctx, t, syncManager, newQueueTestWorkflow("other", queueTestCreated.Add(time.Minute)))
	assert.True(t, acquired, "got %q", msg)
}

// A node that errored while waiting for its locks records only the first lock
// it failed to acquire in SynchronizationStatus.Waiting. ReleaseAll removed it
// only from that one queue, and its stale entry at the front of the others
// kept every other workflow from a free lock (#17013).
func TestReleaseAllRemovesNodeFromEveryWaitQueue(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, _ := newQueueTestManager(ctx, t)
	victimWf := queueVictim(ctx, t, syncManager)

	// As the controller does (markNodeWaitingForLock), the node records only
	// the lock that failed, then errors, and the workflow completes.
	victimWf.Status.Nodes = wfv1.Nodes{"victim-1": wfv1.NodeStatus{
		ID:                    "victim-1",
		Name:                  victimWf.Name,
		Phase:                 wfv1.NodeError,
		SynchronizationStatus: &wfv1.NodeSynchronizationStatus{Waiting: queueTestSemaphoreKey},
	}}
	victimWf.Status.Phase = wfv1.WorkflowError
	syncManager.ReleaseAll(ctx, victimWf)

	acquired, msg := tryAcquireMutex(ctx, t, syncManager, newQueueTestWorkflow("other", queueTestCreated.Add(time.Minute)))
	assert.True(t, acquired, "the free mutex must be acquirable once the errored node's workflow released its locks, got %q", msg)
}

// A workflow waiting for its workflow-level locks is removed from every queue
// it joined when it is released, not only from those its status records: a
// free mutex has no Waiting entry in the workflow's status.
func TestReleaseAllRemovesWorkflowFromEveryWaitQueue(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, _ := newQueueTestManager(ctx, t)

	acquired, _, _, _, err := syncManager.TryAcquire(ctx, newQueueTestWorkflow("holder", queueTestCreated), "", &wfv1.Synchronization{
		Semaphores: []*wfv1.SemaphoreRef{queueTestSemaphore},
	})
	require.NoError(t, err)
	require.True(t, acquired)
	victimWf := newQueueTestWorkflow("victim", queueTestCreated)
	acquired, _, _, _, err = syncManager.TryAcquire(ctx, victimWf, "", queueTestBoth)
	require.NoError(t, err)
	require.False(t, acquired)

	syncManager.ReleaseAll(ctx, victimWf)

	assert.Empty(t, pendingOn(ctx, t, syncManager, queueTestMutexKey))
	acquired, _, msg, _, err := syncManager.TryAcquire(ctx, newQueueTestWorkflow("other", queueTestCreated.Add(time.Minute)), "", &wfv1.Synchronization{
		Mutexes: []*wfv1.Mutex{queueTestMutex},
	})
	require.NoError(t, err)
	assert.True(t, acquired, "got %q", msg)
}

// When TryAcquire fails after it has queued the holder on some of its locks,
// that holder is still removed from those queues.
func TestDequeueAfterPartiallyQueuedTryAcquire(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, _ := newQueueTestManager(ctx, t)

	// The free semaphore is queued first; the semaphore that follows cannot be
	// built, because its ConfigMap does not exist.
	victimWf := newQueueTestWorkflow("victim", queueTestCreated)
	_, _, _, _, err := syncManager.TryAcquire(ctx, victimWf, "victim-1", &wfv1.Synchronization{
		Semaphores: []*wfv1.SemaphoreRef{queueTestSemaphore, {ConfigMapKeyRef: &v1.ConfigMapKeySelector{
			LocalObjectReference: v1.LocalObjectReference{Name: "missing"},
			Key:                  "template",
		}}},
	})
	require.Error(t, err)
	require.Equal(t, []string{"default/victim/victim-1"}, pendingOn(ctx, t, syncManager, queueTestSemaphoreKey))

	syncManager.Dequeue(ctx, victimWf, "victim-1")

	assert.Empty(t, pendingOn(ctx, t, syncManager, queueTestSemaphoreKey))
	acquired, _, msg, _, err := syncManager.TryAcquire(ctx, newQueueTestWorkflow("other", queueTestCreated.Add(time.Minute)), "other-1", &wfv1.Synchronization{
		Semaphores: []*wfv1.SemaphoreRef{queueTestSemaphore},
	})
	require.NoError(t, err)
	assert.True(t, acquired, "got %q", msg)
}

// newQueueVictims queues nodes victim-1 to victim-<n> of workflow victim on
// the semaphore, which another workflow holds, and on the free mutex, then
// queues workflow other on the mutex behind them.
func newQueueVictims(ctx context.Context, t *testing.T, syncManager *Manager, n int) *wfv1.Workflow {
	t.Helper()
	acquired, _, _, _, err := syncManager.TryAcquire(ctx, newQueueTestWorkflow("holder", queueTestCreated), "holder-1", &wfv1.Synchronization{
		Semaphores: []*wfv1.SemaphoreRef{queueTestSemaphore},
	})
	require.NoError(t, err)
	require.True(t, acquired)
	victimWf := newQueueTestWorkflow("victim", queueTestCreated)
	victimWf.Status.Synchronization = &wfv1.SynchronizationStatus{}
	for i := 1; i <= n; i++ {
		acquired, _, _, _, err = syncManager.TryAcquire(ctx, victimWf, fmt.Sprintf("victim-%d", i), queueTestBoth)
		require.NoError(t, err)
		require.False(t, acquired)
	}
	acquired, _ = tryAcquireMutex(ctx, t, syncManager, newQueueTestWorkflow("other", queueTestCreated.Add(time.Minute)))
	require.False(t, acquired)
	return victimWf
}

// ReleaseAll notifies the waiters of each lock once, after it has removed
// every node of the workflow from that lock's queue: a notification sent
// earlier goes to a node that is about to leave the queue.
func TestReleaseAllNotifiesEachLockOnceAfterLeavingItsQueue(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, notified := newQueueTestManager(ctx, t)
	victimWf := newQueueVictims(ctx, t, syncManager, 5)

	*notified = nil
	syncManager.ReleaseAll(ctx, victimWf)

	assert.Equal(t, []string{"default/other"}, *notified)
	assert.Empty(t, pendingOn(ctx, t, syncManager, queueTestSemaphoreKey))
	assert.Equal(t, []string{"default/other/other-1"}, pendingOn(ctx, t, syncManager, queueTestMutexKey))
}

const (
	mockDBController = "test-controller"
	mockDBHeldKey    = "default/Mutex/held"
	mockDBFreeKey    = "default/Mutex/free"
)

// newMockDatabaseManager returns a lock manager whose database is mocked, with
// two database-backed mutexes, and the order in which it queried the database.
func newMockDatabaseManager(ctx context.Context, t *testing.T) (*Manager, *syncdbmocks.SyncQueries, *[]string) {
	t.Helper()
	queries := syncdbmocks.NewSyncQueries(t)
	info := syncdb.Info{
		Config:       syncdb.Config{ControllerName: mockDBController},
		SessionProxy: &sqldb.SessionProxy{},
	}
	syncManager, err := NewLockManager(ctx, fake.NewClientset(), "", nil, nil, func(string) {}, WorkflowExistenceFunc, false)
	require.NoError(t, err)
	syncManager.dbInfo = info
	syncManager.queries = queries
	for _, name := range []string{"held", "free"} {
		mutex := newDatabaseMutex(name, "default/"+name, syncManager.nextWorkflow, info)
		mutex.queries = queries
		syncManager.syncLockMap["default/Mutex/"+name] = mutex
	}

	var calls []string
	queries.EXPECT().RemoveFromQueue(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, semaphoreName, _, _ string) error {
			calls = append(calls, "RemoveFromQueue "+semaphoreName)
			return nil
		}).Maybe()
	queries.EXPECT().GetCurrentState(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ *sqldb.SessionProxy, semaphoreName string, _ bool) ([]syncdb.StateRecord, error) {
			calls = append(calls, "GetCurrentState "+semaphoreName)
			return nil, nil
		}).Maybe()
	queries.EXPECT().GetOrderedQueue(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ *sqldb.SessionProxy, semaphoreName string, _ time.Duration) ([]syncdb.StateRecord, error) {
			calls = append(calls, "GetOrderedQueue "+semaphoreName)
			return nil, nil
		}).Maybe()
	return syncManager, queries, &calls
}

// newMockDatabaseVictim records nodes victim-1 to victim-<n> as queued on both
// database-backed mutexes, the first of which each records as waited for.
func newMockDatabaseVictim(ctx context.Context, syncManager *Manager, n int) *wfv1.Workflow {
	wf := newQueueTestWorkflow("victim", queueTestCreated)
	wf.Status.Synchronization = &wfv1.SynchronizationStatus{}
	wf.Status.Nodes = wfv1.Nodes{}
	for i := 1; i <= n; i++ {
		nodeID := fmt.Sprintf("victim-%d", i)
		wf.Status.Nodes.Set(ctx, nodeID, wfv1.NodeStatus{
			ID:                    nodeID,
			SynchronizationStatus: &wfv1.NodeSynchronizationStatus{Waiting: mockDBHeldKey},
		})
		syncManager.recordQueued(getHolderKey(wf, nodeID), mockDBHeldKey)
		syncManager.recordQueued(getHolderKey(wf, nodeID), mockDBFreeKey)
	}
	return wf
}

// ReleaseAll leaves the queues of database-backed locks with one query,
// however many nodes waited on them, and only then notifies each lock's
// waiters, once.
func TestReleaseAllLeavesDatabaseQueuesWithOneQuery(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, queries, calls := newMockDatabaseManager(ctx, t)
	victimWf := newMockDatabaseVictim(ctx, syncManager, 5)
	queries.EXPECT().RemoveWorkflowFromQueues(mock.Anything, "default/victim", mockDBController).
		RunAndReturn(func(context.Context, string, string) error {
			*calls = append(*calls, "RemoveWorkflowFromQueues")
			return nil
		}).Once()

	syncManager.ReleaseAll(ctx, victimWf)

	require.NotEmpty(t, *calls)
	assert.Equal(t, "RemoveWorkflowFromQueues", (*calls)[0], "waiters are notified only after the queues are left")
	assert.ElementsMatch(t, []string{
		"RemoveWorkflowFromQueues",
		"GetCurrentState mtx/default/held", "GetOrderedQueue mtx/default/held",
		"GetCurrentState mtx/default/free", "GetOrderedQueue mtx/default/free",
	}, *calls)
	assert.Empty(t, syncManager.queued)
}

// When the database queues cannot be left, their waiters are not notified, as
// nothing has changed for them.
func TestReleaseAllDoesNotNotifyDatabaseLocksWhenLeavingFails(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, queries, calls := newMockDatabaseManager(ctx, t)
	victimWf := newMockDatabaseVictim(ctx, syncManager, 2)
	queries.EXPECT().RemoveWorkflowFromQueues(mock.Anything, "default/victim", mockDBController).
		RunAndReturn(func(context.Context, string, string) error {
			*calls = append(*calls, "RemoveWorkflowFromQueues")
			return errors.New("database unavailable")
		}).Once()

	assert.True(t, syncManager.ReleaseAll(ctx, victimWf))

	assert.Equal(t, []string{"RemoveWorkflowFromQueues"}, *calls)
	assert.Nil(t, victimWf.Status.Synchronization)
}

// expectRemoveHolderFromQueues expects Dequeue to leave the database queues of
// node victim-1 with one query, which fails with err.
func expectRemoveHolderFromQueues(queries *syncdbmocks.SyncQueries, calls *[]string, err error) {
	queries.EXPECT().RemoveHolderFromQueues(mock.Anything, "default/victim/victim-1", mockDBController).
		RunAndReturn(func(context.Context, string, string) error {
			*calls = append(*calls, "RemoveHolderFromQueues")
			return err
		}).Once()
}

// Dequeue leaves the queues of database-backed locks with one query, and only
// then notifies each lock's waiters.
func TestDequeueLeavesDatabaseQueuesWithOneQuery(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, queries, calls := newMockDatabaseManager(ctx, t)
	victimWf := newMockDatabaseVictim(ctx, syncManager, 2)
	expectRemoveHolderFromQueues(queries, calls, nil)

	syncManager.Dequeue(ctx, victimWf, "victim-1")

	require.NotEmpty(t, *calls)
	assert.Equal(t, "RemoveHolderFromQueues", (*calls)[0], "waiters are notified only after the queues are left")
	assert.ElementsMatch(t, []string{
		"RemoveHolderFromQueues",
		"GetCurrentState mtx/default/held", "GetOrderedQueue mtx/default/held",
		"GetCurrentState mtx/default/free", "GetOrderedQueue mtx/default/free",
	}, *calls)
	assert.Equal(t, map[string][]string{"default/victim/victim-2": {mockDBHeldKey, mockDBFreeKey}}, syncManager.queued["default/victim"], "only the node is dequeued")
}

// A controller that restarted does not know which database queues the node
// joined before, and still leaves all of them.
func TestDequeueAfterRestartLeavesDatabaseQueues(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, queries, calls := newMockDatabaseManager(ctx, t)
	expectRemoveHolderFromQueues(queries, calls, nil)

	syncManager.Dequeue(ctx, newMockDatabaseVictim(ctx, syncManager, 0), "victim-1")

	assert.Equal(t, []string{"RemoveHolderFromQueues"}, *calls)
}

// When the database queues cannot be left, their waiters are not notified, and
// the queues are remembered, so that a later Dequeue notifies them once it has
// left them.
func TestDequeueRetriesFailedDatabaseQueueRemoval(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	syncManager, queries, calls := newMockDatabaseManager(ctx, t)
	victimWf := newMockDatabaseVictim(ctx, syncManager, 1)
	expectRemoveHolderFromQueues(queries, calls, errors.New("database unavailable"))

	syncManager.Dequeue(ctx, victimWf, "victim-1")

	assert.Equal(t, []string{"RemoveHolderFromQueues"}, *calls)
	assert.Equal(t, map[string][]string{"default/victim/victim-1": {mockDBHeldKey, mockDBFreeKey}}, syncManager.queued["default/victim"])

	*calls = nil
	expectRemoveHolderFromQueues(queries, calls, nil)
	syncManager.Dequeue(ctx, victimWf, "victim-1")

	assert.ElementsMatch(t, []string{
		"RemoveHolderFromQueues",
		"GetCurrentState mtx/default/held", "GetOrderedQueue mtx/default/held",
		"GetCurrentState mtx/default/free", "GetOrderedQueue mtx/default/free",
	}, *calls)
	assert.Empty(t, syncManager.queued)
}

// stateRows returns the rows of the database's lock state table, as
// "<lock> <holder key> <controller> <held>".
func stateRows(ctx context.Context, t *testing.T, info syncdb.Info) []string {
	t.Helper()
	var records []syncdb.StateRecord
	require.NoError(t, info.SessionProxy.Session(ctx).SQL().
		Select(syncdb.StateNameField, syncdb.StateKeyField, syncdb.StateControllerField, syncdb.StateHeldField).
		From(info.Config.StateTable).
		All(&records))
	rows := make([]string, len(records))
	for i, r := range records {
		rows[i] = fmt.Sprintf("%s %s %s %t", r.Name, r.Key, r.Controller, r.Held)
	}
	return rows
}

func insertStateRow(ctx context.Context, t *testing.T, info syncdb.Info, record syncdb.StateRecord) {
	t.Helper()
	record.Time = time.Now()
	_, err := info.SessionProxy.Session(ctx).Collection(info.Config.StateTable).Insert(&record)
	require.NoError(t, err)
}

// The queues of database-backed locks outlive the controller. A controller
// that restarted, and so does not know which queues a workflow joined before,
// must still remove the workflow from all of them when it releases it, and
// only from those: held rows, the rows of a workflow whose name the
// workflow's name prefixes, and another controller's rows are kept.
func TestReleaseAllRemovesNodeFromEveryDatabaseWaitQueueAfterRestart(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
			defer cancel()
			info, deferfn, cfg, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer deferfn()
			newManager := func() *Manager {
				return createLockManager(ctx, info.SessionProxy, &cfg, func(_ context.Context, _ string) (int, error) { return 1, nil }, func(string) {}, WorkflowExistenceFunc)
			}
			held := &wfv1.Mutex{Name: "held", Database: true}
			free := &wfv1.Mutex{Name: "free", Database: true}
			controller := info.Config.ControllerName

			syncManager := newManager()
			acquired, _, _, _, err := syncManager.TryAcquire(ctx, newQueueTestWorkflow("holder", queueTestCreated), "holder-1", &wfv1.Synchronization{
				Mutexes: []*wfv1.Mutex{held},
			})
			require.NoError(t, err)
			require.True(t, acquired)
			victimWf := newQueueTestWorkflow("victim", queueTestCreated)
			acquired, _, _, _, err = syncManager.TryAcquire(ctx, victimWf, "victim-1", &wfv1.Synchronization{
				Mutexes: []*wfv1.Mutex{held, free},
			})
			require.NoError(t, err)
			require.False(t, acquired)
			acquired, _, _, _, err = syncManager.TryAcquire(ctx, newQueueTestWorkflow("victim-2", queueTestCreated), "victim-2-1", &wfv1.Synchronization{
				Mutexes: []*wfv1.Mutex{held},
			})
			require.NoError(t, err)
			require.False(t, acquired)
			// A hold the status does not record, and another controller's
			// entry for the same holder key.
			insertStateRow(ctx, t, info, syncdb.StateRecord{Name: "mtx/default/elsewhere", Key: "default/victim/victim-9", Controller: controller, Held: true})
			insertStateRow(ctx, t, info, syncdb.StateRecord{Name: "mtx/default/held", Key: "default/victim/victim-1", Controller: "other-controller", Held: false})

			// The controller restarts: a new manager over the same database.
			restarted := newManager()
			restarted.ReleaseAll(ctx, victimWf)

			assert.ElementsMatch(t, []string{
				"mtx/default/held default/holder/holder-1 " + controller + " true",
				"mtx/default/held default/victim-2/victim-2-1 " + controller + " false",
				"mtx/default/held default/victim/victim-1 other-controller false",
				"mtx/default/elsewhere default/victim/victim-9 " + controller + " true",
			}, stateRows(ctx, t, info))
			acquired, _, msg, _, err := restarted.TryAcquire(ctx, newQueueTestWorkflow("other", queueTestCreated.Add(time.Minute)), "other-1", &wfv1.Synchronization{
				Mutexes: []*wfv1.Mutex{free},
			})
			require.NoError(t, err)
			assert.True(t, acquired, "got %q", msg)
		})
	}
}

// A node that waits for database-backed locks when the controller restarts,
// and errors before it tries to acquire them again (as on an invalid display
// name), must leave their queues while its workflow still runs, or its
// entries keep a free lock from every later holder until the workflow ends.
// It must leave only its own entries: it keeps the locks it holds.
func TestDequeueAfterRestartRemovesNodeFromEveryDatabaseWaitQueue(t *testing.T) {
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
			defer cancel()
			info, deferfn, cfg, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer deferfn()
			newManager := func() *Manager {
				return createLockManager(ctx, info.SessionProxy, &cfg, func(_ context.Context, _ string) (int, error) { return 1, nil }, func(string) {}, func(string) bool { return true })
			}
			held := &wfv1.Mutex{Name: "held", Database: true}
			free := &wfv1.Mutex{Name: "free", Database: true}

			syncManager := newManager()
			acquired, _, _, _, err := syncManager.TryAcquire(ctx, newQueueTestWorkflow("holder", queueTestCreated), "holder-1", &wfv1.Synchronization{
				Mutexes: []*wfv1.Mutex{held},
			})
			require.NoError(t, err)
			require.True(t, acquired)
			victimWf := newQueueTestWorkflow("victim", queueTestCreated)
			acquired, _, _, _, err = syncManager.TryAcquire(ctx, victimWf, "victim-1", &wfv1.Synchronization{
				Mutexes: []*wfv1.Mutex{held, free},
			})
			require.NoError(t, err)
			require.False(t, acquired)
			// Entries Dequeue must keep: a lock the node holds, the entries of
			// the node's workflow itself, of a node whose ID the node's ID
			// prefixes, of a workflow whose name the workflow's name prefixes,
			// and another controller's entry for the same node.
			controller := info.Config.ControllerName
			for _, record := range []syncdb.StateRecord{
				{Name: "mtx/default/elsewhere", Key: "default/victim/victim-1", Controller: controller, Held: true},
				{Name: "mtx/default/held", Key: "default/victim", Controller: controller},
				{Name: "mtx/default/held", Key: "default/victim/victim-10", Controller: controller},
				{Name: "mtx/default/held", Key: "default/victim-2/victim-1", Controller: controller},
				{Name: "mtx/default/held", Key: "default/victim/victim-1", Controller: "other-controller"},
			} {
				insertStateRow(ctx, t, info, record)
			}

			// The controller restarts, and the node errors on the first
			// reconcile, before TryAcquire. The workflow keeps running.
			restarted := newManager()
			restarted.Dequeue(ctx, victimWf, "victim-1")

			kept := []string{
				"mtx/default/held default/holder/holder-1 " + controller + " true",
				"mtx/default/elsewhere default/victim/victim-1 " + controller + " true",
				"mtx/default/held default/victim " + controller + " false",
				"mtx/default/held default/victim/victim-10 " + controller + " false",
				"mtx/default/held default/victim-2/victim-1 " + controller + " false",
				"mtx/default/held default/victim/victim-1 other-controller false",
			}
			assert.ElementsMatch(t, kept, stateRows(ctx, t, info))
			acquired, _, msg, _, err := restarted.TryAcquire(ctx, newQueueTestWorkflow("other", queueTestCreated.Add(time.Minute)), "other-1", &wfv1.Synchronization{
				Mutexes: []*wfv1.Mutex{free},
			})
			require.NoError(t, err)
			assert.True(t, acquired, "got %q", msg)

			// Dequeue again, as a later reconcile may: nothing else changes.
			restarted.Dequeue(ctx, victimWf, "victim-1")

			assert.ElementsMatch(t, append(kept, "mtx/default/free default/other/other-1 "+controller+" true"), stateRows(ctx, t, info))
		})
	}
}
