package parameterization

import (
	"fmt"
	"strconv"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// Retry attempt identity is the native parent-child relationship and canonical
// attempt name, never a display label or a guessed template-name prefix.
// This subset supports executable Pod leaves, not nested DAG retry expansions.
func retryAttemptParents(nodes wfv1.Nodes) (map[string]string, error) {
	parents := map[string]string{}
	for id, group := range nodes {
		if group.Type != wfv1.NodeTypeRetry {
			continue
		}
		for index, childID := range group.Children {
			child, ok := nodes[childID]
			if !ok || child.Type != wfv1.NodeTypePod || child.Name != group.Name+"("+strconv.Itoa(index)+")" || child.BoundaryID != group.BoundaryID || child.TemplateName != group.TemplateName || child.TemplateRef != nil {
				return nil, fmt.Errorf("retry group contains an unverified attempt identity")
			}
			if _, duplicate := parents[childID]; duplicate {
				return nil, fmt.Errorf("retry attempt belongs to multiple groups")
			}
			parents[childID] = id
		}
	}
	return parents, nil
}
