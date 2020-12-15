package retry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestRemoveDuplicates(t *testing.T) {
	t.Run("EmptySlice", func(t *testing.T) {
		assert.Equal(t, []string{}, RemoveDuplicates([]string{}))
	})
	t.Run("RemoveDuplicates", func(t *testing.T) {
		assert.ElementsMatch(t, []string{"a", "c", "d"}, RemoveDuplicates([]string{"a", "c", "c", "d"}))
	})
}

// func GetFailHosts(nodes wfv1.Nodes, parent string) []string {
func TestGetFailHosts(t *testing.T) {
	var nodes = wfv1.Nodes{
		"retry_node": wfv1.NodeStatus{
			ID:       "retry_node",
			Phase:    wfv1.NodeFailed,
			Children: []string{"retry_node(0)", "retry_node(1)", "retry_node(2)"},
		},
		"retry_node(0)": wfv1.NodeStatus{ID: "retry_node(0)", Phase: wfv1.NodeFailed, Children: []string{}, HostNodeName: "host_1"},
		"retry_node(1)": wfv1.NodeStatus{ID: "retry_node(1)", Phase: wfv1.NodeError, Children: []string{}, HostNodeName: "host_2"},
		"retry_node(2)": wfv1.NodeStatus{ID: "retry_node(2)", Phase: wfv1.NodeRunning, Children: []string{}, HostNodeName: ""},
	}
	t.Run("NotExistParent", func(t *testing.T) {
		assert.Equal(t, GetFailHosts(nodes, "not-exist-node"), []string{})
	})
	t.Run("ParentWithoutChildren", func(t *testing.T) {
		assert.Equal(t, GetFailHosts(nodes, "retry_node(0)"), []string{})
	})
	t.Run("ParentWithChildren", func(t *testing.T) {
		assert.ElementsMatch(t, GetFailHosts(nodes, "retry_node"), []string{"host_1", "host_2"})
	})
}

func TestAddHostnamesToAffinity(t *testing.T) {
	hostNames := []string{"hostnameA", "hostnameB", "hostnameC"}
	hostSelector := "kubernetes.io/hostname"

	t.Run("EmptyAffinity", func(t *testing.T) {
		type retryNode struct {
			targetAffinity *apiv1.Affinity
		}

		targetNode := &retryNode{}
		targetNode.targetAffinity = AddHostnamesToAffinity(hostSelector, hostNames, targetNode.targetAffinity)
		targetNodeSelectorRequirement :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
		sourceNodeSelectorRequirement := apiv1.NodeSelectorRequirement{
			Key:      hostSelector,
			Operator: apiv1.NodeSelectorOpNotIn,
			Values:   hostNames,
		}
		assert.Equal(t, sourceNodeSelectorRequirement, targetNodeSelectorRequirement)
	})
	t.Run("EmptyNodeAffinity", func(t *testing.T) {
		type retryNode struct {
			targetAffinity *apiv1.Affinity
		}
		targetNode := &retryNode{
			targetAffinity: &apiv1.Affinity{
				NodeAffinity: &apiv1.NodeAffinity{},
			},
		}
		targetNode.targetAffinity = AddHostnamesToAffinity(hostSelector, hostNames, targetNode.targetAffinity)
		targetNodeSelectorRequirement :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
		sourceNodeSelectorRequirement := apiv1.NodeSelectorRequirement{
			Key:      hostSelector,
			Operator: apiv1.NodeSelectorOpNotIn,
			Values:   hostNames,
		}
		assert.Equal(t, sourceNodeSelectorRequirement, targetNodeSelectorRequirement)
	})
	t.Run("EmptyRequiredDuringSchedulingIgnoredDuringExecution", func(t *testing.T) {
		type retryNode struct {
			targetAffinity *apiv1.Affinity
		}
		targetNode := &retryNode{
			targetAffinity: &apiv1.Affinity{
				NodeAffinity: &apiv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{},
				},
			},
		}
		targetNode.targetAffinity = AddHostnamesToAffinity(hostSelector, hostNames, targetNode.targetAffinity)
		targetNodeSelectorRequirement :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
		sourceNodeSelectorRequirement := apiv1.NodeSelectorRequirement{
			Key:      hostSelector,
			Operator: apiv1.NodeSelectorOpNotIn,
			Values:   hostNames,
		}
		assert.Equal(t, sourceNodeSelectorRequirement, targetNodeSelectorRequirement)
	})
	t.Run("EmptyNodeSelectorTerms", func(t *testing.T) {
		type retryNode struct {
			targetAffinity *apiv1.Affinity
		}
		targetNode := &retryNode{
			targetAffinity: &apiv1.Affinity{
				NodeAffinity: &apiv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
						NodeSelectorTerms: []apiv1.NodeSelectorTerm{},
					},
				},
			},
		}
		targetNode.targetAffinity = AddHostnamesToAffinity(hostSelector, hostNames, targetNode.targetAffinity)
		targetNodeSelectorRequirement :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
		sourceNodeSelectorRequirement := apiv1.NodeSelectorRequirement{
			Key:      hostSelector,
			Operator: apiv1.NodeSelectorOpNotIn,
			Values:   hostNames,
		}
		assert.Equal(t, sourceNodeSelectorRequirement, targetNodeSelectorRequirement)
	})
	t.Run("EmptyMatchExpressions", func(t *testing.T) {
		type retryNode struct {
			targetAffinity *apiv1.Affinity
		}
		targetNode := &retryNode{
			targetAffinity: &apiv1.Affinity{
				NodeAffinity: &apiv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
						NodeSelectorTerms: []apiv1.NodeSelectorTerm{
							{
								MatchExpressions: []apiv1.NodeSelectorRequirement{},
							},
						},
					},
				},
			},
		}
		targetNode.targetAffinity = AddHostnamesToAffinity(hostSelector, hostNames, targetNode.targetAffinity)
		targetNodeSelectorRequirement :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
		sourceNodeSelectorRequirement := apiv1.NodeSelectorRequirement{
			Key:      hostSelector,
			Operator: apiv1.NodeSelectorOpNotIn,
			Values:   hostNames,
		}
		assert.Equal(t, sourceNodeSelectorRequirement, targetNodeSelectorRequirement)
	})
	t.Run("AddMatchExpressions", func(t *testing.T) {
		type retryNode struct {
			targetAffinity *apiv1.Affinity
		}
		sourceNodeSelectorRequirement0 := apiv1.NodeSelectorRequirement{
			Key:      "metadata.name",
			Operator: apiv1.NodeSelectorOpIn,
			Values:   []string{"hostname1", "hostname2"},
		}
		sourceNodeSelectorRequirement1 := apiv1.NodeSelectorRequirement{
			Key:      hostSelector,
			Operator: apiv1.NodeSelectorOpNotIn,
			Values:   hostNames,
		}
		targetNode := &retryNode{
			targetAffinity: &apiv1.Affinity{
				NodeAffinity: &apiv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
						NodeSelectorTerms: []apiv1.NodeSelectorTerm{
							{
								MatchExpressions: []apiv1.NodeSelectorRequirement{
									sourceNodeSelectorRequirement0,
								},
							},
						},
					},
				},
			},
		}
		targetNode.targetAffinity = AddHostnamesToAffinity(hostSelector, hostNames, targetNode.targetAffinity)
		targetNodeSelectorRequirement0 :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
		targetNodeSelectorRequirement1 :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1]
		assert.Equal(t, sourceNodeSelectorRequirement0, targetNodeSelectorRequirement0)
		assert.Equal(t, sourceNodeSelectorRequirement1, targetNodeSelectorRequirement1)
	})
	t.Run("AddMatchExpressionsSameHostSelector", func(t *testing.T) {
		type retryNode struct {
			targetAffinity *apiv1.Affinity
		}
		sourceNodeSelectorRequirement0 := apiv1.NodeSelectorRequirement{
			Key:      hostSelector,
			Operator: apiv1.NodeSelectorOpIn,
			Values:   []string{"hostname1", "hostname2"},
		}
		sourceNodeSelectorRequirement1 := apiv1.NodeSelectorRequirement{
			Key:      hostSelector,
			Operator: apiv1.NodeSelectorOpNotIn,
			Values:   hostNames,
		}
		targetNode := &retryNode{
			targetAffinity: &apiv1.Affinity{
				NodeAffinity: &apiv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
						NodeSelectorTerms: []apiv1.NodeSelectorTerm{
							{
								MatchExpressions: []apiv1.NodeSelectorRequirement{
									sourceNodeSelectorRequirement0,
								},
							},
						},
					},
				},
			},
		}
		targetNode.targetAffinity = AddHostnamesToAffinity(hostSelector, hostNames, targetNode.targetAffinity)
		targetNodeSelectorRequirement0 :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
		targetNodeSelectorRequirement1 :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1]
		assert.Equal(t, sourceNodeSelectorRequirement0, targetNodeSelectorRequirement0)
		assert.Equal(t, sourceNodeSelectorRequirement1, targetNodeSelectorRequirement1)
	})
	t.Run("AddMatchExpressionsSameOperator", func(t *testing.T) {
		type retryNode struct {
			targetAffinity *apiv1.Affinity
		}
		sourceNodeSelectorRequirement0 := apiv1.NodeSelectorRequirement{
			Key:      "metadata.name",
			Operator: apiv1.NodeSelectorOpNotIn,
			Values:   []string{"hostname1", "hostname2"},
		}
		sourceNodeSelectorRequirement1 := apiv1.NodeSelectorRequirement{
			Key:      hostSelector,
			Operator: apiv1.NodeSelectorOpNotIn,
			Values:   hostNames,
		}
		targetNode := &retryNode{
			targetAffinity: &apiv1.Affinity{
				NodeAffinity: &apiv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
						NodeSelectorTerms: []apiv1.NodeSelectorTerm{
							{
								MatchExpressions: []apiv1.NodeSelectorRequirement{
									sourceNodeSelectorRequirement0,
								},
							},
						},
					},
				},
			},
		}
		targetNode.targetAffinity = AddHostnamesToAffinity(hostSelector, hostNames, targetNode.targetAffinity)
		targetNodeSelectorRequirement0 :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
		targetNodeSelectorRequirement1 :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[1]
		assert.Equal(t, sourceNodeSelectorRequirement0, targetNodeSelectorRequirement0)
		assert.Equal(t, sourceNodeSelectorRequirement1, targetNodeSelectorRequirement1)
	})
	t.Run("MergeMatchExpressionsWithDuplicates", func(t *testing.T) {
		type retryNode struct {
			targetAffinity *apiv1.Affinity
		}
		sourceNodeSelectorRequirement0 := apiv1.NodeSelectorRequirement{
			Key:      hostSelector,
			Operator: apiv1.NodeSelectorOpNotIn,
			Values:   []string{"hostname1", "hostname2", "hostnameA"},
		}
		sourceNodeSelectorRequirementMerged := apiv1.NodeSelectorRequirement{
			Key:      hostSelector,
			Operator: apiv1.NodeSelectorOpNotIn,
			Values:   []string{"hostname1", "hostname2", "hostnameA", "hostnameB", "hostnameC"},
		}
		targetNode := &retryNode{
			targetAffinity: &apiv1.Affinity{
				NodeAffinity: &apiv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
						NodeSelectorTerms: []apiv1.NodeSelectorTerm{
							{
								MatchExpressions: []apiv1.NodeSelectorRequirement{
									sourceNodeSelectorRequirement0,
								},
							},
						},
					},
				},
			},
		}
		targetNode.targetAffinity = AddHostnamesToAffinity(hostSelector, hostNames, targetNode.targetAffinity)
		targetNodeSelectorRequirement :=
			targetNode.targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
		assert.Equal(t, sourceNodeSelectorRequirementMerged, targetNodeSelectorRequirement)
	})
}
