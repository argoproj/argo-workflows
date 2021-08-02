package retry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestRemoveDuplicates(t *testing.T) {
	t.Run("EmptySlice", func(t *testing.T) {
		assert.Equal(t, []string{}, RemoveDuplicates([]string{}))
	})
	t.Run("RemoveDuplicates", func(t *testing.T) {
		assert.ElementsMatch(t, []string{"a", "c", "d"}, RemoveDuplicates([]string{"a", "c", "c", "d"}))
	})
}

func TestGetFailHosts(t *testing.T) {
	nodes := wfv1.Nodes{
		"retry": wfv1.NodeStatus{
			ID:           "A1",
			HostNodeName: "host1",
			Type:         wfv1.NodeTypeRetry,
			Phase:        wfv1.NodeRunning,
			Children:     []string{"n1", "stepgroup"},
		},
		"n1": wfv1.NodeStatus{
			ID:           "n1",
			HostNodeName: "hostn1",
			Type:         wfv1.NodeTypePod,
			Phase:        wfv1.NodeFailed,
			Children:     []string{},
		},
		"stepgroup": wfv1.NodeStatus{
			ID:           "stepgroup",
			HostNodeName: "host2",
			Type:         wfv1.NodeTypeStepGroup,
			Phase:        wfv1.NodeError,
			Children:     []string{"steps"},
		},
		"steps": wfv1.NodeStatus{
			ID:           "steps",
			HostNodeName: "host4",
			Type:         wfv1.NodeTypeSteps,
			Phase:        wfv1.NodeError,
			Children:     []string{"n2", "n3"},
		},
		"n2": wfv1.NodeStatus{
			ID:           "n2",
			HostNodeName: "hostn2",
			Type:         wfv1.NodeTypePod,
			Phase:        wfv1.NodeRunning,
			Children:     []string{},
		},
		"n3": wfv1.NodeStatus{
			ID:           "n3",
			HostNodeName: "hostn3",
			Type:         wfv1.NodeTypePod,
			Phase:        wfv1.NodeError,
			Children:     []string{},
		},
	}
	t.Run("NotExistParent", func(t *testing.T) {
		assert.Equal(t, GetFailHosts(nodes, "not-exist-node"), []string{})
	})
	t.Run("ParentWithoutChildrenPodTypeError", func(t *testing.T) {
		assert.Equal(t, GetFailHosts(nodes, "n3"), []string{"hostn3"})
	})
	t.Run("ParentWithoutChildrenPodTypeRunning", func(t *testing.T) {
		assert.Equal(t, GetFailHosts(nodes, "n2"), []string{})
	})
	t.Run("ParentWithChildrenFromRetryNode", func(t *testing.T) {
		assert.ElementsMatch(t, GetFailHosts(nodes, "retry"), []string{"hostn1", "hostn3"})
	})
	t.Run("ParentWithChildrenFromNonRetryNode", func(t *testing.T) {
		assert.ElementsMatch(t, GetFailHosts(nodes, "steps"), []string{"hostn3"})
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
	t.Run("MergeWithExistingSpecAffinity", func(t *testing.T) {
		existingNode := &apiv1.Affinity{
			NodeAffinity: &apiv1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []apiv1.PreferredSchedulingTerm{
					{
						Preference: apiv1.NodeSelectorTerm{
							MatchExpressions: []apiv1.NodeSelectorRequirement{
								{
									Key:      "topology.kubernetes.io/region",
									Operator: apiv1.NodeSelectorOpNotIn,
									Values:   []string{"REG1", "REG2"},
								},
							},
						},
						Weight: 100,
					},
				},
				RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
					NodeSelectorTerms: []apiv1.NodeSelectorTerm{
						{
							MatchExpressions: []apiv1.NodeSelectorRequirement{
								{
									Key:      "gpu.nvidia.com/model",
									Operator: apiv1.NodeSelectorOpNotIn,
									Values:   []string{"GeForce_RTX_A6000", "NVidia_A100"},
								},
								{
									Key:      hostSelector,
									Operator: apiv1.NodeSelectorOpNotIn,
									Values:   []string{"hostname1", "hostname2", "hostnameA"},
								},
							},
						},
					},
				},
			},
		}
		mergedNode := &apiv1.Affinity{
			NodeAffinity: &apiv1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []apiv1.PreferredSchedulingTerm{
					{
						Preference: apiv1.NodeSelectorTerm{
							MatchExpressions: []apiv1.NodeSelectorRequirement{
								{
									Key:      "topology.kubernetes.io/region",
									Operator: apiv1.NodeSelectorOpNotIn,
									Values:   []string{"REG1", "REG2"},
								},
							},
						},
						Weight: 100,
					},
				},
				RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
					NodeSelectorTerms: []apiv1.NodeSelectorTerm{
						{
							MatchExpressions: []apiv1.NodeSelectorRequirement{
								{
									Key:      "gpu.nvidia.com/model",
									Operator: apiv1.NodeSelectorOpNotIn,
									Values:   []string{"GeForce_RTX_A6000", "NVidia_A100"},
								},
								{
									Key:      hostSelector,
									Operator: apiv1.NodeSelectorOpNotIn,
									Values:   []string{"hostname1", "hostname2", "hostnameA", "hostnameB", "hostnameC"},
								},
							},
						},
					},
				},
			},
		}
		targetNode := AddHostnamesToAffinity(hostSelector, hostNames, existingNode)
		assert.Equal(t, targetNode, mergedNode)
	})
}
