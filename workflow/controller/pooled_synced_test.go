package controller

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kwait "k8s.io/apimachinery/pkg/util/wait"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// pooledTaskState is where a pooled node's task is when something ends it.
type pooledTaskState string

const (
	// taskQueued: enqueued, no worker has claimed it.
	taskQueued pooledTaskState = "queued"
	// taskClaimed: a worker runs it.
	taskClaimed pooledTaskState = "claimed"
	// taskReleased: a worker claimed it and released it, and none has since.
	taskReleased pooledTaskState = "released"
	// taskReclaimed: released, then claimed by another worker, which runs it.
	taskReclaimed pooledTaskState = "reclaimed"
)

// pooledEnding is a pooled node's task as a test has set it up: its node,
// and, while a worker runs it, the worker's handle and pod.
type pooledEnding struct {
	controller *WorkflowController
	fake       *dispatch.Fake
	wf         *wfv1.Workflow
	id         string
	h          dispatch.Claimed
	claimant   string
}

// setUpPooledTask takes pooledTimeoutWf's pooled step to state.
func setUpPooledTask(t *testing.T, state pooledTaskState) (context.Context, *pooledEnding) {
	t.Helper()
	wf := wfv1.MustUnmarshalWorkflow(pooledTimeoutWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	e := &pooledEnding{controller: controller, fake: fake, wf: wf}
	if state == taskQueued {
		e.id = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledTimeoutStep).ID
		observeTaskSet(ctx, t, controller, fake, wf.Namespace, e.id)
		return ctx, e
	}
	e.id, e.h = runPooledNode(ctx, t, controller, fake, wf, pooledTimeoutStep)
	e.claimant = claimantName
	if state == taskClaimed {
		return ctx, e
	}
	releasePooled(ctx, t, controller, wf, e.id, pooledTimeoutStep, e.h)
	e.h, e.claimant = nil, ""
	if state == taskReleased {
		return ctx, e
	}
	const other = "work-0123456789-fghij"
	e.h = claimPooledBy(ctx, t, fake, wf.Namespace, e.id, observeWorkerPod(ctx, t, controller, wf.Namespace, other, "other-pod-uid", 0))
	e.claimant = other
	observeResult(ctx, t, controller, fake, wf.Namespace, e.id)
	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledTimeoutStep)
	require.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())
	require.Equal(t, int32(1), node.PooledTask.Releases)
	return ctx, e
}

// TestPooledTaskResultSyncedAfterEveryEnding checks that every way a pooled
// node can end sets its TaskResultSynced, at once or, for a task that was
// interrupted, once its worker confirms the task stopped, so its workflow can
// complete; and each does so for a task that a worker has released, apart
// from pendingTimeout, which no longer applies once a worker has claimed the
// task.
func TestPooledTaskResultSyncedAfterEveryEnding(t *testing.T) {
	for _, tt := range []struct {
		name   string
		states []pooledTaskState
		end    func(ctx context.Context, t *testing.T, e *pooledEnding)
		// phase is the node's phase once ended, empty if it doesn't end.
		phase wfv1.NodePhase
		// notAfterRelease is whether the ending no longer applies once the
		// task is released.
		notAfterRelease bool
	}{
		{
			name:   "its own result",
			states: []pooledTaskState{taskClaimed, taskReclaimed},
			end: func(ctx context.Context, t *testing.T, e *pooledEnding) {
				require.NoError(t, e.h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded}))
				observeResult(ctx, t, e.controller, e.fake, e.wf.Namespace, e.id)
			},
			phase: wfv1.NodeSucceeded,
		},
		{
			name:   "a dead claimant",
			states: []pooledTaskState{taskClaimed, taskReclaimed},
			end: func(ctx context.Context, t *testing.T, e *pooledEnding) {
				deletePoolPod(ctx, t, e.controller, e.wf.Namespace, e.claimant)
			},
			phase: wfv1.NodeError,
		},
		{
			// A worker claims the task and is gone before its pod is cached.
			name:   "a dead claimant never cached",
			states: []pooledTaskState{taskQueued, taskReleased},
			end: func(ctx context.Context, t *testing.T, e *pooledEnding) {
				t.Setenv("RECENTLY_STARTED_POD_DURATION", "0s")
				claimPooledBy(ctx, t, e.fake, e.wf.Namespace, e.id, dispatch.Worker{Namespace: e.wf.Namespace, PodName: "work-0123456789-vwxyz", PodUID: "gone-pod-uid"})
				observeResult(ctx, t, e.controller, e.fake, e.wf.Namespace, e.id)
			},
			phase: wfv1.NodeError,
		},
		{
			name:   "the lease",
			states: []pooledTaskState{taskClaimed, taskReclaimed},
			end: func(ctx context.Context, t *testing.T, e *pooledEnding) {
				age(ctx, t, e.controller, e.wf, e.id, pastTimeout)
			},
			phase: wfv1.NodeFailed,
		},
		{
			name:   "stop",
			states: []pooledTaskState{taskQueued, taskClaimed, taskReleased},
			end: func(ctx context.Context, t *testing.T, e *pooledEnding) {
				stop(ctx, t, e.controller, e.wf)
			},
			phase: wfv1.NodeFailed,
		},
		{
			name:   "terminate",
			states: []pooledTaskState{taskQueued, taskClaimed, taskReleased},
			end: func(ctx context.Context, t *testing.T, e *pooledEnding) {
				terminate(ctx, t, e.controller, e.wf)
			},
			phase: wfv1.NodeFailed,
		},
		{
			name:   "the workflow deadline",
			states: []pooledTaskState{taskQueued, taskClaimed, taskReleased},
			end: func(ctx context.Context, t *testing.T, e *pooledEnding) {
				pastDeadline(ctx, t, e.controller, e.wf)
			},
			phase: wfv1.NodeFailed,
		},
		{
			name:   "timeout",
			states: []pooledTaskState{taskQueued, taskReleased},
			end: func(ctx context.Context, t *testing.T, e *pooledEnding) {
				age(ctx, t, e.controller, e.wf, e.id, pastTimeout)
			},
			phase: wfv1.NodeFailed,
		},
		{
			name:   "pendingTimeout",
			states: []pooledTaskState{taskQueued, taskReleased},
			end: func(ctx context.Context, t *testing.T, e *pooledEnding) {
				age(ctx, t, e.controller, e.wf, e.id, pastPendingTimeout)
			},
			phase:           wfv1.NodeFailed,
			notAfterRelease: true,
		},
		{
			name:   "a deleted TaskSet",
			states: []pooledTaskState{taskQueued, taskReleased},
			end: func(ctx context.Context, t *testing.T, e *pooledEnding) {
				t.Setenv("RECENTLY_STARTED_POD_DURATION", "0s")
				require.True(t, e.fake.DeleteTaskSet(e.wf.Namespace, pooledTaskSetName(e.id)))
				unobserveTaskSet(ctx, t, e.controller, e.wf.Namespace, e.id)
			},
			phase: wfv1.NodeError,
		},
	} {
		for _, state := range []pooledTaskState{taskQueued, taskClaimed, taskReleased, taskReclaimed} {
			if !slices.Contains(tt.states, state) {
				continue
			}
			t.Run(tt.name+", "+string(state), func(t *testing.T) {
				ctx, e := setUpPooledTask(t, state)
				tt.end(ctx, t, e)
				node := mustNode(t, reconcilePooled(ctx, t, e.controller, e.wf), pooledTimeoutStep)
				if tt.notAfterRelease && state == taskReleased {
					assert.Equal(t, wfv1.NodeRunning, node.Phase)
					assert.False(t, node.Fulfilled())
					return
				}
				require.Equal(t, tt.phase, node.Phase, node.Message)
				if node.PooledTask.State() == wfv1.PooledTaskInterrupted && !node.Fulfilled() {
					// The worker stops the task and confirms it.
					require.NotNil(t, e.h)
					require.Equal(t, dispatch.StopInterrupted, e.h.StopReason())
					require.NoError(t, e.h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeFailed}))
					observeResult(ctx, t, e.controller, e.fake, e.wf.Namespace, e.id)
					node = mustNode(t, reconcilePooled(ctx, t, e.controller, e.wf), pooledTimeoutStep)
					assert.Equal(t, tt.phase, node.Phase, "the first ending wins")
				}
				require.NotNil(t, node.TaskResultSynced)
				assert.True(t, *node.TaskResultSynced)
				assert.True(t, node.Fulfilled())
			})
		}
	}
}

// deletePoolPod deletes the pool pod name and waits for the pool pod
// informer to drop it.
func deletePoolPod(ctx context.Context, t *testing.T, controller *WorkflowController, namespace, name string) {
	t.Helper()
	require.NoError(t, controller.kubeclientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{}))
	require.NoError(t, kwait.PollUntilContextTimeout(ctx, time.Millisecond, 10*time.Second, true, func(context.Context) (bool, error) {
		_, exists, err := controller.poolPodInformer.GetStore().GetByKey(namespace + "/" + name)
		return !exists, err
	}))
}
