package sorting

import (
	"fmt"
)

type TopologicalSortingNode struct {
	NodeName     string
	Dependencies []string
}

func TopologicalSorting(graph []*TopologicalSortingNode) ([]*TopologicalSortingNode, error) {
	priorNodeCountMap := make(map[string]int, len(graph))               // nodeName -> priorNodeCount
	nextNodeMap := make(map[string][]string, len(graph))                // nodeName -> nextNodeList
	nodeNameMap := make(map[string]*TopologicalSortingNode, len(graph)) // nodeName -> node
	for _, node := range graph {
		if _, ok := nodeNameMap[node.NodeName]; ok {
			return nil, fmt.Errorf("duplicated nodeName %s", node.NodeName)
		}
		nodeNameMap[node.NodeName] = node
		priorNodeCountMap[node.NodeName] = len(node.Dependencies)
	}
	for _, node := range graph {
		for _, dependency := range node.Dependencies {
			if _, ok := nodeNameMap[dependency]; !ok {
				return nil, fmt.Errorf("invalid dependency %s", dependency)
			}
			nextNodeMap[dependency] = append(nextNodeMap[dependency], node.NodeName)
		}
	}

	queue := make([]*TopologicalSortingNode, len(graph))
	head, tail := 0, 0
	for nodeName, priorNodeCount := range priorNodeCountMap {
		if priorNodeCount == 0 {
			queue[tail] = nodeNameMap[nodeName]
			tail++
		}
	}

	for head < len(queue) {
		curr := queue[head]
		if curr == nil {
			return nil, fmt.Errorf("graph with cycle")
		}
		for _, next := range nextNodeMap[curr.NodeName] {
			if priorNodeCountMap[next] > 0 {
				if priorNodeCountMap[next] == 1 {
					queue[tail] = nodeNameMap[next]
					tail++
				}
				priorNodeCountMap[next]--
			}
		}
		head++
	}

	return queue, nil
}
