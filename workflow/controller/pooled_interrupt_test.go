package controller

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// updateStored changes the stored workflow, as another writer, or time
// passing, would.
func updateStored(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, change func(*wfv1.Workflow)) {
	t.Helper()
	wfs := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace)
	stored, err := wfs.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	change(stored)
	_, err = wfs.Update(ctx, stored, metav1.UpdateOptions{})
	require.NoError(t, err)
}

// updateStoredNode changes the stored workflow's node id.
func updateStoredNode(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, id string, change func(*wfv1.NodeStatus)) {
	t.Helper()
	updateStored(ctx, t, controller, wf, func(stored *wfv1.Workflow) {
		node := stored.Status.Nodes[id]
		change(&node)
		stored.Status.Nodes[id] = node
	})
}

// endWithoutInterrupt ends the claimed pooled node as Failed with "hook
// failed", as something that doesn't interrupt its task would, such as a hook
// error failing a pooled entrypoint: its task result stays in progress.
func endWithoutInterrupt(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, id string) {
	t.Helper()
	updateStoredNode(ctx, t, controller, wf, id, func(node *wfv1.NodeStatus) {
		node.Phase, node.Message = wfv1.NodeFailed, "hook failed"
	})
}

// interruptedAt moves the time the pooled node's ledger records it was
// interrupted.
func interruptedAt(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, id string, at time.Time) {
	t.Helper()
	updateStoredNode(ctx, t, controller, wf, id, func(node *wfv1.NodeStatus) {
		c := meta.FindStatusCondition(node.PooledTask.Conditions, wfv1.PooledTaskConditionInterrupted)
		require.NotNil(t, c)
		c.LastTransitionTime = metav1.NewTime(at)
	})
}

// interruptedResult returns the node's result as fake has it, and requires
// it to carry the interrupt label.
func interruptedResult(t *testing.T, fake *dispatch.Fake, namespace, id string) *wfv1.WorkflowTaskResult {
	t.Helper()
	r, ok := fake.Result(namespace, id)
	require.True(t, ok)
	require.Equal(t, "true", r.Labels[common.LabelKeyInterrupt])
	return r
}

// TestPooledEndedNodeClaimInterrupted checks that an incomplete claim on a
// node that has already ended is interrupted, its ledger recording the claim
// and the interrupt. A node that had finished stays finished; one still
// unfinished waits for its task to stop.
func TestPooledEndedNodeClaimInterrupted(t *testing.T) {
	t.Run("claim racing a liveness ending", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
		ctx, controller, fake := newPooledDispatchController(t, true, wf)
		controller.poolHealth = fake
		id := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep).ID
		observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
		// A worker claims the task just before its pool is reported
		// unbuildable, which ends the node without deleting anything.
		worker := observeWorkerPod(ctx, t, controller, wf.Namespace, claimantName, claimantUID, 2)
		claimPooledBy(ctx, t, fake, wf.Namespace, id, worker)
		node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
		fake.SetHealth(wf.Namespace, node.PooledTask.PoolHash, dispatch.Health{State: dispatch.HealthUnbuildable, Reason: "gone"})
		node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
		require.Equal(t, wfv1.NodeError, node.Phase)
		require.True(t, node.Fulfilled())
		require.Equal(t, wfv1.PooledTaskEnqueued, node.PooledTask.State())

		observeResult(ctx, t, controller, fake, wf.Namespace, id)
		node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
		claim := interruptedResult(t, fake, wf.Namespace, id)
		assert.Equal(t, wfv1.NodeError, node.Phase)
		assert.True(t, node.Fulfilled(), "a finished node stays finished")
		ledger := node.PooledTask
		assert.Equal(t, wfv1.PooledTaskInterrupted, ledger.State())
		assert.Equal(t, claim.UID, ledger.ClaimUID)
		assert.Equal(t, claimantName, ledger.WorkerPodName)
		assert.Equal(t, int32(2), ledger.WorkerRestartCount, "from the pool pod cache")
		assert.NotNil(t, ledger.InterruptedAt())
	})
	t.Run("claimant not cached", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
		ctx, controller, fake := newPooledDispatchController(t, true, wf)
		controller.poolHealth = fake
		id := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep).ID
		observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
		claimPooledBy(ctx, t, fake, wf.Namespace, id, dispatch.Worker{Namespace: wf.Namespace, PodName: "work-0123456789-fghij", PodUID: "uncached-uid"})
		node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
		fake.SetHealth(wf.Namespace, node.PooledTask.PoolHash, dispatch.Health{State: dispatch.HealthUnbuildable, Reason: "gone"})
		reconcilePooled(ctx, t, controller, wf)

		observeResult(ctx, t, controller, fake, wf.Namespace, id)
		node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
		interruptedResult(t, fake, wf.Namespace, id)
		assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
		assert.Equal(t, "work-0123456789-fghij", node.PooledTask.WorkerPodName)
		assert.Equal(t, int32(0), node.PooledTask.WorkerRestartCount)
	})
	t.Run("claimed node ended without an interrupt", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
		ctx, controller, fake := newPooledDispatchController(t, true, wf)
		id, _ := runPooled(ctx, t, controller, fake, wf)
		claimUID := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep).PooledTask.ClaimUID
		endWithoutInterrupt(ctx, t, controller, wf, id)

		node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
		interruptedResult(t, fake, wf.Namespace, id)
		assert.False(t, hasTaskSet(fake, wf.Namespace, id), "the interrupt deletes the TaskSet")
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
		assert.Equal(t, "hook failed", node.Message)
		assert.False(t, node.Fulfilled(), "the node waits for its task to stop")
		assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
		assert.Equal(t, claimUID, node.PooledTask.ClaimUID)
	})
	t.Run("dead claimant's claim", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
		ctx, controller, fake := newPooledDispatchController(t, true, wf)
		id, _ := runPooled(ctx, t, controller, fake, wf)
		deletePoolPod(ctx, t, controller, wf.Namespace, claimantName)
		require.Equal(t, wfv1.NodeError, mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep).Phase)

		node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
		r, ok := fake.Result(wf.Namespace, id)
		require.True(t, ok)
		assert.Empty(t, r.Labels[common.LabelKeyInterrupt], "nothing runs a dead claimant's task")
		assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())
	})
}

// interruptPooled runs pooledDispatchWf's pooled step on a cached worker and
// ends it by a path that doesn't interrupt its task, so the controller
// interrupts it. It returns the node's ID and the worker's handle, with
// the controller's result cache holding the interrupted claim.
func interruptPooled(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, wf *wfv1.Workflow) (string, dispatch.Claimed) {
	t.Helper()
	id, h := runPooled(ctx, t, controller, fake, wf)
	endWithoutInterrupt(ctx, t, controller, wf, id)
	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
	require.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
	require.False(t, node.Fulfilled())
	observeResult(ctx, t, controller, fake, wf.Namespace, id)
	select {
	case <-h.Stopped():
		require.Equal(t, dispatch.StopInterrupted, h.StopReason())
	default:
		require.Fail(t, "the worker is told to stop its task")
	}
	return id, h
}

// TestPooledInterruptConfirmed checks that an interrupted node is finished,
// keeping the phase and message it was ended with, once its task is known to
// have stopped: its worker completes its claim, whose phase and outputs are
// ignored, or deletes it; its claimant pod is gone; or the lease slack has
// passed since the interrupt, for a worker that never confirms. Until then it
// stays unfinished.
func TestPooledInterruptConfirmed(t *testing.T) {
	for _, tt := range []struct {
		name string
		stop func(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, wf *wfv1.Workflow, id string, h dispatch.Claimed)
		// finished is whether the node finishes.
		finished bool
	}{
		{
			name: "not confirmed",
			stop: func(context.Context, *testing.T, *WorkflowController, *dispatch.Fake, *wfv1.Workflow, string, dispatch.Claimed) {
			},
		},
		{
			name: "within the lease slack",
			stop: func(ctx context.Context, t *testing.T, controller *WorkflowController, _ *dispatch.Fake, wf *wfv1.Workflow, id string, _ dispatch.Claimed) {
				interruptedAt(ctx, t, controller, wf, id, time.Now().Add(-time.Minute))
			},
		},
		{
			name: "worker completed",
			stop: func(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, wf *wfv1.Workflow, id string, h dispatch.Claimed) {
				require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Message: "done", Outputs: &wfv1.Outputs{ExitCode: new("0")}}))
				observeResult(ctx, t, controller, fake, wf.Namespace, id)
			},
			finished: true,
		},
		{
			name: "claim deleted",
			stop: func(ctx context.Context, t *testing.T, controller *WorkflowController, _ *dispatch.Fake, wf *wfv1.Workflow, id string, h dispatch.Claimed) {
				require.NoError(t, h.Release(ctx))
				forgetResult(ctx, t, controller, wf.Namespace, id)
			},
			finished: true,
		},
		{
			name: "claim deleted, cache lagging",
			stop: func(ctx context.Context, t *testing.T, _ *WorkflowController, _ *dispatch.Fake, _ *wfv1.Workflow, _ string, h dispatch.Claimed) {
				require.NoError(t, h.Release(ctx))
			},
		},
		{
			name: "claimant gone",
			stop: func(ctx context.Context, t *testing.T, controller *WorkflowController, _ *dispatch.Fake, wf *wfv1.Workflow, _ string, _ dispatch.Claimed) {
				deletePoolPod(ctx, t, controller, wf.Namespace, claimantName)
			},
			finished: true,
		},
		{
			name: "lease slack passed",
			stop: func(ctx context.Context, t *testing.T, controller *WorkflowController, _ *dispatch.Fake, wf *wfv1.Workflow, id string, _ dispatch.Claimed) {
				interruptedAt(ctx, t, controller, wf, id, time.Now().Add(-time.Duration(controller.Config.HotWorkers.GetLeaseSlackSeconds())*time.Second-time.Second))
			},
			finished: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			id, h := interruptPooled(ctx, t, controller, fake, wf)

			tt.stop(ctx, t, controller, fake, wf, id, h)
			requeued := recordRequeues(controller)
			node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
			if tt.name == "within the lease slack" {
				_, after := requeued.take()
				slack := time.Duration(controller.Config.HotWorkers.GetLeaseSlackSeconds()) * time.Second
				assert.True(t, requeuedAfter(after, slack-time.Minute), "requeued for the end of the lease slack, got %v", after)
			}
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, "hook failed", node.Message)
			assert.Nil(t, node.Outputs)
			assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
			assert.Equal(t, tt.finished, node.Fulfilled())
			assert.Equal(t, tt.finished, !node.FinishedAt.IsZero())
		})
	}
}

// TestPooledInterruptedTaskNotClaimedAgain checks that once interrupted, the
// node's TaskSet is never claimed again, even while its old worker hasn't
// noticed and still runs the task.
func TestPooledInterruptedTaskNotClaimedAgain(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	id, _ := runPooled(ctx, t, controller, fake, wf)
	ts, ok := fake.TaskSet(wf.Namespace, pooledTaskSetName(id))
	require.True(t, ok)
	candidate, ok := dispatch.CandidateFromTaskSet(ts)
	require.True(t, ok)
	endWithoutInterrupt(ctx, t, controller, wf, id)
	reconcilePooled(ctx, t, controller, wf)

	other := fake.Consumer(dispatch.Worker{Namespace: wf.Namespace, PoolHash: candidate.PoolHash, PodName: "work-0123456789-other", PodUID: "other-uid"})
	short, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	_, err := other.Candidates(short)
	require.ErrorIs(t, err, context.DeadlineExceeded, "no candidates")
	_, err = other.Claim(ctx, candidate)
	require.ErrorIs(t, err, dispatch.ErrClaimLost, "the old claim still holds the name")

	// Even once the old claim is gone, the TaskSet is.
	require.True(t, fake.DeleteResult(wf.Namespace, id))
	_, err = other.Claim(ctx, candidate)
	require.ErrorIs(t, err, dispatch.ErrTaskGone)
	_, ok = fake.Result(wf.Namespace, id)
	assert.False(t, ok, "the claim was withdrawn")
}

// TestPooledLease checks a claimed task's lease: the earlier of its
// absolute deadline and its claim's time plus its activeDeadlineSeconds, plus
// the slack, and none with neither, or unclaimed.
func TestPooledLease(t *testing.T) {
	claimed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	slack := 5 * time.Minute
	ledger := func(deadline *time.Time, ads *int64) *wfv1.PooledTaskStatus {
		l := &wfv1.PooledTaskStatus{ActiveDeadlineSeconds: ads}
		if deadline != nil {
			l.Deadline = &metav1.Time{Time: *deadline}
		}
		l.Enqueue(metav1.NewTime(claimed.Add(-time.Hour)))
		l.Claim("claim-uid", metav1.NewTime(claimed), "pod", "pod-uid", 0)
		return l
	}
	early, late := claimed.Add(time.Minute), claimed.Add(time.Hour)
	for _, tt := range []struct {
		name   string
		ledger *wfv1.PooledTaskStatus
		want   time.Time
	}{
		{name: "no deadline", ledger: ledger(nil, nil)},
		{name: "absolute deadline", ledger: ledger(&late, nil), want: late.Add(slack)},
		{name: "activeDeadlineSeconds from the claim", ledger: ledger(nil, new(int64(60))), want: early.Add(slack)},
		{name: "the earlier, activeDeadlineSeconds", ledger: ledger(&late, new(int64(60))), want: early.Add(slack)},
		{name: "the earlier, absolute deadline", ledger: ledger(&early, new(int64(3600))), want: early.Add(slack)},
		{name: "unclaimed", ledger: &wfv1.PooledTaskStatus{Deadline: &metav1.Time{Time: late}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pooledLease(tt.ledger, slack)
			assert.Equal(t, !tt.want.IsZero(), ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestPooledLeaseInterrupts checks that a claimed task past its deadline plus
// the lease slack is interrupted, its node failing with pods' deadline message
// and waiting for the task to stop. Within the slack, or with no deadline, it
// runs on.
func TestPooledLeaseInterrupts(t *testing.T) {
	slack := 5 * time.Minute
	for _, tt := range []struct {
		name string
		set  func(ledger *wfv1.PooledTaskStatus)
		want bool
		// remaining is the time left of the lease of a task that runs on, if
		// it has one.
		remaining time.Duration
	}{
		{name: "no deadline", set: func(*wfv1.PooledTaskStatus) {}},
		{
			name: "deadline passed, within the slack",
			set: func(ledger *wfv1.PooledTaskStatus) {
				ledger.Deadline = &metav1.Time{Time: time.Now().Add(-time.Minute)}
			},
			remaining: slack - time.Minute,
		},
		{
			name: "deadline and slack passed",
			set: func(ledger *wfv1.PooledTaskStatus) {
				ledger.Deadline = &metav1.Time{Time: time.Now().Add(-slack - time.Second)}
			},
			want: true,
		},
		{
			name: "activeDeadlineSeconds and slack passed",
			set: func(ledger *wfv1.PooledTaskStatus) {
				ledger.Deadline = &metav1.Time{Time: time.Now().Add(time.Hour)}
				ledger.ActiveDeadlineSeconds = new(int64(60))
				meta.FindStatusCondition(ledger.Conditions, wfv1.PooledTaskConditionClaimed).LastTransitionTime = metav1.NewTime(time.Now().Add(-time.Minute - slack - time.Second))
			},
			want: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			id, h := runPooled(ctx, t, controller, fake, wf)
			updateStoredNode(ctx, t, controller, wf, id, func(node *wfv1.NodeStatus) { tt.set(node.PooledTask) })
			requeued := recordRequeues(controller)

			node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
			_, after := requeued.take()
			if !tt.want {
				assert.Equal(t, wfv1.NodeRunning, node.Phase)
				assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())
				if tt.remaining == 0 {
					assert.Empty(t, after, "a task without a lease isn't requeued")
				} else {
					assert.True(t, requeuedAfter(after, tt.remaining), "requeued for the lease's end, got %v", after)
				}
				return
			}
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, "Step exceeded its deadline", node.Message)
			assert.False(t, node.Fulfilled())
			assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
			interruptedResult(t, fake, wf.Namespace, id)
			assert.Equal(t, dispatch.StopInterrupted, h.StopReason())
		})
	}
}

// pastLease moves the deadline of the pooled node's claimed task to before
// its lease slack, so its lease has ended.
func pastLease(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, id string) {
	t.Helper()
	slack := time.Duration(controller.Config.HotWorkers.GetLeaseSlackSeconds()) * time.Second
	updateStoredNode(ctx, t, controller, wf, id, func(node *wfv1.NodeStatus) {
		node.PooledTask.Deadline = &metav1.Time{Time: time.Now().Add(-slack - time.Second)}
	})
}

// failResultPatches makes fake fail every patch of a result, as an interrupt
// makes, while fail is set.
func failResultPatches(fake *dispatch.Fake, fail *atomic.Bool) {
	fake.SetFault(func(op dispatch.Op) error {
		if fail.Load() && op.Verb == "patch" && op.Resource == dispatch.ResourceResults {
			return errors.New("injected")
		}
		return nil
	})
}

// TestPooledInterruptFindsClaimGone checks that an interrupt that finds the
// claim already deleted, which the result cache hasn't seen yet, finishes the
// node at once, with its task result synced, since nothing runs its task: for
// a lease that has ended, and for a shutdown.
func TestPooledInterruptFindsClaimGone(t *testing.T) {
	for _, tt := range []struct {
		name    string
		end     func(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, id string)
		message string
	}{
		{name: "lease ended", end: pastLease, message: "Step exceeded its deadline"},
		{
			name: "shutdown",
			end: func(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, _ string) {
				terminate(ctx, t, controller, wf)
			},
			message: "workflow shutdown with strategy:  Terminate",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			id, h := runPooled(ctx, t, controller, fake, wf)
			require.NoError(t, h.Release(ctx))

			tt.end(ctx, t, controller, wf, id)
			node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, tt.message, node.Message)
			assert.True(t, *node.TaskResultSynced)
			assert.True(t, node.Fulfilled(), "nothing runs the task")
			assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
			assert.False(t, hasTaskSet(fake, wf.Namespace, id))
		})
	}
}

// TestPooledInterruptRetried checks that an interrupt that fails, other than
// in a shutdown, leaves the node as it was and requeues the workflow, and the
// next reconcile interrupts it: a claimed task whose lease has ended, and a
// claim on a node that ended without an interrupt (interruptEndedClaim).
func TestPooledInterruptRetried(t *testing.T) {
	for _, tt := range []struct {
		name string
		end  func(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, id string)
		// phase and message are the node's once the interrupt has failed.
		phase   wfv1.NodePhase
		message string
		// wantMessage is the node's once interrupted.
		wantMessage string
	}{
		{name: "lease ended", end: pastLease, phase: wfv1.NodeRunning, wantMessage: "Step exceeded its deadline"},
		{
			name: "ended without an interrupt",
			end: func(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, id string) {
				endWithoutInterrupt(ctx, t, controller, wf, id)
			},
			phase:       wfv1.NodeFailed,
			message:     "hook failed",
			wantMessage: "hook failed",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			id, h := runPooled(ctx, t, controller, fake, wf)
			tt.end(ctx, t, controller, wf, id)
			var fail atomic.Bool
			fail.Store(true)
			failResultPatches(fake, &fail)
			requeued := recordRequeues(controller)

			node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
			assert.Equal(t, tt.phase, node.Phase)
			assert.Equal(t, tt.message, node.Message)
			assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State(), "no interrupt recorded")
			assert.False(t, node.Fulfilled())
			n, _ := requeued.take()
			assert.Equal(t, 1, n, "a failed interrupt requeues")

			fail.Store(false)
			node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, tt.wantMessage, node.Message)
			assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
			assert.False(t, node.Fulfilled(), "the node waits for its task to stop")
			interruptedResult(t, fake, wf.Namespace, id)
			assert.Equal(t, dispatch.StopInterrupted, h.StopReason())
		})
	}
}
