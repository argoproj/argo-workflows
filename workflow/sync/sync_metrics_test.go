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

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	wfmetrics "github.com/argoproj/argo-workflows/v4/workflow/metrics"
)

// takenLabels captures the full label set of a single locks_taken_total increment so tests can
// assert not just the count but that the type/storage/name/namespace labels are correct.
type takenLabels struct {
	lockType, storage, name, namespace string
}

// testMetricsRecorder records locks_taken_total increments for assertions. It implements the
// syncMetrics interface. The held/pending gauges are asserted via Manager.LockMetrics directly.
type testMetricsRecorder struct {
	taken []takenLabels
}

func newTestMetricsRecorder() *testMetricsRecorder {
	return &testMetricsRecorder{}
}

func (t *testMetricsRecorder) RecordLockTaken(_ context.Context, lockType, storage, name, namespace string) {
	t.taken = append(t.taken, takenLabels{lockType, storage, name, namespace})
}

// total returns the number of recorded lock acquisitions across all labels.
func (t *testMetricsRecorder) total() int { return len(t.taken) }

// count returns how many acquisitions were recorded with exactly the given labels.
func (t *testMetricsRecorder) count(want takenLabels) int {
	n := 0
	for _, r := range t.taken {
		if r == want {
			n++
		}
	}
	return n
}

func newTestManagerWithMetrics(ctx context.Context, t *testing.T) (*Manager, *testMetricsRecorder) {
	t.Helper()
	kube := fake.NewSimpleClientset()
	syncLimitFunc := func(ctx context.Context, s string) (int, error) { return 1, nil }
	mgr, err := NewLockManager(ctx, kube, "", nil, syncLimitFunc, func(string) {}, func(string) bool { return false }, false)
	require.NoError(t, err)
	rec := newTestMetricsRecorder()
	mgr.metrics = rec
	return mgr, rec
}

// findLockSample returns the gauge sample for the lock with the given name, or a zero sample.
func findLockSample(samples []wfmetrics.LockGaugeSample, name string) wfmetrics.LockGaugeSample {
	for _, s := range samples {
		if s.Name == name {
			return s
		}
	}
	return wfmetrics.LockGaugeSample{}
}

func TestSyncMetrics(t *testing.T) {
	ctx := logging.TestContext(context.Background())

	t.Run("MutexAcquireRelease", func(t *testing.T) {
		mgr, rec := newTestManagerWithMetrics(ctx, t)

		wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf-mutex", Namespace: "default"}}
		wf.CreationTimestamp = metav1.Time{Time: time.Now()}
		wf.Spec.Synchronization = &wfv1.Synchronization{Mutexes: []*wfv1.Mutex{{Name: "my-mutex"}}}

		acquired, updated, _, _, err := mgr.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, acquired, "expected mutex to be acquired")
		assert.True(t, updated, "expected workflow status to be updated")
		assert.Equal(t, 1, rec.total(), "expected 1 lock taken")
		assert.Equal(t, 1, rec.count(takenLabels{"mutex", "configmap", "my-mutex", "default"}), "counter must carry the correct labels")

		held := findLockSample(mgr.LockMetrics(ctx), "my-mutex")
		assert.Equal(t, "mutex", held.Type)
		assert.Equal(t, "configmap", held.Storage)
		assert.Equal(t, int64(1), held.Held, "expected mutex to be held")
		assert.Equal(t, int64(0), held.Pending, "expected no pending")

		mgr.Release(ctx, wf, "", wf.Spec.Synchronization)
		released := findLockSample(mgr.LockMetrics(ctx), "my-mutex")
		assert.Equal(t, int64(0), released.Held, "expected mutex to be released")
		assert.Equal(t, 1, rec.total(), "taken counter must not decrement on release")
	})

	t.Run("MutexWaitingIsPendingNotHeld", func(t *testing.T) {
		mgr, rec := newTestManagerWithMetrics(ctx, t)

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

		s := findLockSample(mgr.LockMetrics(ctx), "wait-mutex")
		assert.Equal(t, int64(1), s.Held, "one holder")
		assert.Equal(t, int64(1), s.Pending, "one waiter")
		assert.Equal(t, 1, rec.total(), "only wf1 acquired, so 1 lock taken")
	})

	t.Run("SemaphoreAcquireRelease", func(t *testing.T) {
		mgr, rec := newTestManagerWithMetrics(ctx, t)

		// A ConfigMap semaphore's lock name is "<configMapName>/<key>" so that multiple keys in one
		// ConfigMap remain distinct metric series.
		semName := "my-config/workflow"
		wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf-sem", Namespace: "default"}}
		wf.CreationTimestamp = metav1.Time{Time: time.Now()}
		wf.Spec.Synchronization = &wfv1.Synchronization{
			Semaphores: []*wfv1.SemaphoreRef{{
				Namespace: "default",
				ConfigMapKeyRef: &apiv1.ConfigMapKeySelector{
					LocalObjectReference: apiv1.LocalObjectReference{Name: "my-config"},
					Key:                  "workflow",
				},
			}},
		}

		acquired, updated, _, _, err := mgr.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, acquired, "expected semaphore to be acquired")
		assert.True(t, updated, "expected workflow status to be updated")
		assert.Equal(t, 1, rec.total(), "expected 1 lock taken")
		assert.Equal(t, 1, rec.count(takenLabels{"semaphore", "configmap", semName, "default"}), "counter must carry the correct labels")

		held := findLockSample(mgr.LockMetrics(ctx), semName)
		assert.Equal(t, "semaphore", held.Type)
		assert.Equal(t, "configmap", held.Storage)
		assert.Equal(t, int64(1), held.Held, "expected semaphore to be held")

		mgr.Release(ctx, wf, "", wf.Spec.Synchronization)
		released := findLockSample(mgr.LockMetrics(ctx), semName)
		assert.Equal(t, int64(0), released.Held, "expected semaphore to be released")
	})

	t.Run("ReleaseAllMixed", func(t *testing.T) {
		mgr, rec := newTestManagerWithMetrics(ctx, t)

		semName := "my-config/workflow"
		wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf-mixed", Namespace: "default"}}
		wf.CreationTimestamp = metav1.Time{Time: time.Now()}
		wf.Spec.Synchronization = &wfv1.Synchronization{
			Mutexes: []*wfv1.Mutex{{Name: "mixed-mutex"}},
			Semaphores: []*wfv1.SemaphoreRef{{
				Namespace: "default",
				ConfigMapKeyRef: &apiv1.ConfigMapKeySelector{
					LocalObjectReference: apiv1.LocalObjectReference{Name: "my-config"},
					Key:                  "workflow",
				},
			}},
		}

		acquired, updated, _, _, err := mgr.TryAcquire(ctx, wf, "", wf.Spec.Synchronization)
		require.NoError(t, err)
		assert.True(t, acquired, "expected mixed locks to be acquired")
		assert.True(t, updated, "expected workflow status to be updated")
		assert.Equal(t, 2, rec.total(), "expected 2 locks taken (mutex + semaphore)")
		assert.Equal(t, 1, rec.count(takenLabels{"mutex", "configmap", "mixed-mutex", "default"}), "mutex counted with correct labels")
		assert.Equal(t, 1, rec.count(takenLabels{"semaphore", "configmap", semName, "default"}), "semaphore counted with correct labels")

		samples := mgr.LockMetrics(ctx)
		assert.Equal(t, int64(1), findLockSample(samples, "mixed-mutex").Held)
		assert.Equal(t, int64(1), findLockSample(samples, semName).Held)

		mgr.ReleaseAll(ctx, wf)
		samples = mgr.LockMetrics(ctx)
		assert.Equal(t, int64(0), findLockSample(samples, "mixed-mutex").Held, "expected mutex released after ReleaseAll")
		assert.Equal(t, int64(0), findLockSample(samples, semName).Held, "expected semaphore released after ReleaseAll")
	})
}

func TestParseLockKey(t *testing.T) {
	cases := []struct {
		in, namespace, name, storage string
		ok                           bool
	}{
		{in: "default/Mutex/my-mutex", namespace: "default", name: "my-mutex", storage: "configmap", ok: true},
		// ConfigMap semaphores keep the key so distinct keys in one ConfigMap stay distinct.
		{in: "default/ConfigMap/my-config/workflow", namespace: "default", name: "my-config/workflow", storage: "configmap", ok: true},
		{in: "default/ConfigMap/my-config/template", namespace: "default", name: "my-config/template", storage: "configmap", ok: true},
		{in: "ns1/Database/my-db-lock", namespace: "ns1", name: "my-db-lock", storage: "database", ok: true},
		{in: "bad", ok: false},
		{in: "ns/only", ok: false},
		{in: "ns/Unknown/thing", ok: false},
	}
	for _, c := range cases {
		namespace, name, storage, ok := parseLockKey(c.in)
		assert.Equal(t, c.ok, ok, c.in)
		if c.ok {
			assert.Equal(t, c.namespace, namespace, c.in)
			assert.Equal(t, c.name, name, c.in)
			assert.Equal(t, c.storage, storage, c.in)
		}
	}
}

func TestParseDBStateName(t *testing.T) {
	cases := []struct {
		in, lockType, name, namespace string
		ok                            bool
	}{
		{in: "sem/default/my-sem", lockType: "semaphore", name: "my-sem", namespace: "default", ok: true},
		{in: "mtx/ns1/my-mutex", lockType: "mutex", name: "my-mutex", namespace: "ns1", ok: true},
		{in: "bad", ok: false},
		{in: "xxx/ns/name", ok: false},
		{in: "sem/onlyns", ok: false},
	}
	for _, c := range cases {
		lockType, name, namespace, ok := parseDBStateName(c.in)
		assert.Equal(t, c.ok, ok, c.in)
		if c.ok {
			assert.Equal(t, c.lockType, lockType, c.in)
			assert.Equal(t, c.name, name, c.in)
			assert.Equal(t, c.namespace, namespace, c.in)
		}
	}
}

// TestLockMetricsDatabase exercises the controller-scoped aggregate query end-to-end against a real
// database: two workflows hold a limit-2 semaphore and a third waits.
func TestLockMetricsDatabase(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	for _, dbType := range testDBTypes {
		t.Run(string(dbType), func(t *testing.T) {
			const dbLimitKey = "default/my-database-sem"

			info, cleanup, syncConfig, err := createTestDBSession(ctx, t, dbType)
			require.NoError(t, err)
			defer cleanup()

			_, err = info.SessionProxy.Session().SQL().Exec("INSERT INTO sync_limit (name, sizelimit) VALUES (?, ?)", dbLimitKey, 2)
			require.NoError(t, err)

			mgr := createLockManager(ctx, info.SessionProxy, &syncConfig, nil, func(key string) {}, WorkflowExistenceFunc)

			creationTime := metav1.NewTime(time.Now())
			newWF := func(name string) *wfv1.Workflow {
				wf := wfv1.MustUnmarshalWorkflow(wfWithDBSemaphore)
				wf.Name = name
				wf.CreationTimestamp = creationTime
				return wf
			}

			for _, name := range []string{"holder-one", "holder-two"} {
				acquired, _, _, _, acqErr := mgr.TryAcquire(ctx, newWF(name), "", newWF(name).Spec.Synchronization)
				require.NoError(t, acqErr)
				require.True(t, acquired, "%s should acquire under limit 2", name)
			}
			acquired, _, _, _, err := mgr.TryAcquire(ctx, newWF("waiter"), "", newWF("waiter").Spec.Synchronization)
			require.NoError(t, err)
			require.False(t, acquired, "third workflow should wait under limit 2")

			// Use a bare context (no logger) to mimic the metrics scrape callback: the database query
			// path calls RequireLoggerFromContext, so LockMetrics must attach its own logger or panic.
			//nolint:contextcheck // deliberately logger-less to reproduce the scrape-time context
			s := findLockSample(mgr.LockMetrics(context.Background()), "my-database-sem")
			assert.Equal(t, "semaphore", s.Type)
			assert.Equal(t, "database", s.Storage)
			assert.Equal(t, "default", s.Namespace)
			assert.Equal(t, int64(2), s.Held, "two holders")
			assert.Equal(t, int64(1), s.Pending, "one waiter")
		})
	}
}
