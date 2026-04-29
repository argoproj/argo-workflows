package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestFindRetryNode(t *testing.T) {
	allNodes := wfv1.Nodes{
		"A1": wfv1.NodeStatus{
			ID:           "A1",
			Type:         wfv1.NodeTypeSteps,
			Phase:        wfv1.NodeRunning,
			BoundaryID:   "",
			Children:     []string{"B1", "B2", "B3"},
			TemplateName: "tmpl1",
		},
		"B1": wfv1.NodeStatus{
			ID:           "B1",
			Type:         wfv1.NodeTypeSkipped,
			Phase:        wfv1.NodeSkipped,
			BoundaryID:   "A1",
			Children:     []string{},
			TemplateName: "tmpl2",
		},
		// retry node containing steps
		"B2": wfv1.NodeStatus{
			ID:           "B2",
			Type:         wfv1.NodeTypeRetry,
			Phase:        wfv1.NodeRunning,
			BoundaryID:   "A1",
			Children:     []string{"C1"},
			TemplateName: "tmpl1",
		},
		"C1": wfv1.NodeStatus{
			ID:           "C1",
			Type:         wfv1.NodeTypeSteps,
			Phase:        wfv1.NodeRunning,
			BoundaryID:   "A1",
			Children:     []string{"D1", "D2"},
			TemplateName: "tmpl2",
		},
		"D1": wfv1.NodeStatus{
			ID:           "D1",
			Type:         wfv1.NodeTypeSkipped,
			Phase:        wfv1.NodeSkipped,
			BoundaryID:   "C1",
			Children:     []string{},
			TemplateName: "tmpl2",
		},
		"D2": wfv1.NodeStatus{
			ID:           "D2",
			Type:         wfv1.NodeTypePod,
			Phase:        wfv1.NodeRunning,
			BoundaryID:   "C1",
			Children:     []string{},
			TemplateName: "tmpl2",
		},
		// retry node containing single step and templteRef
		"B3": wfv1.NodeStatus{
			ID:         "B3",
			Type:       wfv1.NodeTypeRetry,
			Phase:      wfv1.NodeRunning,
			BoundaryID: "A1",
			Children:   []string{"C2"},
			TemplateRef: &wfv1.TemplateRef{
				Name:     "tmpl1",
				Template: "tmpl3",
			},
		},
		"C2": wfv1.NodeStatus{
			ID:           "C2",
			Type:         wfv1.NodeTypePod,
			Phase:        wfv1.NodeRunning,
			BoundaryID:   "A1",
			Children:     []string{},
			TemplateName: "tmpl2",
		},
	}
	t.Run("Expect to find retry node", func(t *testing.T) {
		node := allNodes["B2"]
		assert.Equal(t, FindRetryNode(allNodes, "D2"), &node)
	})
	t.Run("Expect to get nil", func(t *testing.T) {
		a := FindRetryNode(allNodes, "A1")
		assert.Nil(t, a)
	})
	t.Run("Expect to find retry node has TemplateRef", func(t *testing.T) {
		node := allNodes["B3"]
		assert.Equal(t, FindRetryNode(allNodes, "C2"), &node)
	})
}
