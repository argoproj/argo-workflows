package graph

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type Visitor interface {
	Init()
	Visit(nodeID string)
}

func Visit(nodes wfv1.Nodes, visitors ...Visitor) error {
	nodeIDs, err := TopSort(nodes)
	if err != nil {
		return err
	}
	for _, visitor := range visitors {
		visitor.Init()
	}
	for _, nodeID := range nodeIDs {
		_, ok := nodes[nodeID]
		if !ok {
			continue
		}
		for _, visitor := range visitors {
			visitor.Visit(nodeID)
		}
	}
	return nil
}
