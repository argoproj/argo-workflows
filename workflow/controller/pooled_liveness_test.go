package controller

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// The claimant runPooled's worker claims with.
const (
	claimantName = "work-0123456789-abcde"
	claimantUID  = types.UID("worker-pod-uid")
)

// runPooled enqueues pooledDispatchWf's pooled step, has a cached worker
// claim it, and returns the node's ID and the worker's handle once the
// controller has recorded the claim. The controller's caches hold the
// node's TaskSet, the claim and the claimant pod.
func runPooled(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, wf *wfv1.Workflow) (string, dispatch.Claimed) {
	t.Helper()
	return runPooledNode(ctx, t, controller, fake, wf, pooledStep)
}

// runPooledNode is runPooled for the workflow's pooled node name.
func runPooledNode(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, wf *wfv1.Workflow, name string) (string, dispatch.Claimed) {
	t.Helper()
	woc := reconcilePooled(ctx, t, controller, wf)
	id := mustNode(t, woc, name).ID
	observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
	worker := observeWorkerPod(ctx, t, controller, wf.Namespace, claimantName, claimantUID, 0)
	h := claimPooledBy(ctx, t, fake, wf.Namespace, id, worker)
	observeResult(ctx, t, controller, fake, wf.Namespace, id)
	woc = reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, name)
	require.Equal(t, wfv1.NodeRunning, node.Phase)
	require.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())
	return id, h
}

// updateClaimant changes runPooled's claimant pod with change and waits for
// the pool pod informer to hold the change, as cached tells.
func updateClaimant(ctx context.Context, t *testing.T, controller *WorkflowController, namespace string, change func(*apiv1.Pod), cached func(*apiv1.Pod) bool) {
	t.Helper()
	pods := controller.kubeclientset.CoreV1().Pods(namespace)
	pod, err := pods.Get(ctx, claimantName, metav1.GetOptions{})
	require.NoError(t, err)
	change(pod)
	_, err = pods.Update(ctx, pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForInformer(ctx, controller.poolPodInformer, pod, func(obj any) bool { return cached(obj.(*apiv1.Pod)) })
}

// crashWorker makes runPooled's claimant's worker container, as the pool pod
// informer holds it, wait to restart after restarts restarts, as one that
// crash loops does.
func crashWorker(ctx context.Context, t *testing.T, controller *WorkflowController, namespace string, restarts int32) {
	t.Helper()
	updateClaimant(ctx, t, controller, namespace, func(pod *apiv1.Pod) {
		s := workerStatus(pod)
		s.RestartCount = restarts
		s.State = apiv1.ContainerState{Waiting: &apiv1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}
	}, func(pod *apiv1.Pod) bool { return workerRestartCount(pod) == restarts && !workerRunning(pod) })
}

// TestPooledClaimantGone checks that the claimant pod deleted, evicted,
// replaced, or its worker container restarted since the claim and not
// running, ends the node as Error at the next reconcile, with its task result
// synced. A pod being deleted, or whose restarted worker container runs, still
// runs its task, or fails it.
func TestPooledClaimantGone(t *testing.T) {
	for _, tt := range []struct {
		name string
		kill func(ctx context.Context, t *testing.T, controller *WorkflowController, namespace string)
		// message is the node's, empty if it keeps running.
		message string
	}{
		{
			name: "deleted",
			kill: func(ctx context.Context, t *testing.T, controller *WorkflowController, namespace string) {
				deletePoolPod(ctx, t, controller, namespace, claimantName)
			},
			message: "worker pod " + claimantName + " deleted",
		},
		{
			name: "evicted",
			kill: func(ctx context.Context, t *testing.T, controller *WorkflowController, namespace string) {
				updateClaimant(ctx, t, controller, namespace, func(pod *apiv1.Pod) {
					pod.Status.Phase = apiv1.PodFailed
					pod.Status.Reason = "Evicted"
					pod.Status.Message = "The node was low on resource: memory."
				}, func(pod *apiv1.Pod) bool { return pod.Status.Phase == apiv1.PodFailed })
			},
			message: "worker pod " + claimantName + " failed: Evicted: The node was low on resource: memory.",
		},
		{
			name: "replaced",
			kill: func(ctx context.Context, t *testing.T, controller *WorkflowController, namespace string) {
				deletePoolPod(ctx, t, controller, namespace, claimantName)
				observeWorkerPod(ctx, t, controller, namespace, claimantName, "replacement-pod-uid", 0)
			},
			message: "worker pod " + claimantName + " deleted",
		},
		{
			name: "worker restarted",
			kill: func(ctx context.Context, t *testing.T, controller *WorkflowController, namespace string) {
				crashWorker(ctx, t, controller, namespace, 1)
			},
			message: "worker container of pod " + claimantName + " restarted",
		},
		{
			// Its run fails the claim before it claims anything.
			name: "worker restarted and running",
			kill: func(ctx context.Context, t *testing.T, controller *WorkflowController, namespace string) {
				observeWorkerPod(ctx, t, controller, namespace, claimantName, claimantUID, 1)
			},
		},
		{
			name: "being deleted",
			kill: func(ctx context.Context, t *testing.T, controller *WorkflowController, namespace string) {
				updateClaimant(ctx, t, controller, namespace, func(pod *apiv1.Pod) {
					pod.DeletionTimestamp = &metav1.Time{Time: time.Now()}
				}, func(pod *apiv1.Pod) bool { return pod.DeletionTimestamp != nil })
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			runPooled(ctx, t, controller, fake, wf)

			tt.kill(ctx, t, controller, wf.Namespace)
			woc := reconcilePooled(ctx, t, controller, wf)
			node := mustNode(t, woc, pooledStep)
			if tt.message == "" {
				assert.Equal(t, wfv1.NodeRunning, node.Phase)
				assert.False(t, *node.TaskResultSynced)
				return
			}
			assert.Equal(t, wfv1.NodeError, node.Phase)
			assert.Equal(t, tt.message, node.Message)
			assert.True(t, *node.TaskResultSynced)
			assert.True(t, node.Fulfilled())
			assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State(), "the ledger keeps the claim the node ended with")
		})
	}
}

// TestPooledClaimByRestartedWorker checks that a claim made by a worker
// container that has just restarted, recorded with the restart count the pool
// pod cache still had from before the restart, doesn't end its node when the
// cache catches up: its worker runs the task. The node ends only once that
// worker container stops running.
func TestPooledClaimByRestartedWorker(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	id := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep).ID
	observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
	// The cache has the pod from before its worker container restarted.
	worker := observeWorkerPod(ctx, t, controller, wf.Namespace, claimantName, claimantUID, 0)
	claimPooledBy(ctx, t, fake, wf.Namespace, id, worker)
	observeResult(ctx, t, controller, fake, wf.Namespace, id)
	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
	require.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())
	require.Equal(t, int32(0), node.PooledTask.WorkerRestartCount)

	observeWorkerPod(ctx, t, controller, wf.Namespace, claimantName, claimantUID, 1)
	node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
	assert.Equal(t, wfv1.NodeRunning, node.Phase, "the restarted worker runs the task")
	assert.False(t, node.Fulfilled())

	crashWorker(ctx, t, controller, wf.Namespace, 1)
	node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
	assert.Equal(t, wfv1.NodeError, node.Phase)
	assert.Equal(t, "worker container of pod "+claimantName+" restarted", node.Message)
}

// TestPooledClaimedNodeSurvivesControllerRestart checks that a controller
// started fresh, from a workflow whose pooled node's ledger holds a claim, its
// TaskSet, the claim and the claimant pod, doesn't end the node: it waits for
// its caches to sync before reconciling, so the claim and claimant are cached
// and nothing is read live.
func TestPooledClaimedNodeSurvivesControllerRestart(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, before, fake := newPooledDispatchController(t, true, wf)
	id, _ := runPooled(ctx, t, before, fake, wf)
	stored, err := before.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	taskSet, err := before.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(wf.Namespace).Get(ctx, pooledTaskSetName(id), metav1.GetOptions{})
	require.NoError(t, err)
	claim, err := before.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(wf.Namespace).Get(ctx, id, metav1.GetOptions{})
	require.NoError(t, err)
	claimant, err := before.kubeclientset.CoreV1().Pods(wf.Namespace).Get(ctx, claimantName, metav1.GetOptions{})
	require.NoError(t, err)

	cancel, after := newController(ctx, stored, taskSet, claim, func(wfc *WorkflowController) {
		wfc.dispatcher = fake.Dispatcher()
		wfc.Config.HotWorkers = &config.HotWorkersConfig{Enabled: true}
		_, err := wfc.kubeclientset.CoreV1().Pods(wf.Namespace).Create(ctx, claimant, metav1.CreateOptions{})
		require.NoError(t, err)
	})
	t.Cleanup(cancel)
	ops := countOps(fake)
	node := mustNode(t, reconcilePooled(ctx, t, after, wf), pooledStep)
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.Empty(t, node.Message)
	assert.False(t, node.Fulfilled())
	assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())
	assert.Equal(t, claim.UID, node.PooledTask.ClaimUID)
	assert.Zero(t, ops.Load(), "nothing read live")
}

// TestPooledClaimCompletedBeforeClaimantGone checks that a result completed
// just before its claimant pod went, which the result cache hasn't seen, is
// read live and gives the node its phase and outputs, rather than an Error.
func TestPooledClaimCompletedBeforeClaimantGone(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	_, h := runPooled(ctx, t, controller, fake, wf)

	require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{ExitCode: new("0")}}))
	deletePoolPod(ctx, t, controller, wf.Namespace, claimantName)
	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.Empty(t, node.Message)
	require.NotNil(t, node.Outputs)
	assert.Equal(t, new("0"), node.Outputs.ExitCode)
	assert.True(t, *node.TaskResultSynced)
}

// TestPooledDeadClaimantReadFails checks that a claimant gone from the cache
// whose claim can't be read live isn't taken for dead: the node runs on and the
// workflow is requeued, and once the read succeeds the node ends.
func TestPooledDeadClaimantReadFails(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	runPooled(ctx, t, controller, fake, wf)
	deletePoolPod(ctx, t, controller, wf.Namespace, claimantName)
	fake.SetFault(func(op dispatch.Op) error {
		if op.Verb == "get" && op.Resource == dispatch.ResourceResults {
			return errors.New("injected")
		}
		return nil
	})
	requeued := recordRequeues(controller)

	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.False(t, node.Fulfilled())
	n, _ := requeued.take()
	assert.Equal(t, 1, n)

	fake.SetFault(nil)
	node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
	assert.Equal(t, wfv1.NodeError, node.Phase)
	assert.Equal(t, "worker pod "+claimantName+" deleted", node.Message)
}

// TestPooledDeadClaimantReleasedAndClaimedAgain checks that a claimant gone
// from the cache that released its task, which another worker has claimed
// since, before the result cache saw either, doesn't fail the node: the live
// read finds the claim replaced, and the workflow is requeued for the cache
// to show the new claim, which is then recorded.
func TestPooledDeadClaimantReleasedAndClaimedAgain(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	id, h := runPooled(ctx, t, controller, fake, wf)
	require.NoError(t, h.Release(ctx))
	other := observeWorkerPod(ctx, t, controller, wf.Namespace, "work-0123456789-other", "other-uid", 0)
	reclaim := claimPooledBy(ctx, t, fake, wf.Namespace, id, other)
	deletePoolPod(ctx, t, controller, wf.Namespace, claimantName)
	requeued := recordRequeues(controller)

	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.Empty(t, node.Message)
	assert.Equal(t, claimantName, node.PooledTask.WorkerPodName, "the cache still has the first claim")
	n, _ := requeued.take()
	assert.Equal(t, 1, n)

	observeResult(ctx, t, controller, fake, wf.Namespace, id)
	node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.Equal(t, reclaim.Claim().UID, node.PooledTask.ClaimUID)
	assert.Equal(t, other.PodName, node.PooledTask.WorkerPodName)
	assert.Equal(t, int32(1), node.PooledTask.Releases)
}

// TestPooledClaimantReleasedIsNotDead checks that a claimant that released its
// task during scale-down has deleted its claim, so its pod going is a
// release, recorded once the result cache sees the claim gone, not a death.
func TestPooledClaimantReleasedIsNotDead(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	id, h := runPooled(ctx, t, controller, fake, wf)

	require.NoError(t, h.Release(ctx))
	deletePoolPod(ctx, t, controller, wf.Namespace, claimantName)
	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodeRunning, node.Phase, "the result cache still has the claim")
	assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())

	forgetResult(ctx, t, controller, wf.Namespace, id)
	woc = reconcilePooled(ctx, t, controller, wf)
	node = mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.True(t, node.PooledTask.Released())
	assert.Empty(t, node.Message)
}

// TestPooledUnrecordedClaim checks a claim whose claimant pod isn't
// cached, which ingest leaves unrecorded: within the grace, or while the pod
// exists, it doesn't fail the node; once the grace has passed and the pod is
// gone, the claimant died before the cache saw it.
func TestPooledUnrecordedClaim(t *testing.T) {
	uncached := dispatch.Worker{Namespace: "default", PodName: "work-0123456789-fghij", PodUID: "uncached-uid"}
	for _, tt := range []struct {
		name     string
		grace    string
		podInAPI bool
		complete bool
		want     wfv1.NodePhase
	}{
		{name: "within the grace", grace: "1h", want: wfv1.NodePending},
		{name: "pod not cached yet", grace: "0s", podInAPI: true, want: wfv1.NodePending},
		{name: "pod gone", grace: "0s", want: wfv1.NodeError},
		{name: "pod gone, claim completed", grace: "0s", complete: true, want: wfv1.NodeSucceeded},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("RECENTLY_STARTED_POD_DURATION", tt.grace)
			wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			woc := reconcilePooled(ctx, t, controller, wf)
			id := mustNode(t, woc, pooledStep).ID
			observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
			if tt.podInAPI {
				// The API server has it, the cache has yet to see it.
				observeWorkerPod(ctx, t, controller, wf.Namespace, uncached.PodName, uncached.PodUID, 0)
				obj, _, err := controller.poolPodInformer.GetStore().GetByKey(wf.Namespace + "/" + uncached.PodName)
				require.NoError(t, err)
				require.NoError(t, controller.poolPodInformer.GetStore().Delete(obj))
			}
			h := claimPooledBy(ctx, t, fake, wf.Namespace, id, uncached)
			observeResult(ctx, t, controller, fake, wf.Namespace, id)
			if tt.complete {
				require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded}))
			}

			requeued := recordRequeues(controller)
			woc = reconcilePooled(ctx, t, controller, wf)
			node := mustNode(t, woc, pooledStep)
			assert.Equal(t, tt.want, node.Phase)
			_, after := requeued.take()
			switch tt.want {
			case wfv1.NodePending:
				assert.Equal(t, wfv1.PooledTaskEnqueued, node.PooledTask.State())
				assert.True(t, *node.TaskResultSynced, "nothing of the claim is applied")
				if tt.grace == "1h" {
					assert.True(t, requeuedAfter(after, time.Hour), "requeued for the end of the grace, got %v", after)
				}
			case wfv1.NodeError:
				assert.Equal(t, "worker pod "+uncached.PodName+" deleted", node.Message)
				assert.True(t, *node.TaskResultSynced)
				assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State(), "the claim the node ended with is recorded")
				assert.Equal(t, h.Claim().UID, node.PooledTask.ClaimUID)
				assert.Equal(t, uncached.PodName, node.PooledTask.WorkerPodName)
				assert.Zero(t, node.PooledTask.WorkerRestartCount)
			case wfv1.NodeSucceeded:
				assert.True(t, *node.TaskResultSynced)
				assert.Equal(t, h.Claim().UID, node.PooledTask.ClaimUID)
			}
		})
	}
}

// TestPooledTaskSetLost checks that a TaskSet missing from the cache
// within the grace since its enqueue doesn't fail the node, as it may just
// not be cached yet; after it, the TaskSet was deleted, which ends the node
// as Error with its task result synced, whether it was enqueued, released,
// or claimed and its claim deleted since. A claimed node whose claim
// remains keeps running: its worker can still report it.
func TestPooledTaskSetLost(t *testing.T) {
	lost := func(t *testing.T, woc *wfOperationCtx, id string) {
		t.Helper()
		node := mustNode(t, woc, pooledStep)
		assert.Equal(t, wfv1.NodeError, node.Phase)
		assert.Equal(t, "WorkflowTaskSet "+id+"-task deleted", node.Message)
		assert.True(t, *node.TaskResultSynced)
		assert.True(t, node.Fulfilled())
	}

	t.Run("not cached yet", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
		ctx, controller, _ := newPooledDispatchController(t, true, wf)
		reconcilePooled(ctx, t, controller, wf)
		woc := reconcilePooled(ctx, t, controller, wf)
		node := mustNode(t, woc, pooledStep)
		assert.Equal(t, wfv1.NodePending, node.Phase)
		assert.Empty(t, node.Message)
	})

	t.Run("deleted while enqueued", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
		ctx, controller, fake := newPooledDispatchController(t, true, wf)
		woc := reconcilePooled(ctx, t, controller, wf)
		id := mustNode(t, woc, pooledStep).ID
		observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
		require.True(t, fake.DeleteTaskSet(wf.Namespace, pooledTaskSetName(id)))
		unobserveTaskSet(ctx, t, controller, wf.Namespace, id)

		t.Setenv("RECENTLY_STARTED_POD_DURATION", "1h")
		requeued := recordRequeues(controller)
		woc = reconcilePooled(ctx, t, controller, wf)
		assert.Equal(t, wfv1.NodePending, mustNode(t, woc, pooledStep).Phase, "within the grace")
		_, after := requeued.take()
		assert.True(t, requeuedAfter(after, time.Hour), "requeued for the end of the grace, got %v", after)
		t.Setenv("RECENTLY_STARTED_POD_DURATION", "0s")
		lost(t, reconcilePooled(ctx, t, controller, wf), id)
	})

	t.Run("deleted while released", func(t *testing.T) {
		t.Setenv("RECENTLY_STARTED_POD_DURATION", "0s")
		wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
		ctx, controller, fake := newPooledDispatchController(t, true, wf)
		id, h := runPooled(ctx, t, controller, fake, wf)
		require.NoError(t, h.Release(ctx))
		forgetResult(ctx, t, controller, wf.Namespace, id)
		woc := reconcilePooled(ctx, t, controller, wf)
		require.True(t, mustNode(t, woc, pooledStep).PooledTask.Released())

		require.True(t, fake.DeleteTaskSet(wf.Namespace, pooledTaskSetName(id)))
		unobserveTaskSet(ctx, t, controller, wf.Namespace, id)
		lost(t, reconcilePooled(ctx, t, controller, wf), id)
	})

	t.Run("deleted while claimed", func(t *testing.T) {
		t.Setenv("RECENTLY_STARTED_POD_DURATION", "0s")
		wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
		ctx, controller, fake := newPooledDispatchController(t, true, wf)
		id, h := runPooled(ctx, t, controller, fake, wf)
		require.True(t, fake.DeleteTaskSet(wf.Namespace, pooledTaskSetName(id)))
		unobserveTaskSet(ctx, t, controller, wf.Namespace, id)
		woc := reconcilePooled(ctx, t, controller, wf)
		assert.Equal(t, wfv1.NodeRunning, mustNode(t, woc, pooledStep).Phase, "the claim runs on")

		require.NoError(t, h.Release(ctx))
		forgetResult(ctx, t, controller, wf.Namespace, id)
		lost(t, reconcilePooled(ctx, t, controller, wf), id)
	})
}

// TestPooledAckedTaskSetIsNotLost checks that the TaskSet of a finished node,
// deleted by its ack, is never taken for a lost one.
func TestPooledAckedTaskSetIsNotLost(t *testing.T) {
	t.Setenv("RECENTLY_STARTED_POD_DURATION", "0s")
	wf := wfv1.MustUnmarshalWorkflow(pooledAckWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	id := mustNode(t, woc, pooledAckStep).ID
	observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
	finishPooled(ctx, t, controller, fake, wf.Namespace, id)
	woc = reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledAckStep)
	require.Equal(t, wfv1.NodeSucceeded, node.Phase)
	require.False(t, hasTaskSet(fake, wf.Namespace, id), "acked")

	unobserveTaskSet(ctx, t, controller, wf.Namespace, id)
	woc = reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, node, mustNode(t, woc, pooledAckStep))
}

// countWorkflowUpdates counts the workflow writes the controller makes.
func countWorkflowUpdates(controller *WorkflowController) *atomic.Int32 {
	var n atomic.Int32
	controller.wfclientset.(*fakewfclientset.Clientset).PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
		n.Add(1)
		return false, nil, nil
	})
	return &n
}

// TestPooledPoolHealthInMessage checks that while its pool is degraded an
// enqueued node stays Pending with the reason in its message, written only
// when it changes, and cleared once the pool is OK.
func TestPooledPoolHealthInMessage(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	controller.poolHealth = fake
	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	observeTaskSet(ctx, t, controller, fake, wf.Namespace, node.ID)
	pool := node.PooledTask.PoolName
	// As the pool controller changes a pool's health.
	setHealth := func(state dispatch.HealthState, reason string) {
		fake.SetHealth(wf.Namespace, node.PooledTask.PoolHash, dispatch.Health{State: state, Reason: reason})
	}

	setHealth(dispatch.HealthDegraded, `ImagePullBackOff: container "main" can't pull image "my-image"`)
	woc = reconcilePooled(ctx, t, controller, wf)
	node = mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	assert.Equal(t, `pool "`+pool+`" is degraded: ImagePullBackOff: container "main" can't pull image "my-image"`, node.Message)

	updates := countWorkflowUpdates(controller)
	woc = reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, node, mustNode(t, woc, pooledStep))
	assert.Zero(t, updates.Load(), "an unchanged health isn't written again")

	setHealth(dispatch.HealthDegraded, "ReplicaFailure: exceeded quota")
	woc = reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, `pool "`+pool+`" is degraded: ReplicaFailure: exceeded quota`, mustNode(t, woc, pooledStep).Message)
	assert.Equal(t, int32(1), updates.Load())

	setHealth(dispatch.HealthOK, "")
	woc = reconcilePooled(ctx, t, controller, wf)
	node = mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	assert.Empty(t, node.Message)
}

// TestPooledPoolUnbuildable checks that a pool reported unbuildable ends its
// enqueued nodes as Error with the reason, with their task results synced,
// and their TaskSets are then acked. A claimed node, or one with a claim
// not yet recorded, is being served, and is left alone, as is every node
// while the controller has no pool health.
func TestPooledPoolUnbuildable(t *testing.T) {
	const reason = `the pool spec is for controller instance "a", not "b"`
	enqueue := func(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, wf *wfv1.Workflow) {
		t.Helper()
		woc := reconcilePooled(ctx, t, controller, wf)
		observeTaskSet(ctx, t, controller, fake, wf.Namespace, mustNode(t, woc, pooledStep).ID)
	}
	for _, tt := range []struct {
		name   string
		health bool
		start  func(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, wf *wfv1.Workflow)
		want   wfv1.NodePhase
	}{
		{name: "enqueued", health: true, start: enqueue, want: wfv1.NodeError},
		{name: "no pool health", start: enqueue, want: wfv1.NodePending},
		{
			name:   "claimed",
			health: true,
			start: func(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, wf *wfv1.Workflow) {
				runPooled(ctx, t, controller, fake, wf)
			},
			want: wfv1.NodeRunning,
		},
		{
			name:   "claim not recorded yet",
			health: true,
			start: func(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, wf *wfv1.Workflow) {
				enqueue(ctx, t, controller, fake, wf)
				id := wf.NodeID(pooledStep)
				claimPooledBy(ctx, t, fake, wf.Namespace, id, dispatch.Worker{Namespace: wf.Namespace, PodName: "work-0123456789-fghij", PodUID: "uncached-uid"})
				observeResult(ctx, t, controller, fake, wf.Namespace, id)
			},
			want: wfv1.NodePending,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			tt.start(ctx, t, controller, fake, wf)
			node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
			if tt.health {
				controller.poolHealth = fake
			}
			fake.SetHealth(wf.Namespace, node.PooledTask.PoolHash, dispatch.Health{State: dispatch.HealthUnbuildable, Reason: reason})

			node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
			assert.Equal(t, tt.want, node.Phase)
			if tt.want != wfv1.NodeError {
				assert.Empty(t, node.Message)
				return
			}
			assert.Equal(t, `pool "`+node.PooledTask.PoolName+`" can't be built: `+reason, node.Message)
			assert.True(t, *node.TaskResultSynced)
			assert.False(t, hasTaskSet(fake, wf.Namespace, node.ID), "acked")
		})
	}
}
