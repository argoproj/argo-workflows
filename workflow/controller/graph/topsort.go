package graph

import (
	"github.com/stevenle/topsort"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// Return a list of node IDs that can be iterated over know we'll visit each one only once and that
// we'll visit downstream dependencies before upstream ones.
// The parameter `nodeID` is usually the root node ID (i.e. the workflow's name).
func TopSort(nodes wfv1.Nodes, nodeID string) ([]string, error) {
	graph := topsort.NewGraph()
	for nodeID, node := range nodes {
		graph.AddNode(nodeID)
		for _, childNodeID := range node.Children {
			graph.AddNode(childNodeID)
			err := graph.AddEdge(nodeID, childNodeID)
			if err != nil {
				return nil, err
			}
		}
	}
	return graph.TopSort(nodeID)
}
