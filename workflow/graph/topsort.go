package graph

import (
	"github.com/stevenle/topsort"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

// Return a list of node IDs that can be iterated over know we'll visit each one only once and that
// we'll visit downstream dependencies before upstream ones.
// Children can be:
// * "missing" - appear in `node.Children` but missing from `nodes` - callers must cope with this and probably skip them
func TopSort(nodes wfv1.Nodes) ([]string, error) {
	g, err := graph(nodes)
	if err != nil {
		return nil, err
	}
	var res []string
	for root, value := range parents(nodes) {
		if value == 0 {
			sort, err := g.TopSort(root)
			if err != nil {
				return nil, err
			}
			res = append(res, sort...)
		}
	}
	return res, nil
}

func graph(nodes wfv1.Nodes) (*topsort.Graph, error) {
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
	return g, nil
}

func parents(nodes wfv1.Nodes) map[string]int {
	parents := make(map[string]int)
	for nodeID, node := range nodes {
		if _, exists := parents[nodeID]; !exists {
			parents[nodeID] = 0
		}
		for _, childNodeID := range node.Children {
			parents[childNodeID] = parents[childNodeID] + 1
		}
	}
	return parents
}
