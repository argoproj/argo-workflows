package graph

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type Visitor interface {
	Init()
	Visit(nodeID string)
}

func Visit(nodes wfv1.Nodes, visitor Visitor) error {
	nodeIDs, err := TopSort(nodes)
	if err != nil {
		return err
	}
	visitor.Init()
	for _, nodeID := range nodeIDs {
		_, ok := nodes[nodeID]
		if !ok {
			continue
		}
		visitor.Visit(nodeID)
	}
	return nil
}
