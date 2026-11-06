package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
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
