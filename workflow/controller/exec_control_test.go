package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestKillDaemonChildrenUnmarkPod(t *testing.T) {
	cancel, controller := newController()
	defer cancel()

	woc := newWorkflowOperationCtx(&v1alpha1.Workflow{
		Status: v1alpha1.WorkflowStatus{
			Nodes: v1alpha1.Nodes{
				"a": v1alpha1.NodeStatus{
					ID:         "a",
					BoundaryID: "a",
					Daemoned:   pointer.BoolPtr(true),
				},
			},
		},
	}, controller)

	assert.NotNil(t, woc.wf.Status.Nodes["a"].Daemoned)
	// Error will be that it cannot find the pod, but we only care about the node status for this test
	woc.killDaemonedChildren("a")
	assert.Nil(t, woc.wf.Status.Nodes["a"].Daemoned)
}
