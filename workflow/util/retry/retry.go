package retry

import (
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// GetFailHosts returns slice of all child nodes with fail or error status
func GetFailHosts(nodes wfv1.Nodes, parent string) []string {
	var hostNames = []string{}
	failNodes := nodes.Children(parent).
		Filter(func(x wfv1.NodeStatus) bool { return x.Phase == wfv1.NodeFailed || x.Phase == wfv1.NodeError }).
		Map(func(x wfv1.NodeStatus) interface{} { return x.HostNodeName })
	for _, hostName := range failNodes {
		hostNames = append(hostNames, hostName.(string))
	}
	return hostNames
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
