package sync

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func newTestManagerWithMetrics(ctx context.Context) (*Manager, *testMetricsRecorder) {
	kube := fake.NewSimpleClientset()
	syncLimitFunc := func(ctx context.Context, s string) (int, error) { return 1, nil }
	mgr := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(string) {}, func(string) bool { return false })
	rec := newTestMetricsRecorder()
	mgr.metrics = rec
	return mgr, rec
}

func TestSyncMetrics(t *testing.T) {
	ctx := logging.TestContext(context.Background())

	t.Run("MutexAcquireRelease", func(t *testing.T) {
		mgr, rec := newTestManagerWithMetrics(ctx)

		wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf-mutex", Namespace: "default"}}
		wf.CreationTimestamp = metav1.Time{Time: time.Now()}
		wf.Spec.Synchronization = &wfv1.Synchronization{Mutexes: []*wfv1.Mutex{{Name: "my-mutex"}}}

		acquired, updated, _, _, err := mgr.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, acquired, "expected mutex to be acquired")
		assert.True(t, updated, "expected workflow status to be updated")
		assert.Equal(t, 1, sum(rec.mutexAdds), "expected 1 mutex add")

		mgr.Release(ctx, wf, "", wf.Spec.Synchronization)
		assert.Equal(t, 1, sum(rec.mutexRemoves), "expected 1 mutex remove")
	})

	t.Run("MutexWaitingDoesNotIncrement", func(t *testing.T) {
		mgr, rec := newTestManagerWithMetrics(ctx)

		wf1 := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wfA", Namespace: "default"}}
		wf1.CreationTimestamp = metav1.Time{Time: time.Now()}
		wf1.Spec.Synchronization = &wfv1.Synchronization{Mutexes: []*wfv1.Mutex{{Name: "wait-mutex"}}}
		acquired, _, _, _, err := mgr.TryAcquire(ctx, wf1, "", wf1.Spec.Synchronization)
		require.NoError(t, err)
		require.True(t, acquired, "wf1 should acquire lock")

		wf2 := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wfB", Namespace: "default"}}
		wf2.CreationTimestamp = metav1.Time{Time: time.Now().Add(time.Second)}
		wf2.Spec.Synchronization = &wfv1.Synchronization{Mutexes: []*wfv1.Mutex{{Name: "wait-mutex"}}}
		acquired2, _, _, _, _ := mgr.TryAcquire(ctx, wf2, "", wf2.Spec.Synchronization)
		assert.False(t, acquired2, "wf2 should be waiting, not acquired")
		assert.Equal(t, 1, sum(rec.mutexAdds), "expected only 1 mutex add (waiting should not increment)")
	})

	t.Run("SemaphoreAcquireRelease", func(t *testing.T) {
		mgr, rec := newTestManagerWithMetrics(ctx)

		semName := "my-config"
		wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf-sem", Namespace: "default"}}
		wf.CreationTimestamp = metav1.Time{Time: time.Now()}
		wf.Spec.Synchronization = &wfv1.Synchronization{
			Semaphores: []*wfv1.SemaphoreRef{{
				Namespace: "default",
				ConfigMapKeyRef: &apiv1.ConfigMapKeySelector{
					LocalObjectReference: apiv1.LocalObjectReference{Name: semName},
					Key:                  "workflow",
				},
			}},
		}

		acquired, updated, _, _, err := mgr.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, acquired, "expected semaphore to be acquired")
		assert.True(t, updated, "expected workflow status to be updated")
		assert.Equal(t, 1, sum(rec.semaphoreAdds), "expected 1 semaphore add")

		mgr.Release(ctx, wf, "", wf.Spec.Synchronization)
		assert.Equal(t, 1, sum(rec.semaphoreRemoves), "expected 1 semaphore remove")
	})

	t.Run("ReleaseAllMixed", func(t *testing.T) {
		mgr, rec := newTestManagerWithMetrics(ctx)

		semName := "my-config"
		wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf-mixed", Namespace: "default"}}
		wf.CreationTimestamp = metav1.Time{Time: time.Now()}
		wf.Spec.Synchronization = &wfv1.Synchronization{
			Mutexes: []*wfv1.Mutex{{Name: "mixed-mutex"}},
			Semaphores: []*wfv1.SemaphoreRef{{
				Namespace: "default",
				ConfigMapKeyRef: &apiv1.ConfigMapKeySelector{
					LocalObjectReference: apiv1.LocalObjectReference{Name: semName},
					Key:                  "workflow",
				},
			}},
		}

		acquired, updated, _, _, err := mgr.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, acquired, "expected mixed locks to be acquired")
		assert.True(t, updated, "expected workflow status to be updated")
		assert.Equal(t, 1, sum(rec.mutexAdds), "expected 1 mutex add")
		assert.Equal(t, 1, sum(rec.semaphoreAdds), "expected 1 semaphore add")

		mgr.ReleaseAll(ctx, wf)
		assert.Equal(t, 1, sum(rec.mutexRemoves), "expected 1 mutex remove after ReleaseAll")
		assert.Equal(t, 1, sum(rec.semaphoreRemoves), "expected 1 semaphore remove after ReleaseAll")
	})
}
