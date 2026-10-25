package sync

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-workflows/v4/config"
	argoErr "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
	syncdb "github.com/argoproj/argo-workflows/v4/util/sync/db"
)

var WorkflowExistenceFunc = func(s string) bool {
	return false
}

func GetSyncLimitFunc(kube *fake.Clientset) func(context.Context, string) (int, error) {
	return func(ctx context.Context, lockName string) (int, error) {
		items := strings.Split(lockName, "/")
		if len(items) < 4 {
			return 0, argoErr.New(argoErr.CodeBadRequest, "Invalid Config Map Key")
		}

		configMap, err := kube.CoreV1().ConfigMaps(items[0]).Get(ctx, items[2], metav1.GetOptions{})
		if err != nil {
			return 0, err
		}

		value, found := configMap.Data[items[3]]

		if !found {
			return 0, argoErr.New(argoErr.CodeBadRequest, "Invalid Sync configuration Key")
		}
		return strconv.Atoi(value)
	}
}

func TestSemaphoreWfLevel(t *testing.T) {
	kube := fake.NewClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal("@testdata/sync_manager/config-map.yaml", &cm)

	ctx := logging.TestContext(t.Context())
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("InitializeSynchronization", func(t *testing.T) {
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-status.yaml")
		wfclientset := fakewfclientset.NewClientset(wf)

		wfList, err := wfclientset.ArgoprojV1alpha1().Workflows("default").List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		staleHolds, err := syncManager.Initialize(ctx, wfList.Items)
		require.NoError(t, err)
		require.Empty(t, staleHolds)
		assert.Len(t, syncManager.syncLockMap, 1)
	})
	t.Run("InitializeSynchronizationWithInvalid", func(t *testing.T) {
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-status.yaml")
		invalidSync := []wfv1.SemaphoreHolding{{Semaphore: "default/configmap/my-config1/workflow", Holders: []string{"hello-world-vcrg5"}}}
		wf.Status.Synchronization.Semaphore.Holding = invalidSync
		wfclientset := fakewfclientset.NewClientset(wf)
		wfList, err := wfclientset.ArgoprojV1alpha1().Workflows("default").List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		// An invalid/undecodable lock name now fails closed rather than being
		// silently skipped: we can't poison it or prove the spec re-acquires it,
		// so the controller halts for an operator instead of risking a double-acquire.
		_, err = syncManager.Initialize(ctx, wfList.Items)
		require.Error(t, err)
		assert.Empty(t, syncManager.syncLockMap)
	})
	t.Run("InitializeMultipleWorkflowsHolding", func(t *testing.T) {
		// This test verifies that when multiple workflows claim to hold the same semaphore
		// (which can happen with stale status after a controller restart), ALL of their
		// holders are registered during Initialize, not just those after the first workflow.
		// This was a bug caused by variable shadowing in Initialize (PR #3141).

		// Create a ConfigMap with semaphore limit of 3 to allow multiple holders
		kubeClient := fake.NewClientset()
		_, err := kubeClient.CoreV1().ConfigMaps("default").Create(ctx, &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "my-config"},
			Data:       map[string]string{"workflow": "3"},
		}, metav1.CreateOptions{})
		require.NoError(t, err)

		syncManager, err := NewLockManager(ctx, kubeClient, "", nil, GetSyncLimitFunc(kubeClient), func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		// Create first workflow claiming to hold the semaphore
		wf1 := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-status.yaml")
		wf1.Name = "hello-world-one"
		wf1.Status.Synchronization.Semaphore.Holding[0].Holders = []string{"default/hello-world-one"}

		// Create second workflow also claiming to hold the same semaphore
		wf2 := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-status.yaml")
		wf2.Name = "hello-world-two"
		wf2.Status.Synchronization.Semaphore.Holding[0].Holders = []string{"default/hello-world-two"}

		// Initialize with both workflows
		staleHolds, err := syncManager.Initialize(ctx, []wfv1.Workflow{*wf1, *wf2})
		require.NoError(t, err)
		require.Empty(t, staleHolds)

		// Verify the semaphore was created
		assert.Len(t, syncManager.syncLockMap, 1)

		// Verify BOTH holders are registered (the bug would only register the second one)
		sem := syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		require.NotNil(t, sem)
		holders, err := sem.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Len(t, holders, 2, "both workflows should be registered as holders")
		assert.Contains(t, holders, "default/hello-world-one")
		assert.Contains(t, holders, "default/hello-world-two")
	})

	t.Run("InitializeConfigMapInitFailurePoisonsNotFatal", func(t *testing.T) {
		// The ConfigMap backing the semaphore limit is absent, so the limit fetch
		// fails and initializeSemaphore errors. The name decodes (kind ConfigMap),
		// so this is recoverable - it must not crashloop the controller. It must
		// POISON the lock rather than leave it absent: a ConfigMap semaphore keeps
		// its holders only in memory, so leaving it absent would let a later
		// rebuild start with zero holders and a racer double-acquire this hold.
		kubeClient := fake.NewClientset() // no ConfigMap created
		syncManager, err := NewLockManager(ctx, kubeClient, "", nil, GetSyncLimitFunc(kubeClient), func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-status.yaml")
		wf.Name = "cm-holder"
		wf.Status.Synchronization.Semaphore.Holding[0].Holders = []string{"default/cm-holder"}

		staleHolds, err := syncManager.Initialize(ctx, []wfv1.Workflow{*wf})
		require.NoError(t, err, "a recoverable ConfigMap failure must not be fatal")
		require.Empty(t, staleHolds)
		lock := syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		require.NotNil(t, lock)
		_, poisoned := lock.(*poisonedLock)
		assert.True(t, poisoned, "decodable lock that cannot be initialized must be poisoned, not left absent")
	})

	t.Run("InitializeLoweredLimitForceRegistersAllHolders", func(t *testing.T) {
		// Two workflows are recorded as holders but the limit is now 1 (lowered
		// from 2). Lowering a limit is routine: it must NOT poison the shared
		// semaphore (which would block every contender until restart), NOR drop
		// the over-limit holder (which would let a racer double-acquire once the
		// other holder releases). Both holders must be force-registered so the
		// in-memory count reflects reality and new acquisitions wait until the
		// count drains below the limit.
		kubeClient := fake.NewClientset()
		_, err := kubeClient.CoreV1().ConfigMaps("default").Create(ctx, &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "my-config"},
			Data:       map[string]string{"workflow": "1"},
		}, metav1.CreateOptions{})
		require.NoError(t, err)

		syncManager, err := NewLockManager(ctx, kubeClient, "", nil, GetSyncLimitFunc(kubeClient), func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf1 := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-status.yaml")
		wf1.Name = "holder-one"
		wf1.Status.Synchronization.Semaphore.Holding[0].Holders = []string{"default/holder-one"}
		wf2 := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-status.yaml")
		wf2.Name = "holder-two"
		wf2.Status.Synchronization.Semaphore.Holding[0].Holders = []string{"default/holder-two"}

		staleHolds, err := syncManager.Initialize(ctx, []wfv1.Workflow{*wf1, *wf2})
		require.NoError(t, err)
		require.Empty(t, staleHolds)

		lock := syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		require.NotNil(t, lock)
		_, poisoned := lock.(*poisonedLock)
		assert.False(t, poisoned, "lowering a limit must not poison the semaphore")
		holders, err := lock.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Len(t, holders, 2, "both recorded holders must be force-registered, even over the lowered limit")
		assert.Contains(t, holders, "default/holder-one")
		assert.Contains(t, holders, "default/holder-two")

		// A new contender must NOT be able to acquire: the count (2) is over the
		// limit (1), so the lock is correctly unavailable until holders drain.
		racer := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-status.yaml")
		racer.Name = "racer"
		acquired, _, _, _, err := syncManager.TryAcquire(ctx, racer, "", racer.Spec.Synchronization)
		require.NoError(t, err)
		assert.False(t, acquired, "a racer must wait while recorded holders exceed the lowered limit")
	})

	t.Run("InitializeMutexWithWorkflowTemplateRef", func(t *testing.T) {
		// A workflowTemplateRef workflow has empty Spec.Synchronization and
		// Spec.Templates; its rendered mutex lives in Status.StoredWorkflowSpec.
		// Its recorded holder uses a self-describing V2 key, so Initialize must
		// re-establish the hold WITHOUT needing to resolve a level from wf.Spec.
		// Before the fix, getWorkflowSyncLevelByName could not find the mutex in
		// the empty spec, Initialize hit "cannot obtain lock level" and dropped
		// the holder, leaving the lock free for a racing workflow to acquire.
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "tmplref-holder", Namespace: "default"},
			Spec: wfv1.WorkflowSpec{
				WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "my-template"},
			},
			Status: wfv1.WorkflowStatus{
				StoredWorkflowSpec: &wfv1.WorkflowSpec{
					Synchronization: &wfv1.Synchronization{
						Mutexes: []*wfv1.Mutex{{Name: "my-mutex"}},
					},
				},
				Synchronization: &wfv1.SynchronizationStatus{
					Mutex: &wfv1.MutexStatus{
						Holding: []wfv1.MutexHolding{{
							Mutex:  "default/Mutex/my-mutex",
							Holder: "default/tmplref-holder",
						}},
					},
				},
			},
		}

		staleHolds, err := syncManager.Initialize(ctx, []wfv1.Workflow{*wf})
		require.NoError(t, err)
		require.Empty(t, staleHolds)

		lock := syncManager.syncLockMap["default/Mutex/my-mutex"]
		require.NotNil(t, lock)
		_, poisoned := lock.(*poisonedLock)
		assert.False(t, poisoned, "holder should be re-established, not poisoned")
		holders, err := lock.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"default/tmplref-holder"}, holders)
	})

	t.Run("InitializePoisonsUnresolvableHolder", func(t *testing.T) {
		// A legacy V1 holder key that cannot be resolved to a level (no matching
		// synchronization block in spec or stored spec) must poison the lock,
		// rather than silently dropping the holder. Otherwise a racing workflow
		// could acquire a mutex that persisted state says is already held.
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		holder := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "ghost-holder", Namespace: "default"},
			Status: wfv1.WorkflowStatus{
				Synchronization: &wfv1.SynchronizationStatus{
					Mutex: &wfv1.MutexStatus{
						Holding: []wfv1.MutexHolding{{
							Mutex:  "default/Mutex/my-mutex",
							Holder: "ghost-holder", // V1 key, unresolvable against empty spec
						}},
					},
				},
			},
		}

		// A decodable name whose holder can't be re-established is poisoned, not
		// fatal: the poison key matches what a racer computes, so soundness holds.
		staleHolds, err := syncManager.Initialize(ctx, []wfv1.Workflow{*holder})
		require.NoError(t, err)
		require.Empty(t, staleHolds)

		lock := syncManager.syncLockMap["default/Mutex/my-mutex"]
		require.NotNil(t, lock)
		_, poisoned := lock.(*poisonedLock)
		assert.True(t, poisoned, "unresolvable holder must poison the lock")

		// A racing workflow must not be able to acquire the poisoned mutex.
		racer := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-mutex.yaml")
		racer.Name = "racer"
		acquired, _, msg, _, err := syncManager.TryAcquire(ctx, racer, "", racer.Spec.Synchronization)
		require.NoError(t, err)
		assert.False(t, acquired, "racing workflow must not acquire a poisoned lock")
		assert.Contains(t, msg, "poisoned state")
	})

	t.Run("InitializeFailsClosedOnUndecodableLockName", func(t *testing.T) {
		// A holder whose lock name cannot even be decoded is an unknowable hold:
		// we can neither poison the lock (no key) nor prove the spec re-acquires
		// it, so continuing risks a silent double-acquire. Initialize must fail
		// closed (return an error) so the controller halts for an operator.
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		holder := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "broken-holder", Namespace: "default"},
			Status: wfv1.WorkflowStatus{
				Synchronization: &wfv1.SynchronizationStatus{
					Mutex: &wfv1.MutexStatus{
						Holding: []wfv1.MutexHolding{{
							Mutex:  "garbage-undecodable-name",
							Holder: "default/broken-holder",
						}},
					},
				},
			},
		}

		_, err = syncManager.Initialize(ctx, []wfv1.Workflow{*holder})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "broken-holder")
		assert.NotContains(t, syncManager.syncLockMap, "garbage-undecodable-name")
	})

	t.Run("InitializeFailsClosedWhenDBUnavailable", func(t *testing.T) {
		// A Running workflow holds a database-backed lock, but the manager has no
		// database session (e.g. the DB was unreachable at startup). The lock
		// cannot be re-established, so Initialize must fail closed and let the
		// controller crashloop until the database is reachable.
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)
		require.Nil(t, syncManager.dbInfo.SessionProxy)

		holder := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "db-holder", Namespace: "default"},
			Status: wfv1.WorkflowStatus{
				Synchronization: &wfv1.SynchronizationStatus{
					Mutex: &wfv1.MutexStatus{
						Holding: []wfv1.MutexHolding{{
							Mutex:  "default/Database/my-db-lock",
							Holder: "default/db-holder",
						}},
					},
				},
			},
		}

		_, err = syncManager.Initialize(ctx, []wfv1.Workflow{*holder})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database session")
	})

	t.Run("WfLevelAcquireAndRelease", func(t *testing.T) {
		var nextKey string
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			nextKey = key
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-semaphore.yaml")
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()
		wf3 := wf.DeepCopy()
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
		require.NotNil(t, wf.Status.Synchronization.Semaphore.Holding)
		key := getHolderKey(wf, "")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		// Try to acquire again
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)

		wf1.Name = "two"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		wf2.Spec.Priority = new(int32(5))
		holderKey2 := getHolderKey(wf2, "")
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf3.Name = "four"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf3, "", wf3.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		syncManager.Release(ctx, wf, "", wf.Spec.Synchronization)
		assert.Equal(t, holderKey2, nextKey)
		require.NotNil(t, wf.Status.Synchronization)
		assert.Empty(t, wf.Status.Synchronization.Semaphore.Holding[0].Holders)

		// Low priority workflow try to acquire the lock
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// High Priority workflow acquires the lock
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf2.Status.Synchronization)
		require.NotNil(t, wf2.Status.Synchronization.Semaphore)
		key = getHolderKey(wf2, "")
		assert.Equal(t, key, wf2.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		syncManager.ReleaseAll(ctx, wf2)
		assert.Nil(t, wf2.Status.Synchronization)

		sema := syncManager.syncLockMap["default/ConfigMap/my-config/workflow"].(*prioritySemaphore)
		require.NotNil(t, sema)
		assert.Len(t, sema.pending.items, 2)
		syncManager.ReleaseAll(ctx, wf1)
		assert.Len(t, sema.pending.items, 1)
		syncManager.ReleaseAll(ctx, wf3)
		assert.Empty(t, sema.pending.items)
	})

	t.Run("WorkflowLevelSemaphoreAcquireAndReleaseWithMultipleSemaphores", func(t *testing.T) {
		// Create ConfigMap with multiple semaphore limits
		cm := v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "multiple-sema-config",
				Namespace: "default",
			},
			Data: map[string]string{
				"sem1": "1",
				"sem2": "1",
				"sem3": "1",
			},
		}
		_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
		require.NoError(t, err)

		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			// nextKey = key
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		// Create two workflows that both need all semaphores
		wf1 := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-semaphore.yaml")
		wf1.Name = "wf1"
		wf1.Spec.Synchronization.Semaphores = []*wfv1.SemaphoreRef{
			{
				ConfigMapKeyRef: &v1.ConfigMapKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "multiple-sema-config"},
					Key:                  "sem1",
				},
			},
			{
				ConfigMapKeyRef: &v1.ConfigMapKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "multiple-sema-config"},
					Key:                  "sem2",
				},
			},
			{
				ConfigMapKeyRef: &v1.ConfigMapKeySelector{
					LocalObjectReference: v1.LocalObjectReference{Name: "multiple-sema-config"},
					Key:                  "sem3",
				},
			},
		}

		wf2 := wf1.DeepCopy()
		wf2.Name = "wf2"

		// First workflow should acquire all semaphores
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf1.Status.Synchronization)
		require.NotNil(t, wf1.Status.Synchronization.Semaphore)
		require.NotNil(t, wf1.Status.Synchronization.Semaphore.Holding)
		assert.Len(t, wf1.Status.Synchronization.Semaphore.Holding, 3)

		// Release all semaphores from first workflow
		syncManager.ReleaseAll(ctx, wf1)

		// Second workflow should now be able to acquire all semaphores
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf2.Status.Synchronization)
		require.NotNil(t, wf2.Status.Synchronization.Semaphore)
		require.NotNil(t, wf2.Status.Synchronization.Semaphore.Holding)
		assert.Len(t, wf2.Status.Synchronization.Semaphore.Holding, 3)

		// Clean up
		syncManager.ReleaseAll(ctx, wf2)
	})
}

func TestResizeSemaphoreSize(t *testing.T) {
	kube := fake.NewClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal("@testdata/sync_manager/config-map.yaml", &cm)

	ctx := logging.TestContext(t.Context())
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("WfLevelAcquireAndRelease", func(t *testing.T) {
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-semaphore.yaml")
		wf.CreationTimestamp = metav1.Time{Time: time.Now()}
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()
		// Distinct creation times so the queue order is the submission order
		// this test walks through; tied times are ordered by name.
		wf1.CreationTimestamp = metav1.Time{Time: wf.CreationTimestamp.Add(time.Second)}
		wf2.CreationTimestamp = metav1.Time{Time: wf.CreationTimestamp.Add(2 * time.Second)}
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
		key := getHolderKey(wf, "")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		wf1.Name = "two"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		// Increase the semaphore Size
		cm, err := kube.CoreV1().ConfigMaps("default").Get(ctx, "my-config", metav1.GetOptions{})
		require.NoError(t, err)
		cm.Data["workflow"] = "3"
		_, err = kube.CoreV1().ConfigMaps("default").Update(ctx, cm, metav1.UpdateOptions{})
		require.NoError(t, err)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf1.Status.Synchronization)
		require.NotNil(t, wf1.Status.Synchronization.Semaphore)
		key = getHolderKey(wf1, "")
		assert.Equal(t, key, wf1.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf2.Status.Synchronization)
		require.NotNil(t, wf2.Status.Synchronization.Semaphore)
		key = getHolderKey(wf2, "")
		assert.Equal(t, key, wf2.Status.Synchronization.Semaphore.Holding[0].Holders[0])
	})
}

func TestSemaphoreTmplLevel(t *testing.T) {
	kube := fake.NewClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal("@testdata/sync_manager/config-map.yaml", &cm)

	ctx := logging.TestContext(t.Context())
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("TemplateLevelAcquireAndRelease", func(t *testing.T) {
		// var nextKey string
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			// nextKey = key
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-tmpl-semaphore.yaml")
		tmpl := wf.Spec.Templates[2]

		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
		key := getHolderKey(wf, "semaphore-tmpl-level-xjvln-3448864205")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])

		// Try to acquire again
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-1607747183", tmpl.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/template", failedLockName)
		assert.True(t, wfUpdate)
		assert.False(t, status)

		syncManager.Release(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
		assert.Empty(t, wf.Status.Synchronization.Semaphore.Holding[0].Holders)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-1607747183", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
		key = getHolderKey(wf, "semaphore-tmpl-level-xjvln-1607747183")
		assert.Equal(t, key, wf.Status.Synchronization.Semaphore.Holding[0].Holders[0])
	})
}

type mockGetSyncLimit struct {
	callCount  int
	outputSize int
	outputErr  error
}

func (m *mockGetSyncLimit) getSyncLimit(_ context.Context, s string) (int, error) {
	m.callCount++
	return m.outputSize, m.outputErr
}

func TestSemaphoreSizeCache(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kube := fake.NewClientset()

	mockedNow := time.Now()
	nowFn = func() time.Time {
		return mockedNow
	}
	defer func() {
		nowFn = time.Now
	}()

	t.Run("WfLevelAcquireAndRelease", func(t *testing.T) {
		mock := mockGetSyncLimit{}
		mock.outputSize = 10
		config := config.SyncConfig{
			SemaphoreLimitCacheSeconds: new(int64(1)),
		}

		syncManager, err := NewLockManager(ctx, kube, "", &config, mock.getSyncLimit, func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-semaphore.yaml")
		wf.CreationTimestamp = metav1.Time{Time: time.Now()}

		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.Equal(t, 1, mock.callCount)

		sem := syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 10, sem.getLimit(ctx))

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 1, mock.callCount)

		sem = syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 10, sem.getLimit(ctx))

		mockedNow = mockedNow.Add(1 * time.Second)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 2, mock.callCount)

		sem = syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 10, sem.getLimit(ctx))

		// semaphore age should be updated to now
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 2, mock.callCount)

		sem = syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 10, sem.getLimit(ctx))

		mockedNow = mockedNow.Add(1 * time.Second)
		mock.outputSize = 20

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 3, mock.callCount)

		sem = syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 20, sem.getLimit(ctx))

		// semaphore age should be updated to now again
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 3, mock.callCount)

		sem = syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]
		assert.Equal(t, 20, sem.getLimit(ctx))
	})

	t.Run("TemplateLevelAcquireAndRelease", func(t *testing.T) {
		mock := mockGetSyncLimit{}
		mock.outputSize = 10

		config := config.SyncConfig{
			SemaphoreLimitCacheSeconds: new(int64(1)),
		}

		syncManager, err := NewLockManager(ctx, kube, "", &config, mock.getSyncLimit, func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-tmpl-semaphore.yaml")
		tmpl := wf.Spec.Templates[2]

		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		assert.Equal(t, 1, mock.callCount)

		sem := syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 10, sem.getLimit(ctx))

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 1, mock.callCount)

		sem = syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 10, sem.getLimit(ctx))

		mockedNow = mockedNow.Add(1 * time.Second)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 2, mock.callCount)

		sem = syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 10, sem.getLimit(ctx))

		// semaphore age should be updated to now
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 2, mock.callCount)

		sem = syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 10, sem.getLimit(ctx))

		mockedNow = mockedNow.Add(1 * time.Second)
		mock.outputSize = 20

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 3, mock.callCount)

		sem = syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 20, sem.getLimit(ctx))

		// semaphore age should be updated to now again
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf, "semaphore-tmpl-level-xjvln-3448864205", tmpl.Synchronization)
		require.NoError(t, err)
		assert.True(t, status)
		assert.Empty(t, failedLockName)
		assert.False(t, wfUpdate)
		assert.Empty(t, msg)
		assert.Equal(t, 3, mock.callCount)

		sem = syncManager.syncLockMap["default/ConfigMap/my-config/template"]
		assert.Equal(t, 20, sem.getLimit(ctx))
	})
}

func TestTriggerWFWithAvailableLock(t *testing.T) {
	assert := assert.New(t)
	kube := fake.NewClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal("@testdata/sync_manager/config-map.yaml", &cm)
	cm.Data["workflow"] = "3"

	ctx := logging.TestContext(t.Context())
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("TriggerWfsWithAvailableLocks", func(t *testing.T) {
		triggerCount := 0
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			triggerCount++
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		var wfs []wfv1.Workflow
		for i := range 3 {
			wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-semaphore.yaml")
			wf.Name = fmt.Sprintf("%s-%d", "acquired", i)
			status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
			require.NoError(t, err)
			assert.Empty(msg)
			assert.Empty(failedLockName)
			assert.True(status)
			assert.True(wfUpdate)
			wfs = append(wfs, *wf)
		}
		for i := range 3 {
			wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-semaphore.yaml")
			wf.Name = fmt.Sprintf("%s-%d", "wait", i)
			status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
			require.NoError(t, err)
			assert.NotEmpty(msg)
			assert.Equal("default/ConfigMap/my-config/workflow", failedLockName)
			assert.False(status)
			assert.True(wfUpdate)
		}
		syncManager.Release(ctx, &wfs[0], "", wfs[0].Spec.Synchronization)
		triggerCount = 0
		syncManager.Release(ctx, &wfs[1], "", wfs[1].Spec.Synchronization)
		assert.Equal(2, triggerCount)
	})
}

func TestMutexWfLevel(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kube := fake.NewClientset()
	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("WorkflowLevelMutexAcquireAndRelease", func(t *testing.T) {
		// var nextKey string
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			// nextKey = key
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-mutex.yaml")
		wf1 := wf.DeepCopy()
		wf2 := wf.DeepCopy()

		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.Empty(t, msg)
		assert.Empty(t, failedLockName)
		assert.True(t, status)
		assert.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Mutex)
		require.NotNil(t, wf.Status.Synchronization.Mutex.Holding)

		wf1.Name = "two"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/my-mutex", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		wf2.Name = "three"
		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		require.NoError(t, err)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Mutex/my-mutex", failedLockName)
		assert.False(t, status)
		assert.True(t, wfUpdate)

		mutex := syncManager.syncLockMap["default/Mutex/my-mutex"].(*prioritySemaphore)
		require.NotNil(t, mutex)
		assert.Len(t, mutex.pending.items, 2)
		syncManager.ReleaseAll(ctx, wf1)
		assert.Len(t, mutex.pending.items, 1)
		syncManager.ReleaseAll(ctx, wf2)
		assert.Empty(t, mutex.pending.items)
	})

	t.Run("WorkflowLevelMutexAcquireAndReleaseWithMultipleMutex", func(t *testing.T) {
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			// nextKey = key
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-mutex.yaml")
		mutexes := make([]*wfv1.Mutex, 0, 10)
		for i := range 10 {
			mutexes = append(mutexes, &wfv1.Mutex{Name: fmt.Sprintf("mutex%d", i)})
		}
		wf.Spec.Synchronization.Mutexes = mutexes
		wf1 := wf.DeepCopy()
		wf1.Name = "two"

		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		require.Empty(t, msg)
		require.Empty(t, failedLockName)
		require.True(t, status)
		require.True(t, wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Mutex)
		require.NotNil(t, wf.Status.Synchronization.Mutex.Holding)
		syncManager.ReleaseAll(ctx, wf)

		status, wfUpdate, msg, failedLockName, err = syncManager.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		require.Empty(t, msg)
		require.Empty(t, failedLockName)
		require.True(t, status)
		require.True(t, wfUpdate)
		require.NotNil(t, wf1.Status.Synchronization)
		require.NotNil(t, wf1.Status.Synchronization.Mutex)
		require.NotNil(t, wf1.Status.Synchronization.Mutex.Holding)
		syncManager.ReleaseAll(ctx, wf1)
	})
}

func TestCheckWorkflowExistence(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kube := fake.NewClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal("@testdata/sync_manager/config-map.yaml", &cm)
	cm.Data["workflow"] = "1"

	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)

	syncLimitFunc := GetSyncLimitFunc(kube)
	t.Run("WorkflowDeleted", func(t *testing.T) {
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
			// nextKey = key
		}, func(s string) bool {
			return strings.Contains(s, "test1")
		}, false)
		require.NoError(t, err)

		wfMutex := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-mutex.yaml")
		wfMutex1 := wfMutex.DeepCopy()
		wfMutex1.Name = "test1"
		wfSema := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-semaphore.yaml")
		wfSema1 := wfSema.DeepCopy()
		wfSema1.Name = "test2"
		_, _, _, _, _ = syncManager.TryAcquire(ctx, wfMutex, "", wfMutex.Spec.Synchronization)
		_, _, _, _, _ = syncManager.TryAcquire(ctx, wfMutex1, "", wfMutex.Spec.Synchronization)
		_, _, _, _, _ = syncManager.TryAcquire(ctx, wfSema, "", wfSema.Spec.Synchronization)
		_, _, _, _, _ = syncManager.TryAcquire(ctx, wfSema1, "", wfSema.Spec.Synchronization)
		mutex := syncManager.syncLockMap["default/Mutex/my-mutex"].(*prioritySemaphore)
		sem := syncManager.syncLockMap["default/ConfigMap/my-config/workflow"]

		// Pre-state: mutex has 1 holder (hello-world) and 1 pending (test1)
		holders, err := mutex.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Len(t, holders, 1)
		pending, err := mutex.getCurrentPending(ctx)
		require.NoError(t, err)
		assert.Len(t, pending, 1)

		// Pre-state: semaphore has 1 holder (hello-world) and 1 pending (test2)
		holders, err = sem.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Len(t, holders, 1)
		pending, err = sem.getCurrentPending(ctx)
		require.NoError(t, err)
		assert.Len(t, pending, 1)

		syncManager.CheckWorkflowExistence(ctx)

		// Post-state: mutex holder (hello-world) removed, pending (test1) remains
		holders, err = mutex.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Empty(t, holders)
		pending, err = mutex.getCurrentPending(ctx)
		require.NoError(t, err)
		assert.Len(t, pending, 1)

		// Post-state: semaphore holder (hello-world) and pending (test2) both removed
		holders, err = sem.getCurrentHolders(ctx)
		require.NoError(t, err)
		assert.Empty(t, holders)
		pending, err = sem.getCurrentPending(ctx)
		require.NoError(t, err)
		assert.Empty(t, pending)
	})
}

func TestTriggerWFWithSemaphoreAndMutex(t *testing.T) {
	assert := assert.New(t)
	kube := fake.NewClientset()
	var cm v1.ConfigMap
	wfv1.MustUnmarshal("@testdata/sync_manager/config-map.yaml", &cm)
	cm.Data["test-sem"] = "1"

	ctx := logging.TestContext(t.Context())
	_, err := kube.CoreV1().ConfigMaps("default").Create(ctx, &cm, metav1.CreateOptions{})
	require.NoError(t, err)
	wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/trigger-wf-with-semaphore-and-mutex.yaml")
	syncLimitFunc := GetSyncLimitFunc(kube)

	syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		// nextKey = key
	}, WorkflowExistenceFunc, false)
	require.NoError(t, err)

	t.Run("InitializeMutex", func(t *testing.T) {
		tmpl := wf.Spec.Templates[1]
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "synchronization-tmpl-level-sgg6t-1949670081", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(msg)
		assert.Empty(failedLockName)
		assert.True(status)
		assert.True(wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		assert.NotNil(wf.Status.Synchronization.Mutex)
	})
	t.Run("InitializeSemaphore", func(t *testing.T) {
		tmpl := wf.Spec.Templates[2]
		status, wfUpdate, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "synchronization-tmpl-level-sgg6t-1899337224", tmpl.Synchronization)
		require.NoError(t, err)
		assert.Empty(msg)
		assert.Empty(failedLockName)
		assert.True(status)
		assert.True(wfUpdate)
		require.NotNil(t, wf.Status.Synchronization)
		require.NotNil(t, wf.Status.Synchronization.Semaphore)
	})
}

func TestMutexMigration(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	assert := assert.New(t)
	require := require.New(t)
	kube := fake.NewClientset()

	syncLimitFunc := GetSyncLimitFunc(kube)

	syncMgr, mgrErr := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
	}, WorkflowExistenceFunc, false)
	require.NoError(mgrErr)

	wfMutex := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-mutex.yaml")

	t.Run("RunMigrationWorkflowLevel", func(t *testing.T) {
		syncMgr.syncLockMap = make(map[string]semaphore)
		wfMutex2 := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-v2-mutex-migration-workflow-level.yaml")

		require.Len(wfMutex2.Status.Synchronization.Mutex.Holding, 1)
		holderKey := getHolderKey(wfMutex2, "")
		items := strings.Split(holderKey, "/")
		holdingName := items[len(items)-1]
		assert.Equal(wfMutex2.Status.Synchronization.Mutex.Holding[0].Holder, holdingName)

		syncMgr.syncLockMap = make(map[string]semaphore)
		wfs := []wfv1.Workflow{*wfMutex2.DeepCopy()}
		staleHolds, err := syncMgr.Initialize(ctx, wfs)
		require.NoError(err)
		require.Empty(staleHolds)

		syncItems, err := allSyncItems(wfMutex2.Spec.Synchronization)
		require.NoError(err)
		lockName, err := syncItems[0].lockName(wfMutex2.Namespace)
		require.NoError(err)

		sem, found := syncMgr.syncLockMap[lockName.String(ctx)]
		require.True(found)

		holders, err := sem.getCurrentHolders(ctx)
		require.NoError(err)
		require.Len(holders, 1)

		// PROVE: bug absent
		assert.Equal(holderKey, holders[0])

		// We should already have this lock since we acquired it above
		status, _, _, _, err := syncMgr.TryAcquire(ctx, wfMutex2, "", wfMutex.Spec.Synchronization)
		require.NoError(err)
		// BUG NOT PRESENT: https://github.com/argoproj/argo-workflows/issues/8684
		assert.True(status)
	})

	syncMgr, mgrErr = NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
	}, WorkflowExistenceFunc, false)
	require.NoError(mgrErr)

	t.Run("RunMigrationTemplateLevel", func(t *testing.T) {
		syncMgr.syncLockMap = make(map[string]semaphore)
		wfMutex3 := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-v2-mutex-migration-template-level.yaml")
		require.Len(wfMutex3.Status.Synchronization.Mutex.Holding, 1)

		numFound := 0
		foundNodeID := ""
		for nodeID := range wfMutex3.Status.Nodes {
			holder := getHolderKey(wfMutex3, nodeID)
			if holder == getUpgradedKey(wfMutex3, wfMutex3.Status.Synchronization.Mutex.Holding[0].Holder, TemplateLevel) {
				foundNodeID = nodeID
				numFound++
			}
		}
		assert.Equal(1, numFound)

		wfs := []wfv1.Workflow{*wfMutex3.DeepCopy()}
		staleHolds, err := syncMgr.Initialize(ctx, wfs)
		require.NoError(err)
		require.Empty(staleHolds)

		syncItems, err := allSyncItems(wfMutex3.Spec.Templates[1].Synchronization)
		require.NoError(err)
		lockName, err := syncItems[0].lockName(wfMutex3.Namespace)
		require.NoError(err)

		sem, found := syncMgr.syncLockMap[lockName.String(ctx)]
		require.True(found)

		holders, err := sem.getCurrentHolders(ctx)
		require.NoError(err)
		require.Len(holders, 1)

		holderKey := getHolderKey(wfMutex3, foundNodeID)

		// PROVE: bug absent
		assert.Equal(holderKey, holders[0])

		status, _, _, _, err := syncMgr.TryAcquire(ctx, wfMutex3, foundNodeID, wfMutex.Spec.Synchronization)
		require.NoError(err)
		// BUG NOT PRESENT: https://github.com/argoproj/argo-workflows/issues/8684
		assert.True(status)
	})
}

// getHoldingNameV1 legacy code to get holding name.
func getHoldingNameV1(holderKey string) string {
	items := strings.Split(holderKey, "/")
	return items[len(items)-1]
}

func TestCheckHolderVersion(t *testing.T) {
	t.Run("CheckHolderKeyWithNodeName", func(t *testing.T) {
		assert := assert.New(t)
		wfMutex := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-mutex.yaml")
		key := getHolderKey(wfMutex, wfMutex.Name)

		keyv2 := key
		version := wfv1.CheckHolderKeyVersion(keyv2)
		assert.Equal(wfv1.HoldingNameV2, version)

		keyv1 := getHoldingNameV1(key)
		version = wfv1.CheckHolderKeyVersion(keyv1)
		assert.Equal(wfv1.HoldingNameV1, version)
	})

	t.Run("CheckHolderKeyWithoutNodeName", func(t *testing.T) {
		assert := assert.New(t)
		wfMutex := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-mutex.yaml")

		key := getHolderKey(wfMutex, "")
		keyv2 := key
		version := wfv1.CheckHolderKeyVersion(keyv2)
		assert.Equal(wfv1.HoldingNameV2, version)

		keyv1 := getHoldingNameV1(key)
		version = wfv1.CheckHolderKeyVersion(keyv1)
		assert.Equal(wfv1.HoldingNameV1, version)
	})
}

func TestBackgroundNotifierClearsExpiredLocks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows platforms")
	}

	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			// Create database session and info
			info, deferfunc, _, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer deferfunc()

			// Set up two controllers, one active and one inactive
			activeController := "activeController"
			inactiveController := "inactiveController"

			// Insert controller records - one fresh, one stale
			now := time.Now()
			staleTime := now.Add(-info.Config.InactiveControllerTimeout * 2) // Double the inactive timeout

			_, err = info.SessionProxy.Session(ctx).Collection(info.Config.ControllerTable).Insert(&syncdb.ControllerHealthRecord{
				Controller: activeController,
				Time:       now,
			})
			require.NoError(t, err)

			_, err = info.SessionProxy.Session(ctx).Collection(info.Config.ControllerTable).Insert(&syncdb.ControllerHealthRecord{
				Controller: inactiveController,
				Time:       staleTime,
			})
			require.NoError(t, err)

			// Create a lock with an active controller and one with an inactive controller
			lockName1 := "test-lock-active"
			lockName2 := "test-lock-inactive"

			_, err = info.SessionProxy.Session(ctx).Collection(info.Config.LockTable).Insert(&syncdb.LockRecord{
				Name:       lockName1,
				Controller: activeController,
				Time:       now,
			})
			require.NoError(t, err)

			_, err = info.SessionProxy.Session(ctx).Collection(info.Config.LockTable).Insert(&syncdb.LockRecord{
				Name:       lockName2,
				Controller: inactiveController,
				Time:       now, // Time doesn't matter, controller is what matters
			})
			require.NoError(t, err)

			_, err = info.SessionProxy.Session(ctx).SQL().Exec("INSERT INTO sync_limit (name, sizelimit) VALUES (?, ?)", "foo/test-semaphore", 100)
			require.NoError(t, err)
			// Initialize a semaphore so it gets added to the syncLockMap
			testsem, err := newDatabaseSemaphore(ctx, "test-semaphore", "foo/test-semaphore", func(key string) {}, info, 0)
			require.NoError(t, err)
			syncLockMap := make(map[string]semaphore)
			syncLockMap["sem/test-semaphore"] = testsem

			// Verify both lock records exist initially
			lockCount, err := info.SessionProxy.Session(ctx).Collection(info.Config.LockTable).Count()
			require.NoError(t, err)
			assert.Equal(t, uint64(2), lockCount, "Should have two lock records initially")

			// Run the background notifier manually once
			for _, lock := range syncLockMap {
				lock.probeWaiting(ctx)
			}

			// Check that only the active controller's lock remains
			var remainingLocks []syncdb.LockRecord
			err = info.SessionProxy.Session(ctx).SQL().Select("*").From(info.Config.LockTable).All(&remainingLocks)
			require.NoError(t, err)

			assert.Len(t, remainingLocks, 1, "Should have one lock record remaining")
			if len(remainingLocks) > 0 {
				assert.Equal(t, lockName1, remainingLocks[0].Name, "Active controller's lock should remain")
				assert.Equal(t, activeController, remainingLocks[0].Controller, "Active controller's lock should remain")
			}
		})
	}
}

func TestUnconfiguredSemaphores(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	kube := fake.NewClientset()
	t.Run("UnconfiguredConfigMapSemaphore", func(t *testing.T) {
		// Setup with a fake k8s client but no ConfigMap created
		syncLimitFunc := GetSyncLimitFunc(kube)
		syncManager, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		// Create a workflow with a semaphore referencing a non-existent ConfigMap
		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-semaphore.yaml")

		// Try to acquire the lock
		status, _, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

		// Assertions - expect an error because ConfigMap doesn't exist
		require.Error(t, err)
		assert.False(t, status)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/ConfigMap/my-config/workflow", failedLockName)
		assert.Contains(t, err.Error(), "failed to initialize semaphore")
	})

	t.Run("UnavailableDatabaseSemaphore", func(t *testing.T) {
		// Don't use testDBTypes here, as we can test this on windows
		for _, dbType := range []sqldb.DBType{sqldb.Postgres, sqldb.MySQL} {
			t.Run(string(dbType), func(t *testing.T) {
				// Create appropriate invalid config for the database type
				var syncConfig *config.SyncConfig
				switch dbType {
				case sqldb.Postgres:
					syncConfig = &config.SyncConfig{
						DBConfig: config.DBConfig{
							PostgreSQL: &config.PostgreSQLConfig{
								DatabaseConfig: config.DatabaseConfig{
									Host:     "non-existent-host",
									Port:     5432,
									Database: "non-existent-db",
								},
								SSL: false,
							},
						},
					}
				case sqldb.MySQL:
					syncConfig = &config.SyncConfig{
						DBConfig: config.DBConfig{
							MySQL: &config.MySQLConfig{
								DatabaseConfig: config.DatabaseConfig{
									Host:     "non-existent-host",
									Port:     3306,
									Database: "non-existent-db",
								},
							},
						},
					}
				}

				syncManager, err := NewLockManager(ctx, kube, "", syncConfig, nil, func(key string) {
				}, WorkflowExistenceFunc, false)
				require.NoError(t, err)

				// Create a workflow with a database semaphore
				wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-db-semaphore.yaml")

				// Try to acquire the lock
				status, _, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

				// Assertions - expect it to fail because DB connection will fail
				require.Error(t, err)
				assert.False(t, status)
				assert.NotEmpty(t, msg)
				assert.Equal(t, "default/Database/my-database-sem", failedLockName)
				assert.Contains(t, err.Error(), "database session is not available")
			})
		}
	})

	t.Run("UnconfiguredDBSemaphore", func(t *testing.T) {
		// Setup a LockManager with no database configuration. This doesn't need to distinguish between Postgres and MySQL, neither are configured
		syncConfig := &config.SyncConfig{}

		syncManager, err := NewLockManager(ctx, kube, "", syncConfig, nil, func(key string) {
		}, WorkflowExistenceFunc, false)
		require.NoError(t, err)

		// Create a workflow with a database semaphore
		wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-db-semaphore.yaml")

		// Try to acquire the lock
		status, _, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

		// Assertions - expect it to fail because DB is not configured
		require.Error(t, err)
		assert.False(t, status)
		assert.NotEmpty(t, msg)
		assert.Equal(t, "default/Database/my-database-sem", failedLockName)
		assert.Contains(t, err.Error(), "database session is not available")
	})

	t.Run("MissingLimitSemaphore", func(t *testing.T) {
		for _, dbType := range testDBTypes {
			t.Run(string(dbType), func(t *testing.T) {
				// Setup test database using helper
				info, cleanup, syncConfig, err := createTestDBSession(ctx, t, dbType)
				require.NoError(t, err)
				defer cleanup()

				// Configure sync manager
				syncManager := createLockManager(ctx, info.SessionProxy, &syncConfig, nil, func(key string) {
				}, WorkflowExistenceFunc)
				require.NotNil(t, syncManager)
				require.NotNil(t, syncManager.dbInfo.SessionProxy.Session(ctx))

				// Create a workflow with a database semaphore
				wf := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-db-semaphore.yaml")

				// Try to acquire the lock
				status, _, msg, failedLockName, err := syncManager.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)

				// Assertions - expect it to fail because limit is not in the table
				require.Error(t, err)
				assert.False(t, status)
				assert.NotEmpty(t, msg)
				assert.Equal(t, "default/Database/my-database-sem", failedLockName)
				assert.Contains(t, err.Error(), "failed to initialize semaphore")
			})
		}
	})
}

// TestDatabaseInitializeLoweredLimitAfterRestart drives the full restart path
// through Manager.Initialize for a database-backed semaphore.
//
// Scenario: a controller acquires a database semaphore for two workflows under
// limit 2, the controller is turned off, the limit is lowered to 1, and the
// controller restarts while both workflows are still Running. On restart
// Initialize re-establishes the recorded holders from the workflows' status.
//
// For a database semaphore the holds live in durable rows, so the holder count
// survives the restart regardless of the lowered limit. We assert that:
//   - the lock is live (not poisoned - lowering a limit is routine);
//   - both holders are still counted, even over the lowered limit;
//   - each still-running workflow keeps its lock when it re-reconciles;
//   - a new contender waits until the over-subscription drains below the limit.
func TestDatabaseInitializeLoweredLimitAfterRestart(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			const dbLimitKey = "default/my-database-sem"
			const lockName = "default/Database/my-database-sem"

			info, cleanup, syncConfig, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer cleanup()

			// Original limit: 2.
			_, err = info.SessionProxy.Session(ctx).SQL().Exec("INSERT INTO sync_limit (name, sizelimit) VALUES (?, ?)", dbLimitKey, 2)
			require.NoError(t, err)

			// --- Controller instance #1: two workflows acquire the semaphore. ---
			mgr1 := createLockManager(ctx, info.SessionProxy, &syncConfig, nil, func(key string) {}, WorkflowExistenceFunc)

			creationTime := metav1.NewTime(time.Now())
			wf1 := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-db-semaphore.yaml")
			wf1.Name = "holder-one"
			wf1.CreationTimestamp = creationTime
			wf2 := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-db-semaphore.yaml")
			wf2.Name = "holder-two"
			wf2.CreationTimestamp = creationTime

			acquired, _, _, _, err := mgr1.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
			require.NoError(t, err)
			require.True(t, acquired, "holder-one should acquire under limit 2")
			acquired, _, _, _, err = mgr1.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
			require.NoError(t, err)
			require.True(t, acquired, "holder-two should acquire under limit 2")

			// Both workflows now record the hold in their status - this is what the
			// informer persists and re-lists into Initialize after a restart.
			require.NotNil(t, wf1.Status.Synchronization)
			require.NotNil(t, wf2.Status.Synchronization)

			// --- Limit lowered to 1 while the controller is down. ---
			_, err = info.SessionProxy.Session(ctx).SQL().Exec("UPDATE sync_limit SET sizelimit = ? WHERE name = ?", 1, dbLimitKey)
			require.NoError(t, err)

			// --- Controller instance #2: restart. Initialize from the running workflows. ---
			mgr2 := createLockManager(ctx, info.SessionProxy, &syncConfig, nil, func(key string) {}, WorkflowExistenceFunc)
			staleHolds, err := mgr2.Initialize(ctx, []wfv1.Workflow{*wf1, *wf2})
			require.NoError(t, err)
			require.Empty(t, staleHolds, "holders backed by database rows must not be reported stale")

			lock := mgr2.syncLockMap[lockName]
			require.NotNil(t, lock)
			_, poisoned := lock.(*poisonedLock)
			assert.False(t, poisoned, "lowering a DB semaphore limit must not poison the lock")

			holders, err := lock.getCurrentHolders(ctx)
			require.NoError(t, err)
			assert.Len(t, holders, 2, "both recorded holders must survive the restart, even over the lowered limit")

			// Each still-running workflow re-reconciles and re-acquires the lock it
			// already holds. Both must succeed regardless of the lowered limit, since
			// the already-held check precedes the limit check.
			acquired, _, _, _, err = mgr2.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
			require.NoError(t, err)
			assert.True(t, acquired, "holder-one must keep its lock after the limit was lowered")
			acquired, _, _, _, err = mgr2.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
			require.NoError(t, err)
			assert.True(t, acquired, "holder-two must keep its lock after the limit was lowered")

			// A new contender must wait: holders (2) exceed the lowered limit (1).
			racer := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-db-semaphore.yaml")
			racer.Name = "racer"
			racer.CreationTimestamp = creationTime
			acquired, _, _, _, err = mgr2.TryAcquire(ctx, racer, "", racer.Spec.Synchronization)
			require.NoError(t, err)
			assert.False(t, acquired, "a racer must wait while holders exceed the lowered limit")

			// Drain one holder. Still at the limit (1 holder, limit 1) - racer waits.
			mgr2.Release(ctx, wf1, "", wf1.Spec.Synchronization)
			acquired, _, _, _, err = mgr2.TryAcquire(ctx, racer, "", racer.Spec.Synchronization)
			require.NoError(t, err)
			assert.False(t, acquired, "racer still waits: one holder remains at limit 1")

			// Drain the last holder - now below the limit, so the racer is admitted.
			mgr2.Release(ctx, wf2, "", wf2.Spec.Synchronization)
			acquired, _, _, _, err = mgr2.TryAcquire(ctx, racer, "", racer.Spec.Synchronization)
			require.NoError(t, err)
			assert.True(t, acquired, "racer acquires once holders drain below the lowered limit")
		})
	}
}

// TestDatabaseInitializeStaleHoldFailsWorkflow drives the restart path through
// Manager.Initialize for a database-backed semaphore whose held row has gone
// away while the controller was down (e.g. expired by ExpireInactiveLocks and
// possibly acquired by another holder since).
//
// The database is the single source of truth for a database-backed lock, so
// Initialize must not resurrect the hold: it asserts the row exists, and when
// it does not it reports the workflow as a StaleHold for the controller to
// fail. We assert that:
//   - the stale workflow is reported, with a reason naming the lock;
//   - the lock is live (not poisoned) - the database still arbitrates it;
//   - the hold is not re-inserted into the database;
//   - a workflow whose row survived is re-established normally, not reported.
func TestDatabaseInitializeStaleHoldFailsWorkflow(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			const dbLimitKey = "default/my-database-sem"
			const lockName = "default/Database/my-database-sem"

			info, cleanup, syncConfig, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer cleanup()

			_, err = info.SessionProxy.Session(ctx).SQL().Exec("INSERT INTO sync_limit (name, sizelimit) VALUES (?, ?)", dbLimitKey, 2)
			require.NoError(t, err)

			// --- Controller instance #1: two workflows acquire the semaphore. ---
			mgr1 := createLockManager(ctx, info.SessionProxy, &syncConfig, nil, func(key string) {}, WorkflowExistenceFunc)

			creationTime := metav1.NewTime(time.Now())
			wf1 := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-db-semaphore.yaml")
			wf1.Name = "stale-holder"
			wf1.CreationTimestamp = creationTime
			wf2 := wfv1.MustUnmarshalWorkflow("@testdata/sync_manager/wf-with-db-semaphore.yaml")
			wf2.Name = "live-holder"
			wf2.CreationTimestamp = creationTime

			acquired, _, _, _, err := mgr1.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
			require.NoError(t, err)
			require.True(t, acquired, "stale-holder should acquire under limit 2")
			acquired, _, _, _, err = mgr1.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
			require.NoError(t, err)
			require.True(t, acquired, "live-holder should acquire under limit 2")
			require.NotNil(t, wf1.Status.Synchronization)
			require.NotNil(t, wf2.Status.Synchronization)

			// --- wf1's held row disappears while the controller is down. ---
			_, err = info.SessionProxy.Session(ctx).SQL().Exec("DELETE FROM sync_state WHERE name = ? AND workflowkey = ?", "sem/"+dbLimitKey, "default/stale-holder")
			require.NoError(t, err)

			// --- Controller instance #2: restart. Initialize from the running workflows. ---
			mgr2 := createLockManager(ctx, info.SessionProxy, &syncConfig, nil, func(key string) {}, WorkflowExistenceFunc)
			staleHolds, err := mgr2.Initialize(ctx, []wfv1.Workflow{*wf1, *wf2})
			require.NoError(t, err, "a stale hold is not fatal: the workflow is failed, not the controller")

			require.Len(t, staleHolds, 1, "only the workflow whose row is gone is stale")
			assert.Equal(t, "stale-holder", staleHolds[0].WF.Name)
			assert.Contains(t, staleHolds[0].Reason, lockName)

			// The lock stays live: the database arbitrates it, and the live holder
			// (and any future contender) must not be blocked by wf1's stale status.
			lock := mgr2.syncLockMap[lockName]
			require.NotNil(t, lock)
			_, poisoned := lock.(*poisonedLock)
			assert.False(t, poisoned, "a stale hold must not poison the lock")

			// The stale hold was asserted, not re-acquired: only the surviving row
			// is held.
			holders, err := lock.getCurrentHolders(ctx)
			require.NoError(t, err)
			assert.Equal(t, []string{"default/live-holder"}, holders, "the stale hold must not be re-inserted into the database")

			// The surviving holder keeps its lock when it re-reconciles.
			acquired, _, _, _, err = mgr2.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
			require.NoError(t, err)
			assert.True(t, acquired, "live-holder must keep its lock")
		})
	}
}
