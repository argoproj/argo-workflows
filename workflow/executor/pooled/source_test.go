package pooled

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/mocks"
)

const testNamespace = "argo"

func testTask(nodeID string) dispatch.Task {
	return dispatch.Task{
		Key:          dispatch.Key{Namespace: testNamespace, NodeID: nodeID},
		WorkflowName: "my-wf",
		WorkflowUID:  "wf-uid",
		PoolHash:     "hash-1",
		Template: wfv1.Template{
			Name:      "main",
			Container: &apiv1.Container{Image: "busybox", Command: []string{"echo"}},
		},
		Pool:     wfv1.PoolSpec{Name: "my-pool", InitialReplicas: 1},
		Dispatch: wfv1.TaskDispatch{TaskName: "my-wf-main-" + nodeID},
	}
}

func testWorker(name string) dispatch.Worker {
	return dispatch.Worker{Namespace: testNamespace, PoolHash: "hash-1", PodName: name, PodUID: types.UID(name + "-uid")}
}

func enqueue(t *testing.T, f *dispatch.Fake, task dispatch.Task) *wfv1.WorkflowTaskSet {
	t.Helper()
	enqueued, err := f.Dispatcher().Enqueue(logging.TestContext(t.Context()), task)
	require.NoError(t, err)
	return enqueued.TaskSet
}

// newSource returns a source of the worker over the fake, with short delays
// and per-task paths under a temporary directory.
func newSource(t *testing.T, f *dispatch.Fake, worker dispatch.Worker) *Source {
	t.Helper()
	root := t.TempDir()
	return NewSource(f.Consumer(worker), Options{
		Process: newProcess(t, time.Second),
		Paths: func(cfg executor.TaskConfig) (executor.TaskPaths, error) {
			return executor.WorkerPaths(root, cfg.NodeID)
		},
		MinMissDelay: time.Millisecond,
		MaxMissDelay: 10 * time.Millisecond,
	})
}

// newProcess returns a process whose task-result writes retry for up to
// retryTimeout.
func newProcess(t *testing.T, retryTimeout time.Duration) *executor.Process {
	t.Helper()
	p, err := executor.NewProcess(logging.TestContext(t.Context()), nil, nil, nil,
		func(executor.TaskPaths) executor.ContainerRuntimeExecutor { return &mocks.ContainerRuntimeExecutor{} },
		executor.ProcessConfig{TaskResultRetryTimeout: retryTimeout})
	require.NoError(t, err)
	return p
}

// next returns the source's next task, which must come within a few
// seconds.
func next(t *testing.T, s *Source) executor.Task {
	t.Helper()
	ctx, cancel := context.WithTimeout(logging.TestContext(t.Context()), 5*time.Second)
	defer cancel()
	task, ok, err := s.Next(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	return task
}

func result(t *testing.T, f *dispatch.Fake, nodeID string) (*wfv1.WorkflowTaskResult, bool) {
	t.Helper()
	return f.Result(testNamespace, nodeID)
}

// A claim lost to another worker is skipped: the worker takes the next
// task.
func TestSourceSkipsAClaimLostToAnotherWorker(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	f := dispatch.NewFake()
	taken, ok := dispatch.CandidateFromTaskSet(enqueue(t, f, testTask("taken")))
	require.True(t, ok)
	enqueue(t, f, testTask("next"))
	f.SetFault(func(op dispatch.Op) error {
		if op.Verb == "create" && op.Resource == dispatch.ResourceResults && op.Name == "taken" {
			f.SetFault(nil)
			_, err := f.Consumer(testWorker("worker-b")).Claim(ctx, taken)
			require.NoError(t, err)
		}
		return nil
	})
	task := next(t, newSource(t, f, testWorker("worker-a")))
	assert.Equal(t, "next", task.Config.NodeID)
	r, ok := result(t, f, "taken")
	require.True(t, ok)
	assert.Equal(t, "worker-b-uid", r.Labels[common.LabelKeyWorkerPodUID], "the other worker's claim stands")
}

// A TaskSet deleted or replaced between the claim and the worker reading
// it: the worker deletes its own result and takes the next task.
func TestSourceSkipsATaskGoneAfterTheClaim(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprintf("replaced=%v", replace), func(t *testing.T) {
			f := dispatch.NewFake()
			gone := enqueue(t, f, testTask("gone"))
			enqueue(t, f, testTask("next"))
			f.SetFault(func(op dispatch.Op) error {
				if op.Verb == "get" && op.Resource == dispatch.ResourceTaskSets && op.Name == gone.Name {
					f.SetFault(nil)
					require.True(t, f.DeleteTaskSet(testNamespace, gone.Name))
					if replace {
						enqueue(t, f, testTask("gone"))
					}
				}
				return nil
			})
			task := next(t, newSource(t, f, testWorker("worker-a")))
			assert.Equal(t, "next", task.Config.NodeID)
			_, claimed := result(t, f, "gone")
			assert.False(t, claimed, "the claim on the gone TaskSet is withdrawn")
		})
	}
}

// The claim carries the TaskSet's instance ID label only when the TaskSet
// has one.
func TestSourceClaimInstanceID(t *testing.T) {
	for _, instanceID := range []string{"", "my-instance"} {
		t.Run(fmt.Sprintf("instanceID=%q", instanceID), func(t *testing.T) {
			f := dispatch.NewFake()
			task := testTask("node-1")
			task.Pool.InstanceID = instanceID
			enqueue(t, f, task)
			next(t, newSource(t, f, testWorker("worker-a")))
			r, ok := result(t, f, "node-1")
			require.True(t, ok)
			got, has := r.Labels[common.LabelKeyControllerInstanceID]
			assert.Equal(t, instanceID != "", has)
			assert.Equal(t, instanceID, got)
		})
	}
}

// After the claim is lost, the sink's writes are rejected and the result
// is never recreated or overwritten.
func TestPooledSinkRejectsWritesAfterTheClaimIsLost(t *testing.T) {
	for _, replace := range []bool{false, true} {
		t.Run(fmt.Sprintf("replaced=%v", replace), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			f := dispatch.NewFake()
			ts := enqueue(t, f, testTask("node-1"))
			s := newSource(t, f, testWorker("worker-a"))
			sink := s.Sink()(next(t, s)).(*executor.PooledSink)
			require.NoError(t, sink.Progress(ctx, "1/2"))

			require.True(t, f.DeleteResult(testNamespace, "node-1"))
			if replace {
				other := common.NewTaskResult(testNamespace, "node-1", "my-wf", "wf-uid", "")
				other.Labels[common.LabelKeyTaskSetUID] = string(ts.UID)
				other.Labels[common.LabelKeyWorkerPodUID] = "worker-b-uid"
				_, err := f.CreateResult(other)
				require.NoError(t, err)
			}
			select {
			case <-sink.Stopped():
				assert.Equal(t, dispatch.StopLost, sink.StopReason())
			case <-time.After(5 * time.Second):
				t.Fatal("the task wasn't told its claim is lost")
			}

			require.ErrorIs(t, sink.Progress(ctx, "2/2"), dispatch.ErrClaimLost)
			require.ErrorIs(t, sink.Finalize(ctx), dispatch.ErrClaimLost)
			r, ok := result(t, f, "node-1")
			assert.Equal(t, replace, ok, "never recreated")
			if ok {
				assert.Empty(t, r.Progress)
				assert.Equal(t, "false", r.Labels[common.LabelKeyReportOutputsCompleted])
			}
		})
	}
}

// The sink writes the claim's result once, at Finalize, merging the
// reports as a pod's result merges them.
func TestPooledSinkFinalizesInOneWrite(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	f := dispatch.NewFake()
	enqueue(t, f, testTask("node-1"))
	s := newSource(t, f, testWorker("worker-a"))
	sink := s.Sink()(next(t, s))
	var writes atomic.Int32
	f.SetFault(func(op dispatch.Op) error {
		if op.Resource == dispatch.ResourceResults && op.Verb != "get" {
			writes.Add(1)
		}
		return nil
	})

	require.NoError(t, sink.Start(ctx))
	result1 := "out"
	require.NoError(t, sink.Report(ctx, wfv1.NodeResult{Outputs: &wfv1.Outputs{Result: &result1, Parameters: []wfv1.Parameter{{Name: "p", Value: wfv1.AnyStringPtr("v")}}}}))
	require.NoError(t, sink.Report(ctx, wfv1.NodeResult{Outputs: &wfv1.Outputs{Artifacts: wfv1.Artifacts{{Name: "main-logs"}}}}))
	sink.Fail(ctx, fmt.Errorf("first"))
	sink.Fail(ctx, fmt.Errorf("second"))
	assert.Zero(t, writes.Load(), "nothing is written before Finalize")
	require.NoError(t, sink.Finalize(ctx))
	assert.Equal(t, int32(1), writes.Load())

	r, ok := result(t, f, "node-1")
	require.True(t, ok)
	assert.Equal(t, wfv1.NodeError, r.Phase, "not the task's own failure, so the worker's")
	assert.Equal(t, "first", r.Message)
	require.NotNil(t, r.Outputs)
	assert.Equal(t, "out", *r.Outputs.Result)
	assert.Len(t, r.Outputs.Parameters, 1)
	assert.Equal(t, wfv1.Artifacts{{Name: "main-logs"}}, r.Outputs.Artifacts)
	assert.Equal(t, "true", r.Labels[common.LabelKeyReportOutputsCompleted])
}

func TestPooledSinkSucceedsWithoutAFailure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	f := dispatch.NewFake()
	enqueue(t, f, testTask("node-1"))
	s := newSource(t, f, testWorker("worker-a"))
	require.NoError(t, s.Sink()(next(t, s)).Finalize(ctx))
	r, _ := result(t, f, "node-1")
	assert.Equal(t, wfv1.NodeSucceeded, r.Phase)
}

// An interrupt reaches the task through its sink.
func TestPooledSinkStoppedOnInterrupt(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	f := dispatch.NewFake()
	ts := enqueue(t, f, testTask("node-1"))
	s := newSource(t, f, testWorker("worker-a"))
	sink := s.Sink()(next(t, s)).(*executor.PooledSink)
	select {
	case <-sink.Stopped():
		t.Fatal("stopped before the interrupt")
	default:
	}
	r, _ := result(t, f, "node-1")
	require.NoError(t, f.Dispatcher().Interrupt(ctx, dispatch.Key{Namespace: testNamespace, NodeID: "node-1"}, ts.UID, r.UID))
	select {
	case <-sink.Stopped():
		assert.Equal(t, dispatch.StopInterrupted, sink.StopReason())
	case <-time.After(5 * time.Second):
		t.Fatal("not stopped by the interrupt")
	}
	require.NoError(t, sink.Finalize(ctx), "the worker confirms the interrupt by completing")
}

func TestSourceStop(t *testing.T) {
	t.Run("WhileWaiting", func(t *testing.T) {
		s := newSource(t, dispatch.NewFake(), testWorker("worker-a"))
		done := make(chan bool)
		go func() {
			_, ok, err := s.Next(logging.TestContext(t.Context()))
			assert.NoError(t, err)
			done <- ok
		}()
		select {
		case <-done:
			t.Fatal("Next returned with nothing to claim")
		case <-time.After(50 * time.Millisecond):
		}
		s.Stop()
		select {
		case ok := <-done:
			assert.False(t, ok)
		case <-time.After(5 * time.Second):
			t.Fatal("Next didn't return once stopped")
		}
	})
	t.Run("Before", func(t *testing.T) {
		f := dispatch.NewFake()
		enqueue(t, f, testTask("node-1"))
		s := newSource(t, f, testWorker("worker-a"))
		s.Stop()
		s.Stop()
		_, ok, err := s.Next(logging.TestContext(t.Context()))
		require.NoError(t, err)
		assert.False(t, ok)
		_, claimed := result(t, f, "node-1")
		assert.False(t, claimed)
	})
	t.Run("DuringAClaim", func(t *testing.T) {
		f := dispatch.NewFake()
		enqueue(t, f, testTask("node-1"))
		s := newSource(t, f, testWorker("worker-a"))
		f.SetFault(func(op dispatch.Op) error {
			if op.Verb == "get" && op.Resource == dispatch.ResourceTaskSets {
				s.Stop()
			}
			return nil
		})
		_, ok, err := s.Next(logging.TestContext(t.Context()))
		require.NoError(t, err)
		assert.False(t, ok)
		_, claimed := result(t, f, "node-1")
		assert.False(t, claimed, "the claim made as the source stopped is released")
		_, queued := f.TaskSet(testNamespace, "node-1-task")
		assert.True(t, queued)
	})
	t.Run("ContextEnds", func(t *testing.T) {
		s := newSource(t, dispatch.NewFake(), testWorker("worker-a"))
		ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
		cancel()
		_, ok, err := s.Next(ctx)
		require.ErrorIs(t, err, context.Canceled)
		assert.False(t, ok)
	})
}

// A source with Ready claims nothing until Ready returns, and a Stop while
// it waits ends Next without a claim.
func TestSourceWaitsUntilReady(t *testing.T) {
	newReadySource := func(f *dispatch.Fake, ready chan struct{}) *Source {
		return NewSource(f.Consumer(testWorker("worker-a")), Options{
			Process: newProcess(t, time.Second),
			Paths: func(cfg executor.TaskConfig) (executor.TaskPaths, error) {
				return executor.WorkerPaths(t.TempDir(), cfg.NodeID)
			},
			MinMissDelay: time.Millisecond,
			MaxMissDelay: 10 * time.Millisecond,
			Ready: func(ctx context.Context) error {
				select {
				case <-ready:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			},
		})
	}
	t.Run("ClaimsOnceReady", func(t *testing.T) {
		f := dispatch.NewFake()
		enqueue(t, f, testTask("node-1"))
		ready := make(chan struct{})
		s := newReadySource(f, ready)
		got := make(chan string, 1)
		go func() {
			task, ok, err := s.Next(logging.TestContext(t.Context()))
			assert.NoError(t, err)
			assert.True(t, ok)
			got <- task.Config.NodeID
		}()
		time.Sleep(100 * time.Millisecond)
		_, claimed := result(t, f, "node-1")
		assert.False(t, claimed, "claimed before main was up")
		close(ready)
		select {
		case nodeID := <-got:
			assert.Equal(t, "node-1", nodeID)
		case <-time.After(5 * time.Second):
			t.Fatal("Next didn't claim once ready")
		}
	})
	t.Run("StoppedWhileWaiting", func(t *testing.T) {
		f := dispatch.NewFake()
		enqueue(t, f, testTask("node-1"))
		s := newReadySource(f, make(chan struct{}))
		done := make(chan bool, 1)
		go func() {
			_, ok, err := s.Next(logging.TestContext(t.Context()))
			assert.NoError(t, err)
			done <- ok
		}()
		time.Sleep(50 * time.Millisecond)
		s.Stop()
		select {
		case ok := <-done:
			assert.False(t, ok)
		case <-time.After(5 * time.Second):
			t.Fatal("Next didn't return once stopped")
		}
		_, claimed := result(t, f, "node-1")
		assert.False(t, claimed)
	})
}

// A claimed TaskSet that can't be run fails its node rather than being
// claimed by every worker in turn.
func TestSourceFailsATaskItCannotRun(t *testing.T) {
	f := dispatch.NewFake()
	enqueueUnrunnable(t, f, "bad")
	enqueue(t, f, testTask("good"))

	task := next(t, newSource(t, f, testWorker("worker-a")))
	assert.Equal(t, "good", task.Config.NodeID)
	r, ok := result(t, f, "bad")
	require.True(t, ok)
	assert.Equal(t, wfv1.NodeError, r.Phase)
	assert.Contains(t, r.Message, "spec.dispatch")
	assert.Equal(t, "true", r.Labels[common.LabelKeyReportOutputsCompleted])
}

// enqueueUnrunnable enqueues a TaskSet without spec.dispatch, as stored by
// a CRD without it, which a worker can't run.
func enqueueUnrunnable(t *testing.T, f *dispatch.Fake, nodeID string) {
	t.Helper()
	bad := testTask(nodeID)
	_, err := f.CreateTaskSet(&wfv1.WorkflowTaskSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       testNamespace,
			Name:            bad.TaskSetName(),
			Labels:          map[string]string{common.LabelKeyWorkflow: "my-wf", common.LabelKeyPoolName: "my-pool", common.LabelKeyPoolHash: "hash-1"},
			OwnerReferences: []metav1.OwnerReference{common.WorkflowOwnerReference("my-wf", "wf-uid")},
		},
		Spec: wfv1.WorkflowTaskSetSpec{Tasks: map[string]wfv1.Template{nodeID: {}}},
	})
	require.NoError(t, err)
}

// Completing or releasing a claim the source made itself retries as the
// process's task-result writes do. A claim it still can't complete or
// release is left incomplete under the worker, so Next fails: only the
// worker's restart, which fails the claims it holds, settles it.
func TestSourceSettlesItsOwnClaimsWithTheProcessRetries(t *testing.T) {
	down := apierr.NewServiceUnavailable("down")
	for name, tc := range map[string]struct {
		// release is of a claim made as the source stops; otherwise the
		// claim is failed, as its task can't be run.
		release bool
		// err fails the claim's write, the first failures times.
		err      error
		failures int32
		wantErr  string
	}{
		"FailRetried":       {err: down, failures: 1},
		"FailNotRetried":    {err: errors.New("injected"), failures: 1, wantErr: "failed to fail pooled task bad"},
		"FailGivesUp":       {err: down, failures: math.MaxInt32, wantErr: "failed to fail pooled task bad"},
		"ReleaseRetried":    {release: true, err: down, failures: 1},
		"ReleaseNotRetried": {release: true, err: errors.New("injected"), failures: 1, wantErr: "failed to release pooled task bad"},
	} {
		t.Run(name, func(t *testing.T) {
			f := dispatch.NewFake()
			verb := "patch"
			if tc.release {
				verb = "delete"
				enqueue(t, f, testTask("bad"))
			} else {
				enqueueUnrunnable(t, f, "bad")
				enqueue(t, f, testTask("good"))
			}
			s := newSource(t, f, testWorker("worker-a"))
			// Long enough for one retry, which comes a second after the
			// first try, unless the retries are meant to run out.
			retryTimeout := 5 * time.Second
			if tc.failures == math.MaxInt32 {
				retryTimeout = time.Second
			}
			s.process = newProcess(t, retryTimeout)
			var failed atomic.Int32
			f.SetFault(func(op dispatch.Op) error {
				switch {
				case tc.release && op.Verb == "get" && op.Resource == dispatch.ResourceTaskSets:
					s.Stop()
				case op.Verb == verb && op.Resource == dispatch.ResourceResults && failed.Load() < tc.failures:
					failed.Add(1)
					return tc.err
				}
				return nil
			})
			ctx, cancel := context.WithTimeout(logging.TestContext(t.Context()), 20*time.Second)
			defer cancel()
			task, ok, err := s.Next(ctx)
			r, claimed := result(t, f, "bad")
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				assert.False(t, ok)
				require.True(t, claimed, "the claim is left under the worker")
				assert.Equal(t, "false", r.Labels[common.LabelKeyReportOutputsCompleted])
				return
			}
			require.NoError(t, err)
			assert.Positive(t, failed.Load())
			if tc.release {
				assert.False(t, ok)
				assert.False(t, claimed, "released")
				return
			}
			require.True(t, ok)
			assert.Equal(t, "good", task.Config.NodeID)
			require.True(t, claimed)
			assert.Equal(t, wfv1.NodeError, r.Phase)
			assert.Equal(t, "true", r.Labels[common.LabelKeyReportOutputsCompleted])
		})
	}
}

func TestSourceYieldsTheTaskPaths(t *testing.T) {
	f := dispatch.NewFake()
	enqueue(t, f, testTask("node-1"))
	s := NewSource(f.Consumer(testWorker("worker-a")), Options{
		Process: newProcess(t, time.Second),
		Paths: func(cfg executor.TaskConfig) (executor.TaskPaths, error) {
			return executor.TaskPaths{Root: "/tasks/" + cfg.NodeID}, nil
		},
	})
	assert.Equal(t, "/tasks/node-1", next(t, s).Paths.Root)
}

// A task's progress file is in its own shared directory, so tasks run one
// after another never share it.
func TestSourceGivesEachTaskItsOwnProgressFile(t *testing.T) {
	f := dispatch.NewFake()
	task := testTask("node-1")
	task.Dispatch.ProgressPatchTickDuration = &metav1.Duration{Duration: time.Minute}
	task.Dispatch.ProgressFileTickDuration = &metav1.Duration{Duration: time.Second}
	enqueue(t, f, task)
	enqueue(t, f, testTask("node-2"))
	s := NewSource(f.Consumer(testWorker("worker-a")), Options{
		Process: newProcess(t, time.Second),
		Paths: func(cfg executor.TaskConfig) (executor.TaskPaths, error) {
			return executor.TaskPaths{VarRunArgo: "/run/" + cfg.NodeID}, nil
		},
	})
	got := map[string]string{}
	for range 2 {
		task := next(t, s)
		got[task.Config.NodeID] = task.Config.ProgressFile
	}
	assert.Equal(t, map[string]string{"node-1": filepath.Join("/run/node-1", "progress"), "node-2": ""}, got, "progress is monitored only when both ticks are set")
}

// A result the API server refuses as too large or invalid is never written
// however often it is tried: the claim is completed as Error, saying why,
// with none of the task's outputs, so the node fails visibly.
func TestPooledSinkCompletesAnUnreportableResultAsError(t *testing.T) {
	for name, refusal := range map[string]error{
		"TooLarge":     apierr.NewRequestEntityTooLargeError("limit is 3145728"),
		"EtcdTooLarge": apierr.NewInternalError(errors.New("etcdserver: request is too large")),
		"Invalid":      apierr.NewInvalid(schema.GroupKind{Group: "argoproj.io", Kind: "WorkflowTaskResult"}, "node-1", nil),
	} {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			f := dispatch.NewFake()
			enqueue(t, f, testTask("node-1"))
			s := newSource(t, f, testWorker("worker-a"))
			sink := s.Sink()(next(t, s))
			var patches atomic.Int32
			f.SetFault(func(op dispatch.Op) error {
				if op.Resource == dispatch.ResourceResults && op.Verb == "patch" && patches.Add(1) == 1 {
					return refusal
				}
				return nil
			})
			out := "out"
			require.NoError(t, sink.Report(ctx, wfv1.NodeResult{Outputs: &wfv1.Outputs{Result: &out}}))

			require.NoError(t, sink.Finalize(ctx))

			r, ok := result(t, f, "node-1")
			require.True(t, ok)
			assert.Equal(t, wfv1.NodeError, r.Phase)
			assert.Contains(t, r.Message, "failed to report the task's result")
			assert.Nil(t, r.Outputs, "nothing is truncated")
			assert.Equal(t, "true", r.Labels[common.LabelKeyReportOutputsCompleted])
		})
	}
}

// Releasing a task deletes its claim, so the task can be claimed again;
// its TaskSet stays.
func TestPooledSinkRelease(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	f := dispatch.NewFake()
	enqueue(t, f, testTask("node-1"))
	s := newSource(t, f, testWorker("worker-a"))
	sink := s.Sink()(next(t, s)).(*executor.PooledSink)

	require.NoError(t, sink.Release(ctx))

	_, ok := result(t, f, "node-1")
	assert.False(t, ok)
	_, ok = f.TaskSet(testNamespace, dispatch.Key{NodeID: "node-1"}.TaskSetName())
	assert.True(t, ok)
}
