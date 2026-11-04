package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/sync"
)

// pooledStopWf runs a pooled step holding a mutex, with a pooled exit
// handler.
const pooledStopWf = `
metadata:
  name: pooled-stop
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  onExit: exit
  templates:
  - name: main
    steps:
    - - name: a
        template: work
  - name: work
    pool: {}
    synchronization:
      mutexes:
      - name: pooled
    container:
      image: my-image
  - name: exit
    pool: {}
    container:
      image: my-image
`

const (
	pooledStopStep = "pooled-stop[0].a"
	pooledStopExit = "pooled-stop.onExit"
)

// newPooledStopController is newPooledDispatchController for pooledStopWf,
// with a lock manager for its mutex.
func newPooledStopController(t *testing.T) (context.Context, *WorkflowController, *dispatch.Fake, *wfv1.Workflow) {
	t.Helper()
	wf := wfv1.MustUnmarshalWorkflow(pooledStopWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(string) {}, workflowExistenceFunc, false)
	return ctx, controller, fake, wf
}

// shutDown sets the stored workflow's shutdown strategy.
func shutDown(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, strategy wfv1.ShutdownStrategy) {
	t.Helper()
	updateStored(ctx, t, controller, wf, func(stored *wfv1.Workflow) { stored.Spec.Shutdown = strategy })
}

// pastDeadline gives the stored workflow a deadline that has passed.
func pastDeadline(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow) {
	t.Helper()
	updateStored(ctx, t, controller, wf, func(stored *wfv1.Workflow) {
		stored.Spec.ActiveDeadlineSeconds = new(int64(60))
		stored.Status.StartedAt = metav1.NewTime(time.Now().Add(-time.Hour))
	})
}

// holdsMutex says whether the workflow holds a mutex.
func holdsMutex(woc *wfOperationCtx) bool {
	return woc.wf.Status.Synchronization != nil && woc.wf.Status.Synchronization.Mutex != nil && len(woc.wf.Status.Synchronization.Mutex.Holding) > 0
}

// TestPooledShutdownDropsQueuedTasks checks that stop, terminate and the
// workflow deadline drop an unclaimed task's TaskSet and fail its node at
// once, with pods' message.
func TestPooledShutdownDropsQueuedTasks(t *testing.T) {
	for _, tt := range []struct {
		name    string
		end     func(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow)
		message string
	}{
		{
			name: "stop",
			end: func(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow) {
				shutDown(ctx, t, controller, wf, wfv1.ShutdownStrategyStop)
			},
			message: "workflow shutdown with strategy:  Stop",
		},
		{
			name: "terminate",
			end: func(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow) {
				shutDown(ctx, t, controller, wf, wfv1.ShutdownStrategyTerminate)
			},
			message: "workflow shutdown with strategy:  Terminate",
		},
		{name: "deadline", end: pastDeadline, message: "Step exceeded its deadline"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, controller, fake, wf := newPooledStopController(t)
			id := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep).ID
			require.True(t, hasTaskSet(fake, wf.Namespace, id))

			tt.end(ctx, t, controller, wf)
			woc := reconcilePooled(ctx, t, controller, wf)
			node := mustNode(t, woc, pooledStopStep)
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, tt.message, node.Message)
			assert.True(t, node.Fulfilled(), "nothing runs an unclaimed task")
			assert.Equal(t, wfv1.PooledTaskEnqueued, node.PooledTask.State())
			assert.False(t, hasTaskSet(fake, wf.Namespace, id))
			assert.False(t, holdsMutex(woc))
		})
	}
}

// TestPooledShutdownInterruptsClaimedTasks checks that stop, terminate and
// the workflow deadline fail a claimed node with pods' message and interrupt
// its task, and the node stays unfinished, its mutex held and the workflow
// not completing, until its worker confirms the task stopped; the confirming
// result's phase and outputs are ignored. As for a pod, stop fails the steps
// boundary without waiting for its children, so the workflow's exit handler
// starts at once; past the deadline it waits for the node.
func TestPooledShutdownInterruptsClaimedTasks(t *testing.T) {
	for _, tt := range []struct {
		name    string
		end     func(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow)
		message string
		// exitHandler is whether the exit handler runs, and exitEarly whether
		// it starts before the node finishes.
		exitHandler, exitEarly bool
	}{
		{name: "stop", end: stop, message: "workflow shutdown with strategy:  Stop", exitHandler: true, exitEarly: true},
		{name: "terminate", end: terminate, message: "workflow shutdown with strategy:  Terminate"},
		{name: "deadline", end: pastDeadline, message: "Step exceeded its deadline", exitHandler: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, controller, fake, wf := newPooledStopController(t)
			id, h := runPooledNode(ctx, t, controller, fake, wf, pooledStopStep)
			require.True(t, holdsMutex(reconcilePooled(ctx, t, controller, wf)))
			exitStarted := func(woc *wfOperationCtx) bool { return woc.wf.Status.Nodes.Has(woc.wf.NodeID(pooledStopExit)) }

			tt.end(ctx, t, controller, wf)
			woc := reconcilePooled(ctx, t, controller, wf)
			node := mustNode(t, woc, pooledStopStep)
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, tt.message, node.Message)
			assert.False(t, node.Fulfilled())
			assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
			interruptedResult(t, fake, wf.Namespace, id)
			assert.False(t, hasTaskSet(fake, wf.Namespace, id))
			assert.Equal(t, dispatch.StopInterrupted, h.StopReason())

			observeResult(ctx, t, controller, fake, wf.Namespace, id)
			woc = reconcilePooled(ctx, t, controller, wf)
			assert.False(t, mustNode(t, woc, pooledStopStep).Fulfilled())
			assert.True(t, holdsMutex(woc), "the mutex is held until the task stops")
			assert.False(t, workflowCompleted(woc))
			assert.Equal(t, tt.exitEarly, exitStarted(woc))

			require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{ExitCode: new("0")}}))
			observeResult(ctx, t, controller, fake, wf.Namespace, id)
			woc = reconcilePooled(ctx, t, controller, wf)
			node = mustNode(t, woc, pooledStopStep)
			assert.True(t, node.Fulfilled())
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, tt.message, node.Message)
			assert.Nil(t, node.Outputs)
			woc = reconcilePooled(ctx, t, controller, wf)
			assert.Equal(t, tt.exitHandler, exitStarted(woc))
			if !tt.exitHandler {
				assert.True(t, workflowCompleted(woc))
				assert.False(t, holdsMutex(woc))
			}
		})
	}
}

// TestPooledShutdownExitHandler checks that stop and the workflow deadline
// spare a pooled exit handler, queued or claimed, so it runs on; terminate
// drops or interrupts it, as it kills an exit handler's pod.
func TestPooledShutdownExitHandler(t *testing.T) {
	for _, tt := range []struct {
		name     string
		end      func(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow)
		claimed  bool
		stopped  bool
		finished bool
	}{
		{name: "stop, queued", end: stop},
		{name: "stop, claimed", end: stop, claimed: true},
		{name: "deadline, queued", end: pastDeadline},
		{name: "deadline, claimed", end: pastDeadline, claimed: true},
		{name: "terminate, queued", end: terminate, stopped: true, finished: true},
		{name: "terminate, claimed", end: terminate, claimed: true, stopped: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, controller, fake, wf := newPooledStopController(t)
			id := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep).ID
			finishPooled(ctx, t, controller, fake, wf.Namespace, id)
			reconcilePooled(ctx, t, controller, wf)
			exitID := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopExit).ID
			if tt.claimed {
				runPooledNode(ctx, t, controller, fake, wf, pooledStopExit)
			}

			tt.end(ctx, t, controller, wf)
			node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopExit)
			if !tt.stopped {
				assert.False(t, node.Phase.Completed())
				assert.True(t, hasTaskSet(fake, wf.Namespace, exitID))
				return
			}
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, "workflow shutdown with strategy:  Terminate", node.Message)
			assert.Equal(t, tt.finished, node.Fulfilled())
			assert.False(t, hasTaskSet(fake, wf.Namespace, exitID))
			if tt.claimed {
				interruptedResult(t, fake, wf.Namespace, exitID)
			}
		})
	}
}

func stop(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow) {
	t.Helper()
	shutDown(ctx, t, controller, wf, wfv1.ShutdownStrategyStop)
}

func terminate(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow) {
	t.Helper()
	shutDown(ctx, t, controller, wf, wfv1.ShutdownStrategyTerminate)
}

// TestPooledShutdownDropFindsClaim checks that a claim on a queued task
// that the controller hasn't recorded, because its claimant pod isn't cached
// or its result isn't, is found by the drop and interrupted, the ledger
// recording it with the claimant's restart count from the pool pod cache, or
// 0, which a claimant cached later with restarts doesn't take for a restart
// since; a claim whose result is cached but whose drop finds it completed
// ends the node with its result.
func TestPooledShutdownDropFindsClaim(t *testing.T) {
	for _, tt := range []struct {
		name      string
		podCached bool
		complete  bool
		// restartedClaimant is a claimant not cached at the drop, cached
		// after with its worker container restarted and running.
		restartedClaimant bool
	}{
		{name: "claimant cached", podCached: true},
		{name: "claimant not cached"},
		{name: "claimant not cached, restarted", restartedClaimant: true},
		{name: "completed", podCached: true, complete: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, controller, fake, wf := newPooledStopController(t)
			id := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep).ID
			worker := dispatch.Worker{Namespace: wf.Namespace, PodName: "work-0123456789-fghij", PodUID: "uncached-uid"}
			if tt.podCached {
				worker = observeWorkerPod(ctx, t, controller, wf.Namespace, claimantName, claimantUID, 3)
			}
			h := claimPooledBy(ctx, t, fake, wf.Namespace, id, worker)
			if tt.complete {
				require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{ExitCode: new("0")}}))
			}

			stop(ctx, t, controller, wf)
			node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep)
			assert.False(t, hasTaskSet(fake, wf.Namespace, id))
			if tt.complete {
				assert.Equal(t, wfv1.NodeSucceeded, node.Phase, "the task ended before the drop")
				assert.True(t, node.Fulfilled())
				require.NotNil(t, node.Outputs)
				assert.Equal(t, new("0"), node.Outputs.ExitCode)
				return
			}
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, "workflow shutdown with strategy:  Stop", node.Message)
			assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
			assert.Equal(t, h.Claim().UID, node.PooledTask.ClaimUID)
			assert.Equal(t, worker.PodName, node.PooledTask.WorkerPodName)
			if tt.podCached {
				assert.Equal(t, int32(3), node.PooledTask.WorkerRestartCount)
				assert.False(t, node.Fulfilled(), "the node waits for its task to stop")
			} else {
				assert.Equal(t, int32(0), node.PooledTask.WorkerRestartCount)
				assert.False(t, node.Fulfilled(), "the claimant may just not be cached yet")
				observeResult(ctx, t, controller, fake, wf.Namespace, id)
				if tt.restartedClaimant {
					observeWorkerPod(ctx, t, controller, wf.Namespace, worker.PodName, worker.PodUID, 2)
					node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep)
					assert.False(t, node.Fulfilled(), "a claimant cached late with restarts runs its task")
					return
				}
				// The claimant pod doesn't exist, so it is gone.
				node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep)
				assert.True(t, node.Fulfilled())
			}
			interruptedResult(t, fake, wf.Namespace, id)
			assert.Equal(t, dispatch.StopInterrupted, h.StopReason())
		})
	}
}

// TestPooledShutdownRacingClaim checks, against the fake, that a claim racing
// the drop of a queued task either is found and interrupted, or never starts;
// and a node that finished is never made unfinished by a claim seen later.
func TestPooledShutdownRacingClaim(t *testing.T) {
	t.Run("claimed before the TaskSet is deleted", func(t *testing.T) {
		ctx, controller, fake, wf := newPooledStopController(t)
		id := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep).ID
		ts, ok := fake.TaskSet(wf.Namespace, pooledTaskSetName(id))
		require.True(t, ok)
		candidate, ok := dispatch.CandidateFromTaskSet(ts)
		require.True(t, ok)
		worker := observeWorkerPod(ctx, t, controller, wf.Namespace, claimantName, claimantUID, 0)
		worker.PoolHash = candidate.PoolHash
		var h dispatch.Claimed
		fake.SetFault(func(op dispatch.Op) error {
			if op.Verb == "delete" && op.Resource == dispatch.ResourceTaskSets && h == nil {
				var err error
				h, err = fake.Consumer(worker).Claim(ctx, candidate)
				require.NoError(t, err)
			}
			return nil
		})

		stop(ctx, t, controller, wf)
		node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep)
		require.NotNil(t, h, "the worker claimed the task and started it")
		assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State(), "the drop found the claim")
		assert.False(t, node.Fulfilled())
		assert.Equal(t, dispatch.StopInterrupted, h.StopReason())
	})
	t.Run("claimed after the TaskSet is deleted", func(t *testing.T) {
		ctx, controller, fake, wf := newPooledStopController(t)
		id := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep).ID
		ts, ok := fake.TaskSet(wf.Namespace, pooledTaskSetName(id))
		require.True(t, ok)
		candidate, ok := dispatch.CandidateFromTaskSet(ts)
		require.True(t, ok)
		worker := observeWorkerPod(ctx, t, controller, wf.Namespace, claimantName, claimantUID, 0)
		worker.PoolHash = candidate.PoolHash
		var claimErr error
		claimed := false
		fake.SetFault(func(op dispatch.Op) error {
			if op.Verb == "get" && op.Resource == dispatch.ResourceResults && !claimed {
				claimed = true
				_, claimErr = fake.Consumer(worker).Claim(ctx, candidate)
			}
			return nil
		})

		stop(ctx, t, controller, wf)
		node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep)
		require.True(t, claimed)
		require.ErrorIs(t, claimErr, dispatch.ErrTaskGone, "the claim never starts")
		assert.True(t, node.Fulfilled())
		assert.Equal(t, wfv1.PooledTaskEnqueued, node.PooledTask.State())
		fake.SetFault(nil)

		// A stale claim on the TaskSet seen afterwards is interrupted, but the
		// finished node stays finished.
		claim, err := fake.CreateResult(&wfv1.WorkflowTaskResult{ObjectMeta: metav1.ObjectMeta{
			Namespace: wf.Namespace,
			Name:      id,
			Labels: map[string]string{
				common.LabelKeyWorkflow:               wf.Name,
				common.LabelKeyReportOutputsCompleted: "false",
				common.LabelKeyTaskSetUID:             string(ts.UID),
				common.LabelKeyWorkerPod:              worker.PodName,
				common.LabelKeyWorkerPodUID:           string(worker.PodUID),
			},
		}})
		require.NoError(t, err)
		observeResult(ctx, t, controller, fake, wf.Namespace, id)
		node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep)
		assert.True(t, node.Fulfilled())
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
		assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
		assert.Equal(t, claim.UID, node.PooledTask.ClaimUID)
		interruptedResult(t, fake, wf.Namespace, id)
	})
}

// TestPooledShutdownRetried checks a task that can't be shut down is left as
// it is and shut down at a later reconcile.
func TestPooledShutdownRetried(t *testing.T) {
	ctx, controller, fake, wf := newPooledStopController(t)
	id, h := runPooledNode(ctx, t, controller, fake, wf, pooledStopStep)
	fake.SetFault(func(op dispatch.Op) error {
		if op.Verb == "patch" {
			return assert.AnError
		}
		return nil
	})

	stop(ctx, t, controller, wf)
	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep)
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())

	fake.SetFault(nil)
	node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
	assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
	interruptedResult(t, fake, wf.Namespace, id)
	assert.Equal(t, dispatch.StopInterrupted, h.StopReason())
}

// TestPooledShutdownBeforeDispatch checks a node the shutdown reaches
// before its dispatch: it fails with the message of a pod the shutdown kept
// from being created, and no task is enqueued.
func TestPooledShutdownBeforeDispatch(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledStopWf)
	wf.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStopStep)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
	assert.Equal(t, "workflow shutdown with strategy: Terminate", node.Message)
	assert.Nil(t, node.PooledTask)
	assert.Empty(t, fake.TaskSets(wf.Namespace))
}
