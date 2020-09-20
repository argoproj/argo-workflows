package graph

import (
	"github.com/stevenle/topsort"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// Return a list of node IDs that can be iterated over know we'll visit each one only once and that
// we'll visit downstream dependencies before upstream ones.
// The parameter `nodeID` is usually the root node ID (i.e. the workflow's name).
func TopSort(in wfv1.Nodes, nodeID string) ([]string, error) {
	graph := topsort.NewGraph()
	for _, n := range in {
		graph.AddNode(n.ID)
		for _, w := range n.Children {
			graph.AddNode(w)
			err := graph.AddEdge(n.ID, w)
			if err != nil {
				return nil, err
			}
		}
	}
	return graph.TopSort(nodeID)
}
