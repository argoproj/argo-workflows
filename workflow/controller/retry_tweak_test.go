package controller

import (
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestFindRetryNodeWithTemplate(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf
  namespace: argo
spec:
  entrypoint: main
  templates:
  - name: main
    retryStrategy:
      limit: 10
      retryPolicy: "Always"
      affinity:
        nodeAntiAffinity: {}
    container:
      image: my-image
      command:
      - exit
      - "1"
  `)
	wf.Status.Nodes = wfv1.Nodes{
		"my-wf": wfv1.NodeStatus{
			ID:           "my-wf",
			Type:         wfv1.NodeTypeRetry,
			Children:     []string{"my-wf-4242424242"},
			TemplateName: "main",
		},
		"my-wf-4242424242": wfv1.NodeStatus{
			ID:           "my-wf-4242424242",
			Type:         wfv1.NodeTypePod,
			TemplateName: "main",
		},
	}
	cancel, controller := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, controller)

	t.Run("Expect to find retry node", func(t *testing.T) {
		node := wf.Status.Nodes["my-wf"]
		assert.Equal(t, woc.FindRetryNode(wf.Status.Nodes, "my-wf-4242424242"), &node)
	})
}

func TestFindRetryNodeWithBoundaryID(t *testing.T) {
	allNodes := wfv1.Nodes{
		"A1": wfv1.NodeStatus{
			ID:           "A1",
			Type:         wfv1.NodeTypeSteps,
			Phase:        wfv1.NodeRunning,
			BoundaryID:   "",
			Children:     []string{"B1", "B2"},
			TemplateName: "tmpl1",
		},
		"B1": wfv1.NodeStatus{
			ID:           "B1",
			Type:         wfv1.NodeTypeSkipped,
			Phase:        wfv1.NodeSkipped,
			BoundaryID:   "",
			Children:     []string{},
			TemplateName: "tmpl2",
		},
		// retry node
		"B2": wfv1.NodeStatus{
			ID:           "B2",
			Type:         wfv1.NodeTypeRetry,
			Phase:        wfv1.NodeRunning,
			BoundaryID:   "",
			Children:     []string{"C1"},
			TemplateName: "tmpl1",
		},
		"C1": wfv1.NodeStatus{
			ID:           "C1",
			Type:         wfv1.NodeTypeSteps,
			Phase:        wfv1.NodeRunning,
			BoundaryID:   "",
			Children:     []string{"D1", "D2"},
			TemplateName: "tmpl2",
		},
		"D1": wfv1.NodeStatus{
			ID:           "D1",
			Type:         wfv1.NodeTypeSkipped,
			Phase:        wfv1.NodeSkipped,
			BoundaryID:   "A1",
			Children:     []string{},
			TemplateName: "tmpl2",
		},
		"D2": wfv1.NodeStatus{
			ID:           "D2",
			Type:         wfv1.NodeTypePod,
			Phase:        wfv1.NodeRunning,
			BoundaryID:   "A1",
			Children:     []string{},
			TemplateName: "tmpl2",
		},
	}
	wf := &wfv1.Workflow{}
	wf.Status.Nodes = allNodes
	cancel, controller := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, controller)

	t.Run("Expect to find retry node", func(t *testing.T) {
		node := allNodes["B2"]
		assert.Equal(t, woc.FindRetryNode(allNodes, "D2"), &node)
	})
	t.Run("Expect to get nil", func(t *testing.T) {
		a := woc.FindRetryNode(allNodes, "A1")
		assert.Nil(t, a)
	})
}
