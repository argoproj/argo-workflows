package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// pooledTimeoutWf runs a pooled step with a timeout and a pendingTimeout
// beside a retried one with the same.
const pooledTimeoutWf = `
metadata:
  name: pooled-timeout
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: work
      - name: r
        template: retried
  - name: work
    pool: {}
    timeout: 2h
    pendingTimeout: 1h
    container:
      image: my-image
  - name: retried
    pool: {}
    timeout: 2h
    pendingTimeout: 1h
    retryStrategy:
      limit: 1
    container:
      image: my-image
`

const (
	pooledTimeoutStep     = "pooled-timeout[0].a"
	pooledTimeoutRetry    = "pooled-timeout[0].r"
	pooledTimeoutAttempt  = "pooled-timeout[0].r(0)"
	pooledTimeoutAttempt1 = "pooled-timeout[0].r(1)"
	// Past pendingTimeout but not timeout, and past both.
	pastPendingTimeout = 90 * time.Minute
	pastTimeout        = 3 * time.Hour
)

// age moves the stored pooled node's start back by d, as d passing would:
// its ledger's absolute deadline, which dispatch counted from its start,
// moves with it.
func age(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, id string, d time.Duration) {
	t.Helper()
	updateStoredNode(ctx, t, controller, wf, id, func(node *wfv1.NodeStatus) {
		node.StartedAt = metav1.NewTime(node.StartedAt.Add(-d))
		if ledger := node.PooledTask; ledger != nil && ledger.Deadline != nil {
			ledger.Deadline = &metav1.Time{Time: ledger.Deadline.Add(-d)}
		}
	})
}

// releasePooled has the worker holding h release the node's task, and the
// controller record the release.
func releasePooled(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow, id, name string, h dispatch.Claimed) {
	t.Helper()
	require.NoError(t, h.Release(ctx))
	forgetResult(ctx, t, controller, wf.Namespace, id)
	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), name)
	require.True(t, node.PooledTask.Released())
	require.Equal(t, wfv1.NodeRunning, node.Phase)
}

// TestPooledTimeoutDropsQueuedTask checks that a queued pooled node no
// worker has claimed fails with pods' message once past its template's
// timeout or pendingTimeout, its TaskSet dropped and its node finished at
// once, whether or not its template is retried, in which case the next
// attempt is enqueued.
func TestPooledTimeoutDropsQueuedTask(t *testing.T) {
	for _, tt := range []struct {
		name    string
		node    string
		elapsed time.Duration
	}{
		{name: "pendingTimeout", node: pooledTimeoutStep, elapsed: pastPendingTimeout},
		{name: "timeout", node: pooledTimeoutStep, elapsed: pastTimeout},
		{name: "retried, pendingTimeout", node: pooledTimeoutAttempt, elapsed: pastPendingTimeout},
		{name: "retried, timeout", node: pooledTimeoutAttempt, elapsed: pastTimeout},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledTimeoutWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			id := mustNode(t, reconcilePooled(ctx, t, controller, wf), tt.node).ID
			require.True(t, hasTaskSet(fake, wf.Namespace, id))
			r := recordRequeues(controller)
			woc := reconcilePooled(ctx, t, controller, wf)
			require.Equal(t, wfv1.NodePending, mustNode(t, woc, tt.node).Phase, "within its timeouts")
			_, after := r.take()
			assert.True(t, requeuedAfter(after, time.Hour), "requeued for its pendingTimeout: %v", after)

			age(ctx, t, controller, wf, id, tt.elapsed)
			woc = reconcilePooled(ctx, t, controller, wf)
			node := mustNode(t, woc, tt.node)
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, "timeout", node.Message)
			assert.True(t, node.Fulfilled(), "nothing runs an unclaimed task")
			assert.Equal(t, wfv1.PooledTaskEnqueued, node.PooledTask.State())
			assert.False(t, hasTaskSet(fake, wf.Namespace, id), "its TaskSet is dropped")
			if tt.node == pooledTimeoutAttempt {
				next := mustNode(t, woc, pooledTimeoutAttempt1)
				assert.Equal(t, wfv1.NodePending, next.Phase, "the retry enqueues the next attempt")
				assert.True(t, hasTaskSet(fake, wf.Namespace, next.ID))
			}
		})
	}
}

// TestPooledTimeoutReleasedTask checks that, once released, a pooled node is
// bounded only by its absolute deadline, which holds its timeout, not by its
// pendingTimeout, which covers only the wait for its first claim. Past it,
// its TaskSet is dropped and it fails with the deadline message a claimed
// task gets, finished at once.
func TestPooledTimeoutReleasedTask(t *testing.T) {
	for _, name := range []string{pooledTimeoutStep, pooledTimeoutAttempt} {
		t.Run(name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledTimeoutWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			id, h := runPooledNode(ctx, t, controller, fake, wf, name)
			releasePooled(ctx, t, controller, wf, id, name, h)

			age(ctx, t, controller, wf, id, pastPendingTimeout)
			r := recordRequeues(controller)
			node := mustNode(t, reconcilePooled(ctx, t, controller, wf), name)
			assert.Equal(t, wfv1.NodeRunning, node.Phase, "a released node is Running, which pendingTimeout, a Pending node's, doesn't bound")
			_, after := r.take()
			assert.True(t, requeuedAfter(after, 2*time.Hour-pastPendingTimeout), "requeued for its deadline: %v", after)
			assert.True(t, hasTaskSet(fake, wf.Namespace, id))

			age(ctx, t, controller, wf, id, pastTimeout-pastPendingTimeout)
			node = mustNode(t, reconcilePooled(ctx, t, controller, wf), name)
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, "Step exceeded its deadline", node.Message)
			assert.True(t, node.Fulfilled(), "nothing runs a released task")
			assert.Equal(t, wfv1.PooledTaskEnqueued, node.PooledTask.State())
			assert.False(t, hasTaskSet(fake, wf.Namespace, id))
		})
	}
}

// TestPooledTimeoutClaimedTask checks a claimed task isn't dropped at its
// timeout: its worker stops it at its deadline, and its lease backs that up.
func TestPooledTimeoutClaimedTask(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledTimeoutWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	id, _ := runPooledNode(ctx, t, controller, fake, wf, pooledTimeoutAttempt)
	age(ctx, t, controller, wf, id, 2*time.Hour+time.Minute)
	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledTimeoutAttempt)
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())
	assert.True(t, hasTaskSet(fake, wf.Namespace, id))
}

// TestPooledTimeoutDropFindsClaim checks a per-node drop racing a claim: a
// claim on a timed-out task that the controller hasn't recorded, because its
// claimant pod isn't cached, is found by the drop and interrupted, the node
// waiting for the task to stop; one the drop finds completed ends the node
// with its result, the first ending.
func TestPooledTimeoutDropFindsClaim(t *testing.T) {
	for _, tt := range []struct {
		name     string
		node     string
		complete bool
	}{
		{name: "claim", node: pooledTimeoutStep},
		{name: "retried, claim", node: pooledTimeoutAttempt},
		{name: "completed claim", node: pooledTimeoutStep, complete: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledTimeoutWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			id := mustNode(t, reconcilePooled(ctx, t, controller, wf), tt.node).ID
			worker := dispatch.Worker{Namespace: wf.Namespace, PodName: "work-0123456789-fghij", PodUID: "uncached-uid"}
			h := claimPooledBy(ctx, t, fake, wf.Namespace, id, worker)
			if tt.complete {
				require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{ExitCode: new("0")}}))
			}

			age(ctx, t, controller, wf, id, pastTimeout)
			node := mustNode(t, reconcilePooled(ctx, t, controller, wf), tt.node)
			assert.False(t, hasTaskSet(fake, wf.Namespace, id))
			if tt.complete {
				assert.Equal(t, wfv1.NodeSucceeded, node.Phase, "the task ended before the drop")
				assert.True(t, node.Fulfilled())
				return
			}
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, "timeout", node.Message)
			assert.False(t, node.Fulfilled(), "the node waits for its task to stop")
			assert.Equal(t, wfv1.PooledTaskInterrupted, node.PooledTask.State())
			assert.Equal(t, h.Claim().UID, node.PooledTask.ClaimUID)
			interruptedResult(t, fake, wf.Namespace, id)
			assert.Equal(t, dispatch.StopInterrupted, h.StopReason())

			require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded}))
			observeResult(ctx, t, controller, fake, wf.Namespace, id)
			node = mustNode(t, reconcilePooled(ctx, t, controller, wf), tt.node)
			assert.True(t, node.Fulfilled())
			assert.Equal(t, wfv1.NodeFailed, node.Phase, "the first ending wins")
		})
	}
}

// TestPooledTimeoutDropRetried checks a timed-out task whose drop fails is
// left as it is and dropped at a later reconcile.
func TestPooledTimeoutDropRetried(t *testing.T) {
	for _, name := range []string{pooledTimeoutStep, pooledTimeoutAttempt} {
		t.Run(name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledTimeoutWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			id := mustNode(t, reconcilePooled(ctx, t, controller, wf), name).ID
			fake.SetFault(func(op dispatch.Op) error {
				if op.Verb == "delete" && op.Resource == dispatch.ResourceTaskSets {
					return assert.AnError
				}
				return nil
			})

			age(ctx, t, controller, wf, id, pastTimeout)
			node := mustNode(t, reconcilePooled(ctx, t, controller, wf), name)
			assert.Equal(t, wfv1.NodePending, node.Phase)
			assert.True(t, hasTaskSet(fake, wf.Namespace, id))

			fake.SetFault(nil)
			node = mustNode(t, reconcilePooled(ctx, t, controller, wf), name)
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Equal(t, "timeout", node.Message)
			assert.True(t, node.Fulfilled())
			assert.False(t, hasTaskSet(fake, wf.Namespace, id))
		})
	}
}

// pooledHookTimeoutWf runs a pooled step with a pooled exit hook that has a
// pendingTimeout.
const pooledHookTimeoutWf = `
metadata:
  name: pooled-hook-timeout
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: work
        hooks:
          exit:
            template: hook
  - name: work
    pool: {}
    container:
      image: my-image
  - name: hook
    pool: {}
    pendingTimeout: 1h
    container:
      image: my-image
`

// TestPooledTimeoutHook checks a queued pooled hook past its pendingTimeout
// errors rather than fails, as a pending pod's hook does, its task dropped.
func TestPooledTimeoutHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledHookTimeoutWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	id := mustNode(t, reconcilePooled(ctx, t, controller, wf), "pooled-hook-timeout[0].a").ID
	finishPooled(ctx, t, controller, fake, wf.Namespace, id)
	reconcilePooled(ctx, t, controller, wf)
	const hook = "pooled-hook-timeout[0].a.onExit"
	hookID := mustNode(t, reconcilePooled(ctx, t, controller, wf), hook).ID
	require.True(t, hasTaskSet(fake, wf.Namespace, hookID))

	age(ctx, t, controller, wf, hookID, pastPendingTimeout)
	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), hook)
	assert.Equal(t, wfv1.NodeError, node.Phase)
	assert.Equal(t, "timeout", node.Message)
	assert.True(t, node.Fulfilled())
	assert.False(t, hasTaskSet(fake, wf.Namespace, hookID))
}

// pooledMaxDurationWf runs a pooled step whose retries are bounded by a
// maxDuration, its execution deadline.
const pooledMaxDurationWf = `
metadata:
  name: pooled-max-duration
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: r
        template: retried
  - name: retried
    pool: {}
    retryStrategy:
      limit: 2
      backoff:
        duration: "0"
        maxDuration: 1h
    container:
      image: my-image
`

// TestPooledExecutionDeadlineDropsQueuedTask is the drop of a queued retry
// attempt past its execution deadline, from its retry strategy's
// maxDuration: its TaskSet is dropped and it fails with the deadline message
// its worker would have given.
func TestPooledExecutionDeadlineDropsQueuedTask(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledMaxDurationWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	first := mustNode(t, reconcilePooled(ctx, t, controller, wf), "pooled-max-duration[0].r(0)")
	h := claimPooled(ctx, t, controller, fake, wf.Namespace, first.ID)
	require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: "failed"}))
	observeResult(ctx, t, controller, fake, wf.Namespace, first.ID)
	reconcilePooled(ctx, t, controller, wf)
	const attempt = "pooled-max-duration[0].r(1)"
	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), attempt)
	require.NotNil(t, node.PooledTask.Deadline)
	assert.WithinDuration(t, first.StartedAt.Add(time.Hour), node.PooledTask.Deadline.Time, time.Second)

	age(ctx, t, controller, wf, node.ID, 2*time.Hour)
	node = mustNode(t, reconcilePooled(ctx, t, controller, wf), attempt)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
	assert.Equal(t, "Step exceeded its deadline", node.Message)
	assert.True(t, node.Fulfilled())
	assert.False(t, hasTaskSet(fake, wf.Namespace, node.ID))
}
