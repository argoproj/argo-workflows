package dispatch

import (
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// protocolTests run against every implementation, so the fake behaves as
// the Kubernetes implementation does.
var protocolTests = []struct {
	name string
	test func(t *testing.T, b *backend)
}{
	{"EnqueueCreatesTheTaskSet", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		task.Pool.InstanceID = "my-instance"
		task.OnExit = true
		enqueued, err := b.dispatcher.Enqueue(logging.TestContext(t.Context()), task)
		require.NoError(t, err)
		assert.False(t, enqueued.Adopted)

		ts, ok := b.taskSet(t, task.Key)
		require.True(t, ok)
		assert.Equal(t, "node-1-task", ts.Name)
		assert.NotEmpty(t, ts.UID)
		assert.False(t, ts.CreationTimestamp.IsZero())
		assert.Equal(t, ts.UID, enqueued.TaskSet.UID)
		assert.Equal(t, map[string]string{
			common.LabelKeyWorkflow:             "my-wf",
			common.LabelKeyPoolName:             "my-pool",
			common.LabelKeyPoolHash:             "hash-1",
			common.LabelKeyControllerInstanceID: "my-instance",
			common.LabelKeyOnExit:               "true",
		}, ts.Labels)
		assert.Equal(t, map[string]string{common.AnnotationKeyPoolControllerVersion: "v4"}, ts.Annotations)
		assert.Equal(t, []metav1.OwnerReference{{
			APIVersion: workflow.APIVersion, Kind: workflow.WorkflowKind, Name: "my-wf", UID: "wf-uid",
		}}, ts.OwnerReferences)
		assert.Equal(t, map[string]wfv1.Template{"node-1": task.Template}, ts.Spec.Tasks)
		assert.Equal(t, &task.Pool, ts.Spec.Pool)
		assert.Equal(t, &task.Dispatch, ts.Spec.Dispatch)
	}},
	{"EnqueueWithoutInstanceIDOrExitHandler", func(t *testing.T, b *backend) {
		ts := b.enqueue(t, testTask("node-1"))
		assert.NotContains(t, ts.Labels, common.LabelKeyControllerInstanceID)
		assert.NotContains(t, ts.Labels, common.LabelKeyOnExit)
	}},
	{"EnqueueAdoptsAnExistingTaskSet", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		task := testTask("node-1")
		first, err := b.dispatcher.Enqueue(ctx, task)
		require.NoError(t, err)
		// A changed template must not replace the TaskSet.
		task.Template.Name = "changed"
		second, err := b.dispatcher.Enqueue(ctx, task)
		require.NoError(t, err)
		assert.True(t, second.Adopted)
		assert.Equal(t, first.TaskSet.UID, second.TaskSet.UID)
		ts, _ := b.taskSet(t, task.Key)
		assert.Equal(t, first.TaskSet.UID, ts.UID)
		assert.Equal(t, "main", ts.Spec.Tasks["node-1"].Name)
	}},
	// A TaskSet of the workflow's that was deleted, as when the controller
	// replaced it, but that a finalizer holds, is never claimed, so it isn't
	// adopted either.
	{"EnqueueWaitsForADeletedTaskSet", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		deleting := newTaskSet(task)
		deleting.DeletionTimestamp = new(metav1.Now())
		deleting.Finalizers = []string{"example.com/hold"}
		stored, err := b.store.createTaskSet(t.Context(), deleting)
		require.NoError(t, err)
		_, err = b.dispatcher.Enqueue(logging.TestContext(t.Context()), task)
		require.ErrorIs(t, err, ErrStaleTaskSet)
		ts, ok := b.taskSet(t, task.Key)
		require.True(t, ok)
		assert.Equal(t, stored.UID, ts.UID)
	}},
	// One workflow at a time holds a name, so a pooled TaskSet another
	// workflow UID owns, or none does, is left by an earlier workflow of the
	// same name and is replaced rather than adopted.
	{"EnqueueReplacesATaskSetLeftByAnEarlierWorkflow", func(t *testing.T, b *backend) {
		for _, owners := range [][]metav1.OwnerReference{
			{{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowKind, Name: "my-wf", UID: "old-wf-uid"}},
			nil, // after an orphan deletion
		} {
			task := testTask("node-1")
			old := newTaskSet(task)
			old.OwnerReferences = owners
			stale, err := b.store.createTaskSet(t.Context(), old)
			require.NoError(t, err)

			enqueued, err := b.dispatcher.Enqueue(logging.TestContext(t.Context()), task)
			require.NoError(t, err)
			assert.False(t, enqueued.Adopted)
			assert.NotEqual(t, stale.UID, enqueued.TaskSet.UID)
			ts, ok := b.taskSet(t, task.Key)
			require.True(t, ok)
			assert.Equal(t, enqueued.TaskSet.UID, ts.UID)
			assert.True(t, OwnedByWorkflow(ts, task.WorkflowUID))
			b.deleteTaskSet(t, task.Key)
		}
	}},
	{"EnqueueWaitsForATaskSetLeftByAnEarlierWorkflowToGo", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		old := newTaskSet(task)
		old.OwnerReferences = nil
		stale, err := b.store.createTaskSet(t.Context(), old)
		require.NoError(t, err)
		// The delete is accepted but the TaskSet stays, as while a finalizer
		// holds it: the create that follows finds it still there.
		held := ErrStale
		if b.clientset != nil {
			held = apierr.NewConflict(wfv1.SchemeGroupVersion.WithResource(ResourceTaskSets).GroupResource(), "node-1-task", errors.New("held"))
		}
		b.setFault(func(op Op) error {
			if op.Verb == "delete" && op.Resource == ResourceTaskSets {
				return held
			}
			return nil
		})
		_, err = b.dispatcher.Enqueue(logging.TestContext(t.Context()), task)
		require.ErrorIs(t, err, ErrStaleTaskSet)
		b.setFault(nil)
		ts, ok := b.taskSet(t, task.Key)
		require.True(t, ok)
		assert.Equal(t, stale.UID, ts.UID)
	}},
	{"EnqueueReportsDroppedFields", func(t *testing.T, b *backend) {
		// As an old CRD prunes them.
		task := testTask("node-1")
		pruned := newTaskSet(task)
		pruned.Spec.Pool = nil
		created, err := b.store.createTaskSet(t.Context(), pruned)
		require.NoError(t, err)
		enqueued, err := b.dispatcher.Enqueue(logging.TestContext(t.Context()), task)
		require.ErrorIs(t, err, ErrFieldsDropped)
		require.NotNil(t, enqueued.TaskSet, "the TaskSet is returned so it can be cleaned up")
		assert.Equal(t, created.UID, enqueued.TaskSet.UID)
	}},
	{"EnqueueRefusesATaskSetThatIsNotPooled", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		agent := newTaskSet(task)
		agent.Labels = nil
		_, err := b.store.createTaskSet(t.Context(), agent)
		require.NoError(t, err)
		enqueued, err := b.dispatcher.Enqueue(logging.TestContext(t.Context()), task)
		require.ErrorIs(t, err, ErrNameTaken)
		assert.Nil(t, enqueued.TaskSet)
	}},
	{"ClaimCreatesTheResult", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		task.Pool.InstanceID = "my-instance"
		ts := b.enqueue(t, task)
		h := b.claim(t, testWorker("worker-a"), task.Key)

		r, ok := b.result(t, task.Key)
		require.True(t, ok)
		assert.Equal(t, "node-1", r.Name)
		assert.Equal(t, map[string]string{
			common.LabelKeyWorkflow:               "my-wf",
			common.LabelKeyControllerInstanceID:   "my-instance",
			common.LabelKeyReportOutputsCompleted: "false",
			common.LabelKeyPoolName:               "my-pool",
			common.LabelKeyPoolHash:               "hash-1",
			common.LabelKeyTaskSetUID:             string(ts.UID),
			common.LabelKeyWorkerPod:              "worker-a",
			common.LabelKeyWorkerPodUID:           "worker-a-uid",
		}, r.Labels)
		assert.Equal(t, []metav1.OwnerReference{
			{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowKind, Name: "my-wf", UID: "wf-uid"},
			{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowTaskSetKind, Name: "node-1-task", UID: ts.UID},
		}, r.OwnerReferences)

		claim := h.Claim()
		assert.Equal(t, r.UID, claim.UID)
		assert.Equal(t, ts.UID, claim.TaskSetUID)
		assert.Equal(t, types.UID("worker-a-uid"), claim.WorkerPodUID)
		assert.Equal(t, "worker-a", claim.WorkerPodName)
		assert.Equal(t, r.CreationTimestamp, claim.CreatedAt)
		assert.False(t, claim.CreatedAt.IsZero())
		assert.Equal(t, ts.UID, h.TaskSet().UID)
		assert.Equal(t, task.Template, h.TaskSet().Spec.Tasks["node-1"])
	}},
	// Once the workflow is deleted, garbage collection deletes, in any order,
	// each object none of whose owners exists. Collected claims first, a
	// claimed TaskSet must never be left without its claim, or a worker would
	// claim it and run its task again.
	{"DeletingTheWorkflowNeverLeavesAClaimedTaskSetUnclaimed", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		b.claim(t, testWorker("worker-a"), task.Key)
		// The workflow is gone, so only a TaskSet still there owns anything.
		collectable := func(obj metav1.Object) bool {
			return !slices.ContainsFunc(obj.GetOwnerReferences(), func(owner metav1.OwnerReference) bool {
				ts, ok := b.taskSet(t, task.Key)
				return owner.Kind == workflow.WorkflowTaskSetKind && ok && ts.UID == owner.UID
			})
		}
		for {
			r, claimed := b.result(t, task.Key)
			ts, queued := b.taskSet(t, task.Key)
			if queued {
				require.True(t, claimed, "the TaskSet is left without its claim")
			}
			switch {
			case claimed && collectable(r):
				b.deleteResult(t, task.Key)
			case queued && collectable(ts):
				b.deleteTaskSet(t, task.Key)
			case !claimed && !queued:
				return
			default:
				t.Fatal("nothing can be collected")
			}
		}
	}},
	{"ClaimSetsNoInstanceIDWhenTheTaskSetHasNone", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		b.claim(t, testWorker("worker-a"), task.Key)
		r, _ := b.result(t, task.Key)
		assert.NotContains(t, r.Labels, common.LabelKeyControllerInstanceID)
	}},
	{"OnlyOneConsumerWinsAClaim", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		task := testTask("node-1")
		b.enqueue(t, task)
		candidate := b.candidate(t, task.Key)
		const workers = 8
		errs := make([]error, workers)
		var wg sync.WaitGroup
		for i := range workers {
			wg.Go(func() {
				_, errs[i] = b.consumer(testWorker("worker-"+string(rune('a'+i)))).Claim(ctx, candidate)
			})
		}
		wg.Wait()
		won := 0
		for _, err := range errs {
			if err == nil {
				won++
			} else {
				require.ErrorIs(t, err, ErrClaimLost)
			}
		}
		assert.Equal(t, 1, won)
	}},
	{"ClaimWithdrawsWhenTheTaskSetIsGone", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		candidate := b.candidate(t, task.Key)
		b.deleteTaskSet(t, task.Key)
		_, err := b.consumer(testWorker("worker-a")).Claim(logging.TestContext(t.Context()), candidate)
		require.ErrorIs(t, err, ErrTaskGone)
		_, ok := b.result(t, task.Key)
		assert.False(t, ok, "the claim was withdrawn")
	}},
	// A workflow counts a TaskSet as gone once its deletion is accepted, so a
	// TaskSet a finalizer still holds must not be claimed, or its task could
	// run after its workflow completes.
	{"ClaimWithdrawsWhenTheTaskSetIsBeingDeleted", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		deleting := newTaskSet(task)
		deleting.DeletionTimestamp = new(metav1.Now())
		deleting.Finalizers = []string{"example.com/hold"}
		stored, err := b.store.createTaskSet(t.Context(), deleting)
		require.NoError(t, err)
		live := stored.DeepCopy()
		live.DeletionTimestamp = nil
		candidate, ok := CandidateFromTaskSet(live)
		require.True(t, ok)
		_, err = b.consumer(testWorker("worker-a")).Claim(logging.TestContext(t.Context()), candidate)
		require.ErrorIs(t, err, ErrTaskGone)
		_, ok = b.result(t, task.Key)
		assert.False(t, ok, "the claim was withdrawn")
	}},
	{"ClaimWithdrawsWhenTheTaskSetIsReplaced", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		candidate := b.candidate(t, task.Key)
		b.deleteTaskSet(t, task.Key)
		replacement := b.enqueue(t, task)
		_, err := b.consumer(testWorker("worker-a")).Claim(logging.TestContext(t.Context()), candidate)
		require.ErrorIs(t, err, ErrTaskGone)
		_, ok := b.result(t, task.Key)
		assert.False(t, ok, "the claim was withdrawn")
		ts, ok := b.taskSet(t, task.Key)
		require.True(t, ok)
		assert.Equal(t, replacement.UID, ts.UID)
	}},
	{"ClaimWithdrawsWhenTheTaskSetCantBeRead", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		candidate := b.candidate(t, task.Key)
		b.setFault(failing("get", ResourceTaskSets, task.TaskSetName()))
		_, err := b.consumer(testWorker("worker-a")).Claim(logging.TestContext(t.Context()), candidate)
		b.setFault(nil)
		require.ErrorIs(t, err, errTransient)
		require.NotErrorIs(t, err, ErrTaskGone)
		_, ok := b.result(t, task.Key)
		assert.False(t, ok, "the claim was withdrawn")
		_, ok = b.taskSet(t, task.Key)
		assert.True(t, ok)
	}},
	{"ClaimReportsAFailedWithdrawal", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		candidate := b.candidate(t, task.Key)
		b.deleteTaskSet(t, task.Key)
		b.setFault(failing("delete", ResourceResults, task.ResultName()))
		_, err := b.consumer(testWorker("worker-a")).Claim(logging.TestContext(t.Context()), candidate)
		b.setFault(nil)
		require.ErrorIs(t, err, ErrTaskGone)
		require.ErrorIs(t, err, errTransient)
		_, ok := b.result(t, task.Key)
		assert.True(t, ok, "the claim remains")
	}},
	// A create whose response is lost has still made the claim, and nothing
	// else would run the task: the TaskSet has a result, so it is no
	// worker's candidate.
	{"ClaimKeepsTheClaimAFailedCreateMade", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		candidate := b.candidate(t, task.Key)
		w := testWorker("worker-a")
		b.setFault(createsThenFails(t, b, (&consumer{worker: w}).newResult(candidate)))
		h, err := b.consumer(w).Claim(logging.TestContext(t.Context()), candidate)
		b.setFault(nil)
		require.NoError(t, err)
		r, ok := b.result(t, task.Key)
		require.True(t, ok)
		assert.Equal(t, r.UID, h.Claim().UID)
		assert.Equal(t, w.PodUID, h.Claim().WorkerPodUID)
	}},
	{"ClaimAfterAFailedCreateLeavesAnotherWorkersClaim", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		candidate := b.candidate(t, task.Key)
		other := (&consumer{worker: testWorker("worker-b")}).newResult(candidate)
		b.setFault(createsThenFails(t, b, other))
		_, err := b.consumer(testWorker("worker-a")).Claim(logging.TestContext(t.Context()), candidate)
		b.setFault(nil)
		require.ErrorIs(t, err, ErrClaimLost)
		r, ok := b.result(t, task.Key)
		require.True(t, ok)
		assert.Equal(t, "worker-b-uid", r.Labels[common.LabelKeyWorkerPodUID])
	}},
	{"ClaimAfterAFailedCreateThatMadeNothingFails", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		b.setFault(failing("create", ResourceResults, task.ResultName()))
		_, err := b.consumer(testWorker("worker-a")).Claim(logging.TestContext(t.Context()), b.candidate(t, task.Key))
		b.setFault(nil)
		require.ErrorIs(t, err, errTransient)
		require.NotErrorIs(t, err, ErrClaimLost)
		_, ok := b.result(t, task.Key)
		assert.False(t, ok)
	}},
	// A stale candidate can offer a worker a task it already holds; claiming
	// it again must not hand out a second handle that would run it twice.
	{"ClaimTwiceBySameWorkerIsLost", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		task := testTask("node-1")
		b.enqueue(t, task)
		candidate := b.candidate(t, task.Key)
		c := b.consumer(testWorker("worker-a"))
		h, err := c.Claim(ctx, candidate)
		require.NoError(t, err)
		_, err = c.Claim(ctx, candidate)
		require.ErrorIs(t, err, ErrClaimLost)
		r, ok := b.result(t, task.Key)
		require.True(t, ok, "the first claim remains")
		assert.Equal(t, h.Claim().UID, r.UID)
	}},
	{"ClaimRefusesAnotherPool", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		w := testWorker("worker-a")
		w.PoolHash = "hash-2"
		_, err := b.consumer(w).Claim(logging.TestContext(t.Context()), b.candidate(t, task.Key))
		require.ErrorIs(t, err, ErrWrongPool)
		_, ok := b.result(t, task.Key)
		assert.False(t, ok)
	}},
	{"Complete", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		task := testTask("node-1")
		b.enqueue(t, task)
		h := b.claim(t, testWorker("worker-a"), task.Key)
		require.NoError(t, h.Progress(ctx, "1/2"))
		r, _ := b.result(t, task.Key)
		assert.Equal(t, wfv1.Progress("1/2"), r.Progress)
		assert.Equal(t, "false", r.Labels[common.LabelKeyReportOutputsCompleted])

		result := "hello"
		require.NoError(t, h.Complete(ctx, wfv1.NodeResult{
			Phase:    wfv1.NodeSucceeded,
			Message:  "done",
			Outputs:  &wfv1.Outputs{Result: &result},
			Progress: "2/2",
		}))
		r, _ = b.result(t, task.Key)
		assert.Equal(t, h.Claim().UID, r.UID)
		assert.Equal(t, wfv1.NodeSucceeded, r.Phase)
		assert.Equal(t, "done", r.Message)
		assert.Equal(t, &wfv1.Outputs{Result: &result}, r.Outputs)
		assert.Equal(t, wfv1.Progress("2/2"), r.Progress)
		assert.Equal(t, "true", r.Labels[common.LabelKeyReportOutputsCompleted])
		claim, ok := ClaimFromResult(r)
		require.True(t, ok)
		assert.True(t, claim.Completed)
	}},
	{"CompleteAfterTheResultWasDeletedReportsTheClaimLost", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		task := testTask("node-1")
		b.enqueue(t, task)
		h := b.claim(t, testWorker("worker-a"), task.Key)
		b.deleteResult(t, task.Key)
		err := h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded})
		require.ErrorIs(t, err, ErrClaimLost)
		require.ErrorIs(t, err, ErrNotFound)
		_, ok := b.result(t, task.Key)
		assert.False(t, ok, "the result is not recreated")
		assertStopped(t, h, StopLost)
	}},
	{"ProgressAfterTheResultWasDeletedReportsTheClaimLost", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		h := b.claim(t, testWorker("worker-a"), task.Key)
		b.deleteResult(t, task.Key)
		err := h.Progress(logging.TestContext(t.Context()), "1/2")
		require.ErrorIs(t, err, ErrClaimLost)
		_, ok := b.result(t, task.Key)
		assert.False(t, ok)
	}},
	{"CompleteOnAnotherWorkersResultFailsAndLeavesItUntouched", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		stale := b.claim(t, testWorker("worker-a"), task.Key)
		b.deleteResult(t, task.Key)
		b.claim(t, testWorker("worker-b"), task.Key)
		before, _ := b.result(t, task.Key)

		err := stale.Complete(logging.TestContext(t.Context()), wfv1.NodeResult{Phase: wfv1.NodeSucceeded})
		require.ErrorIs(t, err, ErrClaimLost)
		require.ErrorIs(t, err, ErrStale)
		after, _ := b.result(t, task.Key)
		assert.Equal(t, before, after)
	}},
	// The same worker pod's earlier claim has its pod UID but not the
	// result's UID.
	{"CompleteOnTheSameWorkersNewerClaimFailsAndLeavesItUntouched", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		task := testTask("node-1")
		b.enqueue(t, task)
		w := testWorker("worker-a")
		stale := b.claim(t, w, task.Key)
		require.NoError(t, stale.Release(ctx))
		b.claim(t, w, task.Key)
		before, _ := b.result(t, task.Key)

		err := stale.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded})
		require.ErrorIs(t, err, ErrClaimLost)
		require.ErrorIs(t, err, ErrStale)
		after, _ := b.result(t, task.Key)
		assert.Equal(t, before, after)
	}},
	{"ReleaseDeletesOnlyTheCallersResult", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		task := testTask("node-1")
		b.enqueue(t, task)
		h := b.claim(t, testWorker("worker-a"), task.Key)
		require.NoError(t, h.Release(ctx))
		_, ok := b.result(t, task.Key)
		assert.False(t, ok)
		_, ok = b.taskSet(t, task.Key)
		assert.True(t, ok, "a released task can be claimed again")
		require.NoError(t, h.Release(ctx), "releasing twice is fine")

		other := b.claim(t, testWorker("worker-b"), task.Key)
		err := h.Release(ctx)
		require.ErrorIs(t, err, ErrClaimLost)
		require.ErrorIs(t, err, ErrStale)
		r, ok := b.result(t, task.Key)
		require.True(t, ok)
		assert.Equal(t, other.Claim().UID, r.UID)
	}},
	{"FailOwnFinalisesOnlyTheCallersIncompleteResults", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		a, other := testWorker("worker-a"), testWorker("worker-b")
		for _, node := range []string{"node-1", "node-2", "node-3", "node-4"} {
			b.enqueue(t, testTask(node))
		}
		key := func(node string) Key { return Key{Namespace: testNamespace, NodeID: node} }
		b.claim(t, a, key("node-1"))
		b.claim(t, a, key("node-2"))
		done := b.claim(t, a, key("node-3"))
		require.NoError(t, done.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded}))
		b.claim(t, other, key("node-4"))
		doneBefore, _ := b.result(t, key("node-3"))
		otherBefore, _ := b.result(t, key("node-4"))

		failed, err := b.consumer(a).FailOwn(ctx, "worker restarted")
		require.NoError(t, err)
		assert.ElementsMatch(t, []Key{key("node-1"), key("node-2")}, failed)
		for _, node := range []string{"node-1", "node-2"} {
			r, _ := b.result(t, key(node))
			assert.Equal(t, wfv1.NodeError, r.Phase)
			assert.Equal(t, "worker restarted", r.Message)
			assert.Equal(t, "true", r.Labels[common.LabelKeyReportOutputsCompleted])
		}
		doneAfter, _ := b.result(t, key("node-3"))
		assert.Equal(t, doneBefore, doneAfter)
		otherAfter, _ := b.result(t, key("node-4"))
		assert.Equal(t, otherBefore, otherAfter)
	}},
	{"FailOwnFinalisesTheRestWhenOneWriteFails", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		a := testWorker("worker-a")
		key := func(node string) Key { return Key{Namespace: testNamespace, NodeID: node} }
		for _, node := range []string{"node-1", "node-2"} {
			b.enqueue(t, testTask(node))
			b.claim(t, a, key(node))
		}
		b.setFault(failing("patch", ResourceResults, "node-1"))
		failed, err := b.consumer(a).FailOwn(ctx, "worker restarted")
		b.setFault(nil)
		require.ErrorIs(t, err, errTransient)
		assert.Equal(t, []Key{key("node-2")}, failed)
		r, _ := b.result(t, key("node-1"))
		assert.Equal(t, "false", r.Labels[common.LabelKeyReportOutputsCompleted])
		r, _ = b.result(t, key("node-2"))
		assert.Equal(t, wfv1.NodeError, r.Phase)
		assert.Equal(t, "true", r.Labels[common.LabelKeyReportOutputsCompleted])
	}},
	{"FailOwnLeavesAResultReplacedSinceTheList", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		a := testWorker("worker-a")
		key := func(node string) Key { return Key{Namespace: testNamespace, NodeID: node} }
		for _, node := range []string{"node-1", "node-2"} {
			b.enqueue(t, testTask(node))
			b.claim(t, a, key(node))
		}
		// Between the list and the write, another worker's claim replaces it.
		replacement := (&consumer{worker: testWorker("worker-b")}).newResult(b.candidate(t, key("node-1")))
		var once sync.Once
		b.setFault(func(op Op) error {
			if op.Verb == "patch" && op.Name == "node-1" {
				once.Do(func() { assert.NoError(t, b.replaceResult(replacement)) })
			}
			return nil
		})
		failed, err := b.consumer(a).FailOwn(ctx, "worker restarted")
		b.setFault(nil)
		require.NoError(t, err)
		assert.Equal(t, []Key{key("node-2")}, failed)
		r, _ := b.result(t, key("node-1"))
		assert.Equal(t, "worker-b-uid", r.Labels[common.LabelKeyWorkerPodUID])
		assert.Equal(t, "false", r.Labels[common.LabelKeyReportOutputsCompleted])
		assert.Empty(t, r.Phase)
	}},
	{"AckDeletesTheTaskSet", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		task := testTask("node-1")
		ts := b.enqueue(t, task)
		require.NoError(t, b.dispatcher.Ack(ctx, task.Key, ts.UID))
		_, ok := b.taskSet(t, task.Key)
		assert.False(t, ok)
		require.NoError(t, b.dispatcher.Ack(ctx, task.Key, ts.UID), "a TaskSet already gone is fine")
	}},
	{"AckWithAStaleUIDFailsAndLeavesTheTaskSetUntouched", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		stale := b.enqueue(t, task)
		b.deleteTaskSet(t, task.Key)
		fresh := b.enqueue(t, task)
		err := b.dispatcher.Ack(logging.TestContext(t.Context()), task.Key, stale.UID)
		require.ErrorIs(t, err, ErrStale)
		ts, ok := b.taskSet(t, task.Key)
		require.True(t, ok)
		assert.Equal(t, fresh, ts)
	}},
	{"DiscardDeletesTheResult", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		task := testTask("node-1")
		b.enqueue(t, task)
		h := b.claim(t, testWorker("pod-a"), task.Key)
		require.NoError(t, b.dispatcher.Discard(ctx, task.Key, h.Claim().UID))
		_, ok := b.result(t, task.Key)
		assert.False(t, ok)
		require.NoError(t, b.dispatcher.Discard(ctx, task.Key, h.Claim().UID), "a result already gone is fine")
	}},
	{"DiscardLeavesAReplacedResultUntouched", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		b.enqueue(t, task)
		stale := b.claim(t, testWorker("pod-a"), task.Key)
		require.NoError(t, stale.Release(logging.TestContext(t.Context())))
		b.claim(t, testWorker("pod-b"), task.Key)
		fresh, _ := b.result(t, task.Key)
		require.NoError(t, b.dispatcher.Discard(logging.TestContext(t.Context()), task.Key, stale.Claim().UID))
		r, ok := b.result(t, task.Key)
		require.True(t, ok)
		assert.Equal(t, fresh, r)
	}},
	{"ResultReadsTheClaim", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		task := testTask("node-1")
		b.enqueue(t, task)
		_, err := b.dispatcher.Result(ctx, task.Key)
		require.ErrorIs(t, err, ErrNotFound)
		h := b.claim(t, testWorker("pod-a"), task.Key)
		require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded}))
		result, err := b.dispatcher.Result(ctx, task.Key)
		require.NoError(t, err)
		stored, _ := b.result(t, task.Key)
		assert.Equal(t, stored, result)
		claim, ok := ClaimFromResult(result)
		require.True(t, ok)
		assert.True(t, claim.Completed)
		assert.Equal(t, h.Claim().UID, claim.UID)
	}},
	{"DropAnUnclaimedTask", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		ts := b.enqueue(t, task)
		claim, err := b.dispatcher.Drop(logging.TestContext(t.Context()), task.Key, ts.UID)
		require.NoError(t, err)
		assert.Nil(t, claim)
		_, ok := b.taskSet(t, task.Key)
		assert.False(t, ok)
	}},
	{"DropAClaimedTaskFindsTheClaim", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		ts := b.enqueue(t, task)
		h := b.claim(t, testWorker("worker-a"), task.Key)
		stored, ok := b.result(t, task.Key)
		require.True(t, ok)
		found, err := b.dispatcher.Drop(logging.TestContext(t.Context()), task.Key, ts.UID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, h.Claim(), found.Claim)
		assert.Equal(t, stored, found.Result, "the result as read live")
		_, ok = b.taskSet(t, task.Key)
		assert.False(t, ok)
		_, ok = b.result(t, task.Key)
		assert.True(t, ok, "the claim is left for the interrupt")
	}},
	{"DropIgnoresAResultForAnotherTaskSet", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		stale := b.enqueue(t, task)
		b.deleteTaskSet(t, task.Key)
		fresh := b.enqueue(t, task)
		b.claim(t, testWorker("worker-a"), task.Key)
		claim, err := b.dispatcher.Drop(logging.TestContext(t.Context()), task.Key, stale.UID)
		require.NoError(t, err)
		assert.Nil(t, claim)
		ts, ok := b.taskSet(t, task.Key)
		require.True(t, ok, "a TaskSet with another UID is left alone")
		assert.Equal(t, fresh, ts)
	}},
	// A claim racing a drop is found by the drop or never starts.
	{"ClaimAfterDropNeverStarts", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		ts := b.enqueue(t, task)
		candidate := b.candidate(t, task.Key)
		claim, err := b.dispatcher.Drop(logging.TestContext(t.Context()), task.Key, ts.UID)
		require.NoError(t, err)
		require.Nil(t, claim)
		_, err = b.consumer(testWorker("worker-a")).Claim(logging.TestContext(t.Context()), candidate)
		require.ErrorIs(t, err, ErrTaskGone)
		_, ok := b.result(t, task.Key)
		assert.False(t, ok)
	}},
	{"DropFailsWhenTheResultCantBeRead", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		ts := b.enqueue(t, task)
		b.setFault(failing("get", ResourceResults, task.ResultName()))
		claim, err := b.dispatcher.Drop(logging.TestContext(t.Context()), task.Key, ts.UID)
		b.setFault(nil)
		require.ErrorIs(t, err, errTransient, "an unread result is never taken as no claim")
		assert.Nil(t, claim)
		_, ok := b.taskSet(t, task.Key)
		assert.False(t, ok, "the TaskSet is deleted first")
	}},
	{"InterruptLabelsTheClaim", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		task := testTask("node-1")
		ts := b.enqueue(t, task)
		h := b.claim(t, testWorker("worker-a"), task.Key)
		require.NoError(t, b.dispatcher.Interrupt(ctx, task.Key, ts.UID, h.Claim().UID))
		_, ok := b.taskSet(t, task.Key)
		assert.False(t, ok)
		r, _ := b.result(t, task.Key)
		assert.Equal(t, "true", r.Labels[common.LabelKeyInterrupt])
		claim, _ := ClaimFromResult(r)
		assert.True(t, claim.Interrupted)
		require.NoError(t, b.dispatcher.Interrupt(ctx, task.Key, ts.UID, h.Claim().UID), "interrupting twice is fine")

		// The worker confirms by completing with no outputs.
		require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeFailed}))
		// After the interrupt, the task can't be claimed again.
		_, err := b.consumer(testWorker("worker-b")).Claim(ctx, Candidate{
			Key: task.Key, TaskSetUID: ts.UID, WorkflowName: "my-wf", WorkflowUID: "wf-uid",
			PoolName: "my-pool", PoolHash: "hash-1",
		})
		require.ErrorIs(t, err, ErrClaimLost)
	}},
	{"InterruptOfADeletedClaimReportsItLost", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		ts := b.enqueue(t, task)
		h := b.claim(t, testWorker("worker-a"), task.Key)
		b.deleteResult(t, task.Key)
		err := b.dispatcher.Interrupt(logging.TestContext(t.Context()), task.Key, ts.UID, h.Claim().UID)
		require.ErrorIs(t, err, ErrClaimLost)
		require.ErrorIs(t, err, ErrNotFound)
		_, ok := b.taskSet(t, task.Key)
		assert.False(t, ok, "the TaskSet is deleted first")
	}},
	{"InterruptWithAStaleClaimUIDLeavesTheResultUntouched", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		ts := b.enqueue(t, task)
		stale := b.claim(t, testWorker("worker-a"), task.Key)
		b.deleteResult(t, task.Key)
		b.claim(t, testWorker("worker-b"), task.Key)
		before, _ := b.result(t, task.Key)
		err := b.dispatcher.Interrupt(logging.TestContext(t.Context()), task.Key, ts.UID, stale.Claim().UID)
		require.ErrorIs(t, err, ErrClaimLost)
		require.ErrorIs(t, err, ErrStale)
		after, _ := b.result(t, task.Key)
		assert.Equal(t, before, after)
	}},
	{"InterruptWaitsForTheTaskSetDelete", func(t *testing.T, b *backend) {
		task := testTask("node-1")
		ts := b.enqueue(t, task)
		h := b.claim(t, testWorker("worker-a"), task.Key)
		b.setFault(func(op Op) error {
			if op.Verb == "delete" && op.Resource == ResourceTaskSets {
				return errTransient
			}
			return nil
		})
		err := b.dispatcher.Interrupt(logging.TestContext(t.Context()), task.Key, ts.UID, h.Claim().UID)
		b.setFault(nil)
		require.ErrorIs(t, err, errTransient)
		require.NotErrorIs(t, err, ErrClaimLost)
		r, _ := b.result(t, task.Key)
		assert.NotContains(t, r.Labels, common.LabelKeyInterrupt)
	}},
	{"Shutdown", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		key := func(node string) Key { return Key{Namespace: testNamespace, NodeID: node} }
		queued := b.enqueue(t, testTask("queued"))
		racing := b.enqueue(t, testTask("racing"))
		racingClaim := b.claim(t, testWorker("worker-a"), key("racing"))
		claimed := b.enqueue(t, testTask("claimed"))
		claimedClaim := b.claim(t, testWorker("worker-b"), key("claimed"))
		gone := b.enqueue(t, testTask("gone"))
		goneClaim := b.claim(t, testWorker("worker-c"), key("gone"))
		b.deleteResult(t, key("gone"))
		racingResult, _ := b.result(t, key("racing"))

		outcomes := b.dispatcher.Shutdown(ctx, []LedgerEntry{
			{Key: key("queued"), Ledger: ledger(queued.UID, "")},
			// Claimed since the ledger was written.
			{Key: key("racing"), Ledger: ledger(racing.UID, "")},
			{Key: key("claimed"), Ledger: ledger(claimed.UID, claimedClaim.Claim().UID)},
			{Key: key("gone"), Ledger: ledger(gone.UID, goneClaim.Claim().UID, "interrupt")},
		})
		assert.Equal(t, []ShutdownOutcome{
			{Key: key("queued"), Finished: true},
			{Key: key("racing"), Found: &FoundClaim{Claim: racingClaim.Claim(), Result: racingResult}},
			{Key: key("claimed")},
			{Key: key("gone"), Finished: true},
		}, outcomes)
		for _, node := range []string{"queued", "racing", "claimed", "gone"} {
			_, ok := b.taskSet(t, key(node))
			assert.False(t, ok, node)
		}
		for _, node := range []string{"racing", "claimed"} {
			r, _ := b.result(t, key(node))
			assert.Equal(t, "true", r.Labels[common.LabelKeyInterrupt], node)
		}
	}},
	{"ShutdownReportsWhatFailed", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		key := func(node string) Key { return Key{Namespace: testNamespace, NodeID: node} }
		dropFails := b.enqueue(t, testTask("drop-fails"))
		readFails := b.enqueue(t, testTask("read-fails"))
		racing := b.enqueue(t, testTask("racing"))
		racingClaim := b.claim(t, testWorker("worker-a"), key("racing"))
		claimed := b.enqueue(t, testTask("claimed"))
		claimedClaim := b.claim(t, testWorker("worker-b"), key("claimed"))
		racingResult, _ := b.result(t, key("racing"))
		b.setFault(faults(
			failing("delete", ResourceTaskSets, dropFails.Name),
			failing("get", ResourceResults, "read-fails"),
			failing("patch", ResourceResults, "racing"),
			failing("patch", ResourceResults, "claimed"),
		))
		outcomes := b.dispatcher.Shutdown(ctx, []LedgerEntry{
			{Key: key("drop-fails"), Ledger: ledger(dropFails.UID, "")},
			{Key: key("read-fails"), Ledger: ledger(readFails.UID, "")},
			// Claimed since the ledger was written.
			{Key: key("racing"), Ledger: ledger(racing.UID, "")},
			{Key: key("claimed"), Ledger: ledger(claimed.UID, claimedClaim.Claim().UID)},
		})
		b.setFault(nil)
		for i := range outcomes {
			require.ErrorIs(t, outcomes[i].Err, errTransient, outcomes[i].NodeID)
			outcomes[i].Err = nil
		}
		assert.Equal(t, []ShutdownOutcome{
			{Key: key("drop-fails")},
			{Key: key("read-fails")},
			{Key: key("racing"), Found: &FoundClaim{Claim: racingClaim.Claim(), Result: racingResult}},
			{Key: key("claimed")},
		}, outcomes, "none is finished")
		_, ok := b.taskSet(t, key("drop-fails"))
		assert.True(t, ok)
		for _, node := range []string{"racing", "claimed"} {
			r, _ := b.result(t, key(node))
			assert.NotContains(t, r.Labels, common.LabelKeyInterrupt, node)
		}
	}},
	{"CompletionDeletesCachedAndLedgerTaskSets", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		ref := func(ts *wfv1.WorkflowTaskSet) TaskSetRef {
			r, ok := TaskSetRefFromMeta(ts)
			require.True(t, ok)
			return r
		}
		both := b.enqueue(t, testTask("both"))
		cachedOnly := b.enqueue(t, testTask("cached-only"))
		ledgerOnly := b.enqueue(t, testTask("ledger-only"))
		staleCache := b.enqueue(t, testTask("stale-cache"))
		b.deleteTaskSet(t, Key{Namespace: testNamespace, NodeID: "stale-cache"})
		fresh := b.enqueue(t, testTask("stale-cache"))
		alreadyGone := TaskSetRef{NodeID: "already-gone", UID: "x"}

		gone, err := b.dispatcher.Completion(ctx, testNamespace,
			[]TaskSetRef{ref(both), ref(cachedOnly), ref(staleCache), alreadyGone},
			[]TaskSetRef{ref(both), ref(ledgerOnly), ref(fresh), {NodeID: "no-uid"}})
		require.NoError(t, err)
		assert.ElementsMatch(t, []TaskSetRef{ref(both), ref(cachedOnly), ref(staleCache), ref(fresh), alreadyGone, ref(ledgerOnly)}, gone)
		for _, node := range []string{"both", "cached-only", "ledger-only", "stale-cache"} {
			_, ok := b.taskSet(t, Key{Namespace: testNamespace, NodeID: node})
			assert.False(t, ok, node)
		}
	}},
	{"CompletionReportsWhatFailedToDelete", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		failing := b.enqueue(t, testTask("failing"))
		deleted := b.enqueue(t, testTask("deleted"))
		refs := []TaskSetRef{{NodeID: "failing", UID: failing.UID}, {NodeID: "deleted", UID: deleted.UID}}
		b.setFault(func(op Op) error {
			if op.Verb == "delete" && op.Name == failing.Name {
				return errTransient
			}
			return nil
		})
		gone, err := b.dispatcher.Completion(ctx, testNamespace, refs, nil)
		b.setFault(nil)
		require.ErrorIs(t, err, errTransient)
		assert.Equal(t, []TaskSetRef{refs[1]}, gone)
		_, ok := b.taskSet(t, Key{Namespace: testNamespace, NodeID: "failing"})
		assert.True(t, ok)

		gone, err = b.dispatcher.Completion(ctx, testNamespace, refs[:1], nil)
		require.NoError(t, err)
		assert.Equal(t, refs[:1], gone)
	}},
	// A delete finding another UID counts as done.
	{"CompletionLeavesATaskSetItDoesntKnow", func(t *testing.T, b *backend) {
		ctx := logging.TestContext(t.Context())
		key := Key{Namespace: testNamespace, NodeID: "node-1"}
		stale := b.enqueue(t, testTask("node-1"))
		b.deleteTaskSet(t, key)
		unknown := b.enqueue(t, testTask("node-1"))
		refs := []TaskSetRef{{NodeID: "node-1", UID: stale.UID}}
		gone, err := b.dispatcher.Completion(ctx, testNamespace, refs, nil)
		require.NoError(t, err)
		assert.Equal(t, refs, gone)
		got, _ := b.taskSet(t, key)
		assert.Equal(t, unknown, got)
	}},
}

func TestProtocol(t *testing.T) {
	for _, tt := range protocolTests {
		t.Run(tt.name, func(t *testing.T) {
			forEachBackend(t, tt.test)
		})
	}
}

func TestCandidateFromTaskSet(t *testing.T) {
	task := testTask("node-1")
	task.Pool.InstanceID = "my-instance"
	stored := func() *wfv1.WorkflowTaskSet {
		ts := newTaskSet(task)
		ts.UID = "ts-uid"
		ts.CreationTimestamp = metav1.Unix(1, 0)
		return ts
	}
	c, ok := CandidateFromTaskSet(stored())
	require.True(t, ok)
	assert.Equal(t, Candidate{
		Key:          task.Key,
		TaskSetUID:   "ts-uid",
		WorkflowName: "my-wf",
		WorkflowUID:  "wf-uid",
		InstanceID:   "my-instance",
		PoolName:     "my-pool",
		PoolHash:     "hash-1",
		CreatedAt:    metav1.Unix(1, 0),
	}, c)

	for _, tt := range []struct {
		name   string
		mutate func(ts *wfv1.WorkflowTaskSet)
	}{
		{"NotPooled", func(ts *wfv1.WorkflowTaskSet) { delete(ts.Labels, common.LabelKeyPoolHash) }},
		{"NotNamedForANode", func(ts *wfv1.WorkflowTaskSet) { ts.Name = "my-wf" }},
		{"NoWorkflowLabel", func(ts *wfv1.WorkflowTaskSet) { delete(ts.Labels, common.LabelKeyWorkflow) }},
		{"NoOwner", func(ts *wfv1.WorkflowTaskSet) { ts.OwnerReferences = nil }},
		{"OwnedByAnotherWorkflow", func(ts *wfv1.WorkflowTaskSet) { ts.OwnerReferences[0].Name = "other-wf" }},
		{"BeingDeleted", func(ts *wfv1.WorkflowTaskSet) { ts.DeletionTimestamp = new(metav1.Now()) }},
		{"OwnedByAnotherKind", func(ts *wfv1.WorkflowTaskSet) { ts.OwnerReferences[0].Kind = workflow.WorkflowTemplateKind }},
		{"OwnerWithoutUID", func(ts *wfv1.WorkflowTaskSet) { ts.OwnerReferences[0].UID = "" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ts := stored()
			tt.mutate(ts)
			_, ok := CandidateFromTaskSet(ts)
			assert.False(t, ok)
		})
	}
}

// assertStopped checks the claim was told to stop for the reason.
func assertStopped(t *testing.T, h Claimed, reason StopReason) {
	t.Helper()
	select {
	case <-h.Stopped():
		assert.Equal(t, reason, h.StopReason())
	default:
		t.Errorf("claim not stopped, want %s", reason)
	}
}

// ledger is a node's ledger for the TaskSet taskSetUID, enqueued, then
// claimed by claimUID if it is set, then taken through steps ("interrupt").
func ledger(taskSetUID, claimUID types.UID, steps ...string) wfv1.PooledTaskStatus {
	task := wfv1.PooledTaskStatus{TaskSetUID: taskSetUID}
	now := metav1.Now()
	task.Enqueue(now)
	if claimUID != "" {
		task.Claim(claimUID, now, "worker", "worker-uid", 0)
	}
	for _, step := range steps {
		if step == "interrupt" {
			task.Interrupt(now)
		}
	}
	return task
}

// errTransient is the error a fault injects.
var errTransient = errors.New("transient")

// failing is a fault failing calls of the verb on the named object.
func failing(verb, resource, name string) func(Op) error {
	return func(op Op) error {
		if op.Verb == verb && op.Resource == resource && op.Name == name {
			return errTransient
		}
		return nil
	}
}

// createsThenFails is a fault that, on the first create of a result, makes
// result in its place and then fails the create, as when the create's
// response is lost. Only the first: the Fake's replaceResult creates too.
func createsThenFails(t *testing.T, b *backend, result *wfv1.WorkflowTaskResult) func(Op) error {
	var fired atomic.Bool
	return func(op Op) error {
		if op.Verb == "create" && op.Resource == ResourceResults && fired.CompareAndSwap(false, true) {
			assert.NoError(t, b.replaceResult(result))
			return errTransient
		}
		return nil
	}
}

// faults is a fault failing a call as the first of fs that fails it.
func faults(fs ...func(Op) error) func(Op) error {
	return func(op Op) error {
		for _, f := range fs {
			if err := f(op); err != nil {
				return err
			}
		}
		return nil
	}
}
