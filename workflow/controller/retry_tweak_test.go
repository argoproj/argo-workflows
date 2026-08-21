package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wfretry "github.com/argoproj/argo-workflows/v4/workflow/util/retry"
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

// TestRetryOnDifferentHostAntiAffinityType checks that the configured nodeAntiAffinity
// type decides whether the failed host lands in the required or the preferred node
// affinity term.
func TestRetryOnDifferentHostAntiAffinityType(t *testing.T) {
	const (
		retryNodeName = "retry-node"
		hostSelector  = "kubernetes.io/hostname"
		failedHost    = "test-fail-hostname"
	)

	nodes := wfv1.Nodes{
		retryNodeName: wfv1.NodeStatus{
			ID:       retryNodeName,
			Type:     wfv1.NodeTypeRetry,
			Phase:    wfv1.NodeRunning,
			Children: []string{"child"},
		},
		"child": wfv1.NodeStatus{
			ID:           "child",
			HostNodeName: failedHost,
			Type:         wfv1.NodeTypePod,
			Phase:        wfv1.NodeFailed,
		},
	}

	expectedRequirement := apiv1.NodeSelectorRequirement{
		Key:      hostSelector,
		Operator: apiv1.NodeSelectorOpNotIn,
		Values:   []string{failedHost},
	}

	tweak := func(t *testing.T, antiAffinity *wfv1.RetryNodeAntiAffinity) *apiv1.Affinity {
		t.Helper()

		pod := &apiv1.Pod{Spec: apiv1.PodSpec{Affinity: &apiv1.Affinity{}}}
		retryStrategy := wfv1.RetryStrategy{
			Affinity: &wfv1.RetryAffinity{NodeAntiAffinity: antiAffinity},
		}
		RetryOnDifferentHost(retryNodeName)(retryStrategy, nodes, pod)

		return pod.Spec.Affinity
	}

	t.Run("DefaultIsRequired", func(t *testing.T) {
		affinity := tweak(t, &wfv1.RetryNodeAntiAffinity{})
		required := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
		require.NotNil(t, required, "an unset type must keep the pre-existing required behaviour")
		assert.Equal(t, expectedRequirement, required.NodeSelectorTerms[0].MatchExpressions[0])
		assert.Empty(t, affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	})
	t.Run("ExplicitRequired", func(t *testing.T) {
		affinity := tweak(t, &wfv1.RetryNodeAntiAffinity{Type: wfv1.RetryNodeAntiAffinityRequired})
		required := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
		require.NotNil(t, required)
		assert.Equal(t, expectedRequirement, required.NodeSelectorTerms[0].MatchExpressions[0])
		assert.Empty(t, affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	})
	t.Run("Preferred", func(t *testing.T) {
		affinity := tweak(t, &wfv1.RetryNodeAntiAffinity{Type: wfv1.RetryNodeAntiAffinityPreferred})
		preferred := affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
		require.Len(t, preferred, 1)
		assert.Equal(t, wfretry.PreferredHostAntiAffinityWeight, preferred[0].Weight)
		assert.Equal(t, expectedRequirement, preferred[0].Preference.MatchExpressions[0])
		// the required term must stay empty, otherwise the retry would still be unschedulable
		// once every host has been tried, which is the whole point of the preferred type
		assert.Nil(t, affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
	})
	t.Run("NoNodeAntiAffinity", func(t *testing.T) {
		affinity := tweak(t, nil)
		assert.Nil(t, affinity.NodeAffinity)
	})
}
