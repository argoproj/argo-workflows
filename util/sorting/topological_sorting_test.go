package sorting

import (
	"strings"
	"testing"
)

func graphToString(graph []*TopologicalSortingNode) string {
	var nodeNames []string
	for _, node := range graph {
		nodeNames = append(nodeNames, node.NodeName)
	}
	return strings.Join(nodeNames, ",")
}

func TestTopologicalSorting_EmptyInput(t *testing.T) {
	result, err := TopologicalSorting([]*TopologicalSortingNode{})
	if err != nil {
		t.Error(err)
	}
	if len(result) != 0 {
		t.Error("return value not empty", result)
	}
}

func TestTopologicalSorting_DuplicatedNode(t *testing.T) {
	graph := []*TopologicalSortingNode{
		{
			NodeName: "a",
		},
		{
			NodeName: "b",
			Dependencies: []string{
				"a",
			},
		},
		{
			NodeName: "a",
			Dependencies: []string{
				"b",
			},
		},
	}
	_, err := TopologicalSorting(graph)
	if err == nil {
		t.Error("error missing")
	}
}

func TestTopologicalSorting_InvalidDependency(t *testing.T) {
	graph := []*TopologicalSortingNode{
		{
			NodeName: "a",
		},
		{
			NodeName: "b",
			Dependencies: []string{
				"a",
			},
		},
		{
			NodeName: "c",
			Dependencies: []string{
				"a",
				"d",
			},
		},
	}
	_, err := TopologicalSorting(graph)
	if err == nil {
		t.Error("error missing")
	}
}

func TestTopologicalSorting_GraphWithCycle(t *testing.T) {
	graph := []*TopologicalSortingNode{
		{
			NodeName: "a",
			Dependencies: []string{
				"b",
			},
		},
		{
			NodeName: "b",
			Dependencies: []string{
				"a",
			},
		},
	}
	_, err := TopologicalSorting(graph)
	if err == nil {
		t.Error("error missing")
	}
}

func TestTopologicalSorting_GraphWithCycle2(t *testing.T) {
	graph := []*TopologicalSortingNode{
		{
			NodeName: "a",
		},
		{
			NodeName: "b",
			Dependencies: []string{
				"a",
				"c",
			},
		},
		{
			NodeName: "c",
			Dependencies: []string{
				"a",
				"b",
			},
		},
	}
	_, err := TopologicalSorting(graph)
	if err == nil {
		t.Error("error missing")
	}
}

func TestTopologicalSorting_ValidInput(t *testing.T) {
	graph := []*TopologicalSortingNode{
		{
			NodeName: "a",
		},
		{
			NodeName: "b",
			Dependencies: []string{
				"a",
			},
		},
		{
			NodeName: "c",
			Dependencies: []string{
				"b",
			},
		},
	}
	result, err := TopologicalSorting(graph)
	if err != nil {
		t.Error(err)
	}
	resultStr := graphToString(result)
	if resultStr != "a,b,c" {
		t.Error("wrong output", resultStr)
	}
}

func TestTopologicalSorting_ValidInput2(t *testing.T) {
	graph := []*TopologicalSortingNode{
		{
			NodeName: "a",
		},
		{
			NodeName: "b",
			Dependencies: []string{
				"a",
			},
		},
		{
			NodeName: "c",
			Dependencies: []string{
				"a",
			},
		},
		{
			NodeName: "d",
			Dependencies: []string{
				"b",
				"c",
			},
		},
	}
	result, err := TopologicalSorting(graph)
	if err != nil {
		t.Error(err)
	}
	resultStr := graphToString(result)
	if resultStr != "a,b,c,d" && resultStr != "a,c,b,d" {
		t.Error("wrong output", resultStr)
	}
}

func TestTopologicalSorting_ValidInput3(t *testing.T) {
	graph := []*TopologicalSortingNode{
		{
			NodeName: "a",
		},
		{
			NodeName: "b",
		},
	}
	result, err := TopologicalSorting(graph)
	if err != nil {
		t.Error(err)
	}
	resultStr := graphToString(result)
	if resultStr != "a,b" && resultStr != "b,a" {
		t.Error("wrong output", resultStr)
	}
}
