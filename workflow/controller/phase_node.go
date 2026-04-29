package controller

import (
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// A phaseNode is a node in a BFS of all nodes for the purposes of determining overall DAG phase. nodeId is the corresponding
// nodeId and phase is the current branchPhase associated with the node
type phaseNode struct {
	nodeID string
	phase  wfv1.NodePhase
}

func generatePhaseNodes(children []string, branchPhase wfv1.NodePhase) []phaseNode {
	out := make([]phaseNode, len(children))
	for i, child := range children {
		out[i] = phaseNode{nodeID: child, phase: branchPhase}
	}
	return out
}

type uniquePhaseNodeQueue struct {
	seen  map[string]bool
	queue []phaseNode
}

// A uniquePhaseNodeQueue is a queue that only accepts a phaseNode only once during its life. If a node with a
// phaseNode is added while another had already been added before, the add will not succeed. Even if a phaseNode
// is added, popped, and re-added, the re-add will not succeed. Failed adds fail silently. Note that two phaseNodes
// with the same nodeId but different phases may be added, but only once per nodeId-phase combination. This is to ensure
// that branches with different branchPhases can still be processed: if an Omitted node is reached first from a step
// that succeeded, we consider the omitted node succeeded. However, it may be subsequently reached from another step
// that did not succeed. In that case we want to update the deduced status of the omitted node, and we may only do so by
// adding it to the queue again.
func newUniquePhaseNodeQueue(nodes ...phaseNode) *uniquePhaseNodeQueue {
	uq := &uniquePhaseNodeQueue{
		seen:  make(map[string]bool),
		queue: []phaseNode{},
	}
	uq.add(nodes...)
	return uq
}

// If a phaseNode has already existed, it will not be added silently
func (uq *uniquePhaseNodeQueue) add(nodes ...phaseNode) {
	for _, node := range nodes {
		key := fmt.Sprintf("%s-%s", node.nodeID, node.phase)
		if _, ok := uq.seen[key]; !ok {
			uq.seen[key] = true
			uq.queue = append(uq.queue, node)
		}
	}
}

func (uq *uniquePhaseNodeQueue) pop() phaseNode {
	var head phaseNode
	head, uq.queue = uq.queue[0], uq.queue[1:]
	return head
}

func (uq *uniquePhaseNodeQueue) empty() bool {
	return uq.len() == 0
}

func (uq *uniquePhaseNodeQueue) len() int {
	return len(uq.queue)
}
