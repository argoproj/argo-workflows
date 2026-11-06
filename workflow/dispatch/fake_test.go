package dispatch

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TestFakeWatchReportsAnEarlierLoss checks that a watch started after its
// claim was replaced reports it lost at once, as Watches reports what its
// cache already shows.
func TestFakeWatchReportsAnEarlierLoss(t *testing.T) {
	b := newFakeBackend()
	ctx := logging.TestContext(t.Context())
	task := testTask("node-1")
	b.enqueue(t, task)
	h := b.claim(t, testWorker("worker-a"), task.Key)
	require.NoError(t, h.Release(ctx))
	b.claim(t, testWorker("worker-b"), task.Key)
	stop := newStopSignal()
	cancel := fakeWatcher{fake: b.fake}.Watch(ctx, h.Claim(), stop.stop)
	defer cancel()
	select {
	case <-stop.done:
		assert.Equal(t, StopLost, stop.reason())
	default:
		t.Error("replacement not reported")
	}
}

func TestFakeFaultStagesARace(t *testing.T) {
	b := newFakeBackend()
	f := b.fake
	task := testTask("node-1")
	ts := b.enqueue(t, task)
	candidate := b.candidate(t, task.Key)
	// The node is dropped between the worker creating its claim and reading
	// the TaskSet.
	var dropped *Claim
	f.SetFault(func(op Op) error {
		if op.Verb == "get" && op.Resource == ResourceTaskSets {
			f.SetFault(nil)
			var err error
			dropped, err = f.Dispatcher().Drop(logging.TestContext(t.Context()), task.Key, ts.UID)
			require.NoError(t, err)
		}
		return nil
	})
	_, err := b.consumer(testWorker("worker-a")).Claim(logging.TestContext(t.Context()), candidate)
	require.ErrorIs(t, err, ErrTaskGone)
	require.NotNil(t, dropped, "the drop found the claim")
	_, ok := f.Result(testNamespace, "node-1")
	assert.False(t, ok, "the worker withdrew it")
}

func TestFakeFaultFailsCalls(t *testing.T) {
	f := NewFake()
	injected := errors.New("injected")
	f.SetFault(func(op Op) error {
		if op.Verb == "create" && op.Resource == ResourceTaskSets {
			return injected
		}
		return nil
	})
	_, err := f.Dispatcher().Enqueue(logging.TestContext(t.Context()), testTask("node-1"))
	require.ErrorIs(t, err, injected)
	assert.Empty(t, f.TaskSets(testNamespace))
}

func TestFakeCreateTaskSet(t *testing.T) {
	f := NewFake()
	agent := &wfv1.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{Namespace: testNamespace, Name: testTask("node-1").TaskSetName()}}
	stored, err := f.CreateTaskSet(agent)
	require.NoError(t, err)
	assert.NotEmpty(t, stored.UID)
	_, err = f.CreateTaskSet(agent)
	require.Error(t, err, "the name is taken")

	_, err = f.Dispatcher().Enqueue(logging.TestContext(t.Context()), testTask("node-1"))
	require.ErrorIs(t, err, ErrNameTaken)
	got, ok := f.TaskSet(testNamespace, agent.Name)
	require.True(t, ok)
	assert.Equal(t, stored.UID, got.UID, "the other client's TaskSet is untouched")
}

func TestFakeClock(t *testing.T) {
	f := NewFake()
	at := time.Date(2026, 1, 2, 3, 4, 5, 600, time.UTC)
	f.SetClock(func() time.Time { return at })
	enqueued, err := f.Dispatcher().Enqueue(logging.TestContext(t.Context()), testTask("node-1"))
	require.NoError(t, err)
	assert.Equal(t, at.Truncate(time.Second), enqueued.TaskSet.CreationTimestamp.UTC())
}

func TestFakePoolHealth(t *testing.T) {
	f := NewFake()
	var health PoolHealth = f
	_, ok := health.Health(testNamespace, "hash-1")
	assert.False(t, ok)
	f.SetHealth(testNamespace, "hash-1", Health{State: HealthDegraded, Reason: "ImagePullBackOff"})
	got, ok := health.Health(testNamespace, "hash-1")
	require.True(t, ok)
	assert.Equal(t, Health{State: HealthDegraded, Reason: "ImagePullBackOff"}, got)
	_, ok = health.Health("other", "hash-1")
	assert.False(t, ok, "pools are per namespace")
}
