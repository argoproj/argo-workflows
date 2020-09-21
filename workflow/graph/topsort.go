package graph

import (
	"github.com/stevenle/topsort"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// Return a list of node IDs that can be iterated over know we'll visit each one only once and that
// we'll visit downstream dependencies before upstream ones.
// The parameter `nodeID` is usually the root node ID (i.e. the workflow's name).
func TopSort(nodes wfv1.Nodes, nodeID string) ([]string, error) {
	g := topsort.NewGraph()
	for nodeID, node := range nodes {
		g.AddNode(nodeID)
		for _, childNodeID := range node.Children {
			g.AddNode(childNodeID)
			err := g.AddEdge(nodeID, childNodeID)
			if err != nil {
				return nil, err
			}
		}
	}
	return g.TopSort(nodeID)
}
