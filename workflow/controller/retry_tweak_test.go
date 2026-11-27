package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

func TestFindRetryNodeByName(t *testing.T) {
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "wf"}}
	wf.Status.Nodes = wfv1.Nodes{}
	add := func(name string, nodeType wfv1.NodeType, boundary string, children ...string) string {
		id := wf.NodeID(name)
		childIDs := make([]string, len(children))
		for i, child := range children {
			childIDs[i] = wf.NodeID(child)
		}
		boundaryID := ""
		if boundary != "" {
			boundaryID = wf.NodeID(boundary)
		}
		wf.Status.Nodes[id] = wfv1.NodeStatus{ID: id, Name: name, Type: nodeType, BoundaryID: boundaryID, Children: childIDs}
		return id
	}
	root := add("wf", wfv1.NodeTypeSteps, "", "wf[0].steps", "wf[0].ref", "wf[0].plain", "wf[0].items(0:a)", "wf[0].decoy")
	// A retried steps template: its pod's retry parent is found through the boundary.
	stepsRetry := add("wf[0].steps", wfv1.NodeTypeRetry, "wf", "wf[0].steps(0)")
	add("wf[0].steps(0)", wfv1.NodeTypeSteps, "wf", "wf[0].steps(0)[0].pod")
	stepsPod := add("wf[0].steps(0)[0].pod", wfv1.NodeTypePod, "wf[0].steps(0)")
	// A retried pod on its second attempt.
	refRetry := add("wf[0].ref", wfv1.NodeTypeRetry, "wf", "wf[0].ref(0)", "wf[0].ref(1)")
	add("wf[0].ref(0)", wfv1.NodeTypePod, "wf")
	refPod := add("wf[0].ref(1)", wfv1.NodeTypePod, "wf")
	plainPod := add("wf[0].plain", wfv1.NodeTypePod, "wf")
	itemsPod := add("wf[0].items(0:a)", wfv1.NodeTypePod, "wf")
	// Named like an attempt, but its parent is not a retry node.
	add("wf[0].decoy", wfv1.NodeTypeSteps, "wf", "wf[0].decoy(0)")
	decoyPod := add("wf[0].decoy(0)", wfv1.NodeTypePod, "wf[0].decoy")

	t.Run("Finds the retry node through the boundary", func(t *testing.T) {
		assert.Equal(t, stepsRetry, FindRetryNodeByName(wf, stepsPod).ID)
	})
	t.Run("Finds the retry node of a later attempt", func(t *testing.T) {
		assert.Equal(t, refRetry, FindRetryNodeByName(wf, refPod).ID)
	})
	t.Run("Finds nothing without a retry parent", func(t *testing.T) {
		for _, id := range []string{root, plainPod, itemsPod, decoyPod, "missing"} {
			assert.Nil(t, FindRetryNodeByName(wf, id), wf.Status.Nodes[id].Name)
		}
	})
	t.Run("Agrees with FindRetryNode for every node", func(t *testing.T) {
		for id, node := range wf.Status.Nodes {
			assert.Equal(t, FindRetryNode(wf.Status.Nodes, id), FindRetryNodeByName(wf, id), node.Name)
		}
	})
}

func TestRetryParentName(t *testing.T) {
	for name, want := range map[string]string{
		"wf[0].step(0)":        "wf[0].step",
		"wf[0].step(12)":       "wf[0].step",
		"wf[0].step(0:a)(3)":   "wf[0].step(0:a)",
		"wf[0].step(0:a)":      "",
		"wf[0].step":           "",
		"wf[0].step()":         "",
		"wf[0].step(-1)":       "",
		"wf[0].step(1)[0].pod": "",
	} {
		got, ok := retryParentName(name)
		assert.Equal(t, want != "", ok, name)
		assert.Equal(t, want, got, name)
	}
}
