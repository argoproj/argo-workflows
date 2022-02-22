package retry

import (
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// GetFailHosts iterates over the node subtree and find pod in error or fail
func GetFailHosts(nodes wfv1.Nodes, retryNodeName string) []string {
	toVisit := []string{retryNodeName}
	hostNames := []string{}
	for len(toVisit) > 0 {
		n := len(toVisit) - 1
		nodeToVisit := toVisit[n]
		toVisit = toVisit[:n]
		if x, ok := nodes[nodeToVisit]; ok {
			if (x.Phase == wfv1.NodeFailed || x.Phase == wfv1.NodeError) && x.Type == wfv1.NodeTypePod {
				hostNames = append(hostNames, x.HostNodeName)
			}
			for i := 0; i < len(x.Children); i++ {
				childNode := x.Children[i]
				if y, ok := nodes[childNode]; ok {
					toVisit = append(toVisit, y.ID)
				}
			}
		}
	}
	return RemoveDuplicates(hostNames)
}

// RemoveDuplicates removes duplicate strings from slice
func RemoveDuplicates(strSlice []string) []string {
	keys := make(map[string]bool)
	outputList := []string{}
	for _, strEntry := range strSlice {
		if _, value := keys[strEntry]; !value {
			keys[strEntry] = true
			outputList = append(outputList, strEntry)
		}
	}
	return outputList
}

// AddHostnamesToAffinity will add unique hostNames to existing matchExpressions in targetAffinity with
// key hostSelector or insert new matchExpressions with operator NotIn.
func AddHostnamesToAffinity(hostSelector string, hostNames []string, targetAffinity *apiv1.Affinity) *apiv1.Affinity {
	if len(hostNames) == 0 {
		return targetAffinity
	}

	nodeSelectorRequirement := apiv1.NodeSelectorRequirement{
		Key:      hostSelector,
		Operator: apiv1.NodeSelectorOpNotIn,
		Values:   hostNames,
	}

	sourceAffinity := &apiv1.Affinity{
		NodeAffinity: &apiv1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &apiv1.NodeSelector{
				NodeSelectorTerms: []apiv1.NodeSelectorTerm{
					{
						MatchExpressions: []apiv1.NodeSelectorRequirement{
							nodeSelectorRequirement,
						},
					},
				},
			},
		},
	}

	if targetAffinity == nil {
		targetAffinity = sourceAffinity
		return targetAffinity
	}

	if targetAffinity.NodeAffinity == nil {
		targetAffinity.NodeAffinity = sourceAffinity.NodeAffinity
		return targetAffinity
	}

	targetExecution := targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	sourceExecution := sourceAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution

	if targetExecution == nil {
		targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution =
			sourceAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
		return targetAffinity
	}

	if len(targetExecution.NodeSelectorTerms) == 0 {
		targetAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms =
			sourceAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
		return targetAffinity
	}

	// find if specific NodeSelectorTerm exists and append
	for i := range targetExecution.NodeSelectorTerms {
		if len(targetExecution.NodeSelectorTerms[i].MatchExpressions) == 0 {
			targetExecution.NodeSelectorTerms[i].MatchExpressions =
				append(targetExecution.NodeSelectorTerms[i].MatchExpressions, sourceExecution.NodeSelectorTerms[0].MatchExpressions[0])
			return targetAffinity
		}

		for j := range targetExecution.NodeSelectorTerms[i].MatchExpressions {
			if targetExecution.NodeSelectorTerms[i].MatchExpressions[j].Key == hostSelector &&
				targetExecution.NodeSelectorTerms[i].MatchExpressions[j].Operator == apiv1.NodeSelectorOpNotIn {
				targetExecution.NodeSelectorTerms[i].MatchExpressions[j].Values =
					append(targetExecution.NodeSelectorTerms[i].MatchExpressions[j].Values, hostNames...)
				targetExecution.NodeSelectorTerms[i].MatchExpressions[j].Values =
					RemoveDuplicates(targetExecution.NodeSelectorTerms[i].MatchExpressions[j].Values)
				return targetAffinity
			}
		}
	}

	targetExecution.NodeSelectorTerms[0].MatchExpressions =
		append(targetExecution.NodeSelectorTerms[0].MatchExpressions, nodeSelectorRequirement)

	return targetAffinity
}
