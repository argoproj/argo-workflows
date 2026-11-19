package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kwait "k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// forgetResult deletes the result the controller's result informer holds, as
// the API server's watch would deliver its deletion.
func forgetResult(ctx context.Context, t *testing.T, controller *WorkflowController, namespace, name string) {
	t.Helper()
	require.NoError(t, controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(namespace).Delete(ctx, name, metav1.DeleteOptions{}))
	err := kwait.PollUntilContextTimeout(ctx, time.Millisecond, 10*time.Second, true, func(context.Context) (bool, error) {
		_, exists, err := controller.taskResultInformer.GetStore().GetByKey(namespace + "/" + name)
		return !exists, err
	})
	require.NoError(t, err)
}

// claimedCondition is the ledger's Claimed condition.
func claimedCondition(t *testing.T, ledger *wfv1.PooledTaskStatus) metav1.Condition {
	t.Helper()
	c := meta.FindStatusCondition(ledger.Conditions, wfv1.PooledTaskConditionClaimed)
	require.NotNil(t, c)
	return *c
}

// TestPooledClaimMakesNodeRunning checks that a claim makes the node
// Running, with the claimant in the ledger, its worker restart count as it
// was when the claim was first seen, and the claim's creation time as
// claimedAt; the node's host node and pod IP are the claimant's.
func TestPooledClaimMakesNodeRunning(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	id := mustNode(t, woc, pooledStep).ID
	assert.Equal(t, wfv1.NodePending, mustNode(t, woc, pooledStep).Phase)

	worker := observeWorkerPod(ctx, t, controller, wf.Namespace, "work-0123456789-abcde", "worker-pod-uid", 2)
	claimPooledBy(ctx, t, fake, wf.Namespace, id, worker)
	observeResult(ctx, t, controller, fake, wf.Namespace, id)
	result, ok := fake.Result(wf.Namespace, id)
	require.True(t, ok)

	woc = reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.Empty(t, node.Message)
	assert.False(t, *node.TaskResultSynced)
	ledger := node.PooledTask
	assert.Equal(t, wfv1.PooledTaskClaimed, ledger.State())
	assert.Equal(t, result.UID, ledger.ClaimUID)
	assert.Equal(t, "work-0123456789-abcde", ledger.WorkerPodName)
	assert.Equal(t, types.UID("worker-pod-uid"), ledger.WorkerPodUID)
	assert.Equal(t, int32(2), ledger.WorkerRestartCount)
	require.NotNil(t, ledger.ClaimedAt())
	assert.True(t, ledger.ClaimedAt().Equal(&result.CreationTimestamp), "claimedAt is on the API server's clock")
	assert.Equal(t, "work-0123456789-abcde", claimedCondition(t, ledger).Message)
	assert.Equal(t, workerNode, node.HostNodeName)
	assert.Equal(t, workerPodIP, node.PodIP)

	woc = reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, *node, *mustNode(t, woc, pooledStep), "reconciling the claim again changes nothing")
}

// TestPooledClaimWaitsForClaimantPod checks a claim seen before its
// claimant pod: an incomplete claim is left alone until the pod is cached,
// since its restart count can't be known before, while a completed one is
// applied at once.
func TestPooledClaimWaitsForClaimantPod(t *testing.T) {
	uncached := dispatch.Worker{Namespace: "default", PodName: "work-0123456789-fghij", PodUID: "uncached-uid"}

	t.Run("incomplete", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
		ctx, controller, fake := newPooledDispatchController(t, true, wf)
		woc := reconcilePooled(ctx, t, controller, wf)
		id := mustNode(t, woc, pooledStep).ID
		claimPooledBy(ctx, t, fake, wf.Namespace, id, uncached)
		observeResult(ctx, t, controller, fake, wf.Namespace, id)
		// A pod of that name, but not the claimant.
		observeWorkerPod(ctx, t, controller, wf.Namespace, uncached.PodName, "another-pod-uid", 0)

		woc = reconcilePooled(ctx, t, controller, wf)
		node := mustNode(t, woc, pooledStep)
		assert.Equal(t, wfv1.NodePending, node.Phase)
		assert.Equal(t, wfv1.PooledTaskEnqueued, node.PooledTask.State())
		assert.True(t, *node.TaskResultSynced, "nothing of the claim is applied")

		observeWorkerPod(ctx, t, controller, wf.Namespace, uncached.PodName, uncached.PodUID, 1)
		woc = reconcilePooled(ctx, t, controller, wf)
		node = mustNode(t, woc, pooledStep)
		assert.Equal(t, wfv1.NodeRunning, node.Phase)
		assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())
		assert.Equal(t, uncached.PodUID, node.PooledTask.WorkerPodUID)
		assert.Equal(t, int32(1), node.PooledTask.WorkerRestartCount)
		assert.False(t, *node.TaskResultSynced)
	})

	t.Run("completed", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
		ctx, controller, fake := newPooledDispatchController(t, true, wf)
		woc := reconcilePooled(ctx, t, controller, wf)
		id := mustNode(t, woc, pooledStep).ID
		h := claimPooledBy(ctx, t, fake, wf.Namespace, id, uncached)
		require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{ExitCode: new("0")}}))
		observeResult(ctx, t, controller, fake, wf.Namespace, id)

		woc = reconcilePooled(ctx, t, controller, wf)
		node := mustNode(t, woc, pooledStep)
		assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
		assert.True(t, *node.TaskResultSynced)
		assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())
		assert.Equal(t, uncached.PodName, node.PooledTask.WorkerPodName)
	})
}

// TestPooledRelease checks that a claim deleted by its worker while the ledger
// holds it is a release. The node stays Running with no task result in
// progress, and the ledger counts the release, names the worker that released
// it and no longer has a claimant.
func TestPooledRelease(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	id := mustNode(t, woc, pooledStep).ID
	observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
	worker := observeWorkerPod(ctx, t, controller, wf.Namespace, "work-0123456789-abcde", "worker-pod-uid", 2)
	h := claimPooledBy(ctx, t, fake, wf.Namespace, id, worker)
	observeResult(ctx, t, controller, fake, wf.Namespace, id)
	woc = reconcilePooled(ctx, t, controller, wf)
	require.Equal(t, wfv1.PooledTaskClaimed, mustNode(t, woc, pooledStep).PooledTask.State())

	require.NoError(t, h.Release(ctx))
	forgetResult(ctx, t, controller, wf.Namespace, id)
	woc = reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.True(t, *node.TaskResultSynced, "no task result is in progress")
	assert.False(t, node.Fulfilled())
	assert.False(t, woc.wf.Status.TaskResultsInProgress())
	ledger := node.PooledTask
	assert.Equal(t, wfv1.PooledTaskEnqueued, ledger.State())
	assert.True(t, ledger.Released())
	assert.Equal(t, "work-0123456789-abcde", claimedCondition(t, ledger).Message)
	assert.Equal(t, int32(1), ledger.Releases)
	assert.Empty(t, ledger.ClaimUID)
	assert.Empty(t, ledger.WorkerPodName)
	assert.Empty(t, ledger.WorkerPodUID)
	assert.Zero(t, ledger.WorkerRestartCount)
	assert.Nil(t, ledger.ClaimedAt())
	_, exists := fake.TaskSet(wf.Namespace, pooledTaskSetName(id))
	assert.True(t, exists, "the task can be claimed again")

	woc = reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, *node, *mustNode(t, woc, pooledStep), "a release is counted once")
}

// TestPooledReleaseWaitsForTaskSet checks that a claim gone from the result
// cache while the node's TaskSet isn't cached, as when it was created moments
// ago, isn't recorded as a release, and the workflow is requeued, since the
// TaskSet's arrival requeues nothing. Once the TaskSet is cached the release is
// recorded.
func TestPooledReleaseWaitsForTaskSet(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	id, h := runPooled(ctx, t, controller, fake, wf)
	unobserveTaskSet(ctx, t, controller, wf.Namespace, id)
	require.NoError(t, h.Release(ctx))
	forgetResult(ctx, t, controller, wf.Namespace, id)
	requeued := recordRequeues(controller)

	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
	assert.Equal(t, wfv1.PooledTaskClaimed, node.PooledTask.State())
	assert.False(t, node.PooledTask.Released())
	n, _ := requeued.take()
	assert.Equal(t, 1, n, "requeued until the TaskSet is cached")

	observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
	node = mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
	assert.True(t, node.PooledTask.Released())
	n, _ = requeued.take()
	assert.Zero(t, n)
}

// TestPooledReleaseThenReclaim checks that a release and a claim by another
// worker between two reconciles records the new claim, counts the release,
// and doesn't fail the node, whether or not the new claimant's pod is cached
// yet.
func TestPooledReleaseThenReclaim(t *testing.T) {
	for _, tt := range []struct {
		name       string
		podCached  bool
		wantStates []wfv1.PooledTaskState
	}{
		{name: "claimant cached", podCached: true, wantStates: []wfv1.PooledTaskState{wfv1.PooledTaskClaimed}},
		{name: "claimant not cached yet", wantStates: []wfv1.PooledTaskState{wfv1.PooledTaskEnqueued, wfv1.PooledTaskClaimed}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			woc := reconcilePooled(ctx, t, controller, wf)
			id := mustNode(t, woc, pooledStep).ID
			observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
			first := claimPooled(ctx, t, controller, fake, wf.Namespace, id)
			observeResult(ctx, t, controller, fake, wf.Namespace, id)
			woc = reconcilePooled(ctx, t, controller, wf)
			require.Equal(t, wfv1.PooledTaskClaimed, mustNode(t, woc, pooledStep).PooledTask.State())

			require.NoError(t, first.Release(ctx))
			other := dispatch.Worker{Namespace: wf.Namespace, PodName: "work-0123456789-fghij", PodUID: "other-pod-uid"}
			if tt.podCached {
				observeWorkerPod(ctx, t, controller, wf.Namespace, other.PodName, other.PodUID, 0)
			}
			second := claimPooledBy(ctx, t, fake, wf.Namespace, id, other)
			forgetResult(ctx, t, controller, wf.Namespace, id)
			observeResult(ctx, t, controller, fake, wf.Namespace, id)
			result, ok := fake.Result(wf.Namespace, id)
			require.True(t, ok)

			for i, want := range tt.wantStates {
				if want == wfv1.PooledTaskClaimed && !tt.podCached {
					observeWorkerPod(ctx, t, controller, wf.Namespace, other.PodName, other.PodUID, 0)
				}
				woc = reconcilePooled(ctx, t, controller, wf)
				node := mustNode(t, woc, pooledStep)
				assert.Equal(t, wfv1.NodeRunning, node.Phase, "reconcile %d", i)
				assert.Equal(t, want, node.PooledTask.State(), "reconcile %d", i)
				assert.Equal(t, int32(1), node.PooledTask.Releases, "reconcile %d", i)
			}
			node := mustNode(t, woc, pooledStep)
			assert.Equal(t, result.UID, node.PooledTask.ClaimUID)
			assert.Equal(t, other.PodName, node.PooledTask.WorkerPodName)
			assert.Equal(t, other.PodUID, node.PooledTask.WorkerPodUID)
			assert.False(t, *node.TaskResultSynced)

			require.NoError(t, second.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{ExitCode: new("0")}}))
			observeResult(ctx, t, controller, fake, wf.Namespace, id)
			woc = reconcilePooled(ctx, t, controller, wf)
			assert.Equal(t, wfv1.NodeSucceeded, mustNode(t, woc, pooledStep).Phase)
		})
	}
}

// TestPooledClaimGoneIsNotARelease checks that a claim missing from the
// result cache is a release only while its node hasn't ended, its ledger is
// claimed and its TaskSet is cached. An acked or deleted TaskSet, one not
// cached yet, an ended node or an interrupted ledger leave the node as it is,
// except that an interrupted task whose claim is gone has stopped, which
// finishes its node.
func TestPooledClaimGoneIsNotARelease(t *testing.T) {
	for _, tt := range []struct {
		name       string
		phase      wfv1.NodePhase
		state      wfv1.PooledTaskState
		taskSetUID types.UID
		// claimUID is the UID of the cached result, if any.
		claimUID types.UID
		legacy   bool
		released bool
		// stopped is set when the claim going finishes the interrupted node.
		stopped bool
	}{
		{name: "released", phase: wfv1.NodeRunning, state: wfv1.PooledTaskClaimed, taskSetUID: "task-set-uid", released: true},
		{name: "released, legacy completion status", phase: wfv1.NodeRunning, state: wfv1.PooledTaskClaimed, taskSetUID: "task-set-uid", legacy: true, released: true},
		{name: "claim cached", phase: wfv1.NodeRunning, state: wfv1.PooledTaskClaimed, taskSetUID: "task-set-uid", claimUID: "claim-uid"},
		{name: "TaskSet not cached", phase: wfv1.NodeRunning, state: wfv1.PooledTaskClaimed},
		{name: "another TaskSet cached", phase: wfv1.NodeRunning, state: wfv1.PooledTaskClaimed, taskSetUID: "another-task-set-uid"},
		{name: "node ended, TaskSet acked", phase: wfv1.NodeSucceeded, state: wfv1.PooledTaskClaimed},
		{name: "node ended, TaskSet not acked yet", phase: wfv1.NodeSucceeded, state: wfv1.PooledTaskClaimed, taskSetUID: "task-set-uid"},
		{name: "interrupted", phase: wfv1.NodeFailed, state: wfv1.PooledTaskInterrupted, taskSetUID: "task-set-uid", stopped: true},
		{name: "interrupted, claimed again", phase: wfv1.NodeFailed, state: wfv1.PooledTaskInterrupted, taskSetUID: "task-set-uid", claimUID: "another-claim-uid"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synced := tt.phase == wfv1.NodeSucceeded
			ctx, woc := newPooledNodesWoc(t, "main", wfv1.NodeStatus{
				Name:             pooledChild,
				Phase:            tt.phase,
				TaskResultSynced: new(synced || tt.legacy),
				PooledTask:       pooledTask(tt.state),
			})
			id := woc.wf.NodeID(pooledChild)
			if tt.legacy {
				woc.wf.Status.TaskResultsCompletionStatus = map[string]bool{id: synced}
			}
			if tt.taskSetUID != "" {
				require.NoError(t, woc.controller.pooledTaskSetInformer.GetIndexer().Add(&metav1.PartialObjectMetadata{
					TypeMeta: metav1.TypeMeta{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowTaskSetKind},
					ObjectMeta: metav1.ObjectMeta{
						Namespace: woc.wf.Namespace,
						Name:      pooledTaskSetName(id),
						UID:       tt.taskSetUID,
						Labels:    map[string]string{common.LabelKeyWorkflow: woc.wf.Name, common.LabelKeyPoolHash: "hash"},
					},
				}))
			}
			if tt.claimUID != "" {
				result := pooledClaim(woc, id, false)
				result.UID = tt.claimUID
				require.NoError(t, woc.controller.taskResultInformer.GetIndexer().Add(result))
			}
			before, err := woc.wf.Status.Nodes.Get(id)
			require.NoError(t, err)

			woc.taskResultReconciliation(ctx)
			node, err := woc.wf.Status.Nodes.Get(id)
			require.NoError(t, err)
			if tt.stopped {
				assert.True(t, node.Fulfilled())
				assert.False(t, node.FinishedAt.IsZero())
				node.TaskResultSynced, node.FinishedAt = before.TaskResultSynced, before.FinishedAt
				assert.Equal(t, before, node)
				return
			}
			if !tt.released {
				assert.Equal(t, before, node)
				assert.Equal(t, synced, !woc.wf.Status.IsTaskResultIncomplete(id))
				return
			}
			assert.Equal(t, wfv1.NodeRunning, node.Phase)
			assert.True(t, node.PooledTask.Released())
			assert.Equal(t, int32(1), node.PooledTask.Releases)
			assert.False(t, woc.wf.Status.IsTaskResultIncomplete(id))
			assert.False(t, woc.wf.Status.TaskResultsInProgress())
		})
	}
}
