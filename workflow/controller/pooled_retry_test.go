package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kwait "k8s.io/apimachinery/pkg/util/wait"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	wfutil "github.com/argoproj/argo-workflows/v4/workflow/util"
)

const pooledRetryWf = `
metadata:
  name: pooled-retry
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: work
  - name: work
    pool: {}
    container:
      image: my-image
`

// unobserveResult makes the controller's result informer drop the result,
// as the API server's watch would on its deletion.
func unobserveResult(ctx context.Context, t *testing.T, controller *WorkflowController, namespace, name string) {
	t.Helper()
	require.NoError(t, controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(namespace).Delete(ctx, name, metav1.DeleteOptions{}))
	require.NoError(t, kwait.PollUntilContextTimeout(ctx, time.Millisecond, 10*time.Second, true, func(context.Context) (bool, error) {
		_, exists, err := controller.taskResultInformer.GetStore().GetByKey(namespace + "/" + name)
		return !exists, err
	}))
}

// TestPooledRetryRunsTheNodeAgain checks that `argo retry` of a failed pooled
// node, which touches no TaskSet or result, runs it again: the next reconcile
// enqueues a fresh TaskSet, also when the failed attempt's claim was left
// behind, as when deleting the workflow's results failed, and when its
// TaskSet was left too, as by an archived workflow whose retry re-created it
// with a new UID before garbage collection got to them. The TaskSet is
// replaced, the claim discarded, and a worker claims the fresh TaskSet and
// runs the node.
func TestPooledRetryRunsTheNodeAgain(t *testing.T) {
	for _, tt := range []struct {
		name string
		// archived re-creates the workflow with a new UID, leaving its old
		// TaskSet as well as its claim.
		archived bool
	}{
		{name: "ClaimLeft"},
		{name: "TaskSetAndClaimLeftByTheArchivedWorkflow", archived: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledRetryWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			const step = "pooled-retry[0].a"

			woc := reconcilePooled(ctx, t, controller, wf)
			id := mustNode(t, woc, step).ID
			oldTaskSet, ok := fake.TaskSet(wf.Namespace, pooledTaskSetName(id))
			require.True(t, ok)
			worker := observeWorkerPod(ctx, t, controller, wf.Namespace, "work-0123456789-abcde", "worker-pod-uid", 0)
			h := claimPooledBy(ctx, t, fake, wf.Namespace, id, worker)
			require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeFailed, Message: "Error (exit code 1)"}))
			observeResult(ctx, t, controller, fake, wf.Namespace, id)
			for range 3 {
				woc = reconcilePooled(ctx, t, controller, wf)
			}
			require.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
			_, ok = fake.TaskSet(wf.Namespace, pooledTaskSetName(id))
			require.False(t, ok, "completion deleted the TaskSet")
			_, ok = fake.Result(wf.Namespace, id)
			require.True(t, ok, "the claim is left")

			stored, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
			require.NoError(t, err)
			retried, podsToDelete, err := wfutil.FormulateRetryWorkflow(ctx, stored.DeepCopy(), false, "", nil)
			require.NoError(t, err)
			assert.Empty(t, podsToDelete)
			workflows := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace)
			staleUID := oldTaskSet.UID
			if tt.archived {
				left := oldTaskSet.DeepCopy()
				left.ResourceVersion = ""
				left, err = fake.CreateTaskSet(left)
				require.NoError(t, err)
				staleUID = left.UID
				require.NoError(t, workflows.Delete(ctx, wf.Name, metav1.DeleteOptions{}))
				retried.UID, retried.ResourceVersion = "retried-uid", ""
				_, err = workflows.Create(ctx, retried, metav1.CreateOptions{})
			} else {
				_, err = workflows.Update(ctx, retried, metav1.UpdateOptions{})
			}
			require.NoError(t, err)

			woc = reconcilePooled(ctx, t, controller, wf)
			node := mustNode(t, woc, step)
			assert.Equal(t, wfv1.NodePending, node.Phase)
			require.NotNil(t, node.PooledTask)
			assert.Equal(t, wfv1.PooledTaskEnqueued, node.PooledTask.State())
			ts, ok := fake.TaskSet(wf.Namespace, pooledTaskSetName(id))
			require.True(t, ok)
			assert.Equal(t, ts.UID, node.PooledTask.TaskSetUID)
			assert.NotEqual(t, staleUID, ts.UID, "a fresh TaskSet")
			assert.True(t, dispatch.OwnedByWorkflow(ts, woc.wf.UID))

			// The left claim holds the result's name until the controller
			// discards it, as it isn't on the fresh TaskSet.
			reconcilePooled(ctx, t, controller, wf)
			_, ok = fake.Result(wf.Namespace, id)
			require.False(t, ok, "the left claim is discarded")
			unobserveResult(ctx, t, controller, wf.Namespace, id)

			h = claimPooledBy(ctx, t, fake, wf.Namespace, id, worker)
			assert.Equal(t, ts.UID, h.Claim().TaskSetUID)
			require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded}))
			observeResult(ctx, t, controller, fake, wf.Namespace, id)
			for range 3 {
				woc = reconcilePooled(ctx, t, controller, wf)
			}
			assert.Equal(t, wfv1.NodeSucceeded, mustNode(t, woc, step).Phase)
			assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
		})
	}
}
