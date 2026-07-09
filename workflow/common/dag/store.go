package dag

import (
	"context"
	"fmt"
	"strings"
	"sync"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// workflowStore adapts Argo's Workflow.Status.Nodes as a store for task states.
// It provides access to workflow node states using task names as keys.
type workflowStore struct {
	nodes        wfv1.Nodes
	boundaryID   string
	boundaryName string
	workflow     *wfv1.Workflow

	mu sync.RWMutex
	// phases caches evaluator-managed phases (e.g. Omitted for unreachable tasks)
	phases map[Key]wfv1.NodePhase
}

// newWorkflowStore creates a new workflowStore from a workflow and DAG context.
func newWorkflowStore(wf *wfv1.Workflow, boundaryID, boundaryName string) *workflowStore {
	return &workflowStore{
		nodes:        wf.Status.Nodes,
		boundaryID:   boundaryID,
		boundaryName: boundaryName,
		workflow:     wf,
		phases:       make(map[Key]wfv1.NodePhase),
	}
}

// taskNodeName computes the node name for a task (same as dagContext.taskNodeName).
func (s *workflowStore) taskNodeName(taskName string) string {
	if strings.HasPrefix(taskName, "[") {
		return fmt.Sprintf("%s%s", s.boundaryName, taskName)
	}
	return fmt.Sprintf("%s.%s", s.boundaryName, taskName)
}

// taskNameFromNodeName is the inverse of taskNodeName: it strips the boundary
// prefix from a node name to recover the task name. Returns the input unchanged
// if it doesn't look like a child of this boundary (no proper separator).
//
// taskNodeName always inserts "." (DAG) or "[" (Steps) between the boundary
// and the task name, so we require the same separator on the way back —
// otherwise a node merely sharing the boundary as a string prefix (e.g.
// "boundaryother") would be incorrectly truncated to "other".
func (s *workflowStore) taskNameFromNodeName(nodeName string) string {
	rest, ok := strings.CutPrefix(nodeName, s.boundaryName)
	if !ok || rest == "" {
		return nodeName
	}
	switch rest[0] {
	case '.':
		return rest[1:]
	case '[':
		return rest
	default:
		return nodeName
	}
}

// taskNodeID computes the node ID for a task (same as dagContext.taskNodeID).
func (s *workflowStore) taskNodeID(taskName string) string {
	nodeName := s.taskNodeName(taskName)
	return s.workflow.NodeID(nodeName)
}

// getPhase returns the current phase of a task.
// It checks the workflow nodes first, then falls back to the internal phases map
// for terminal states (like Omitted) that the evaluator manages but don't have
// corresponding workflow nodes yet.
func (s *workflowStore) getPhase(_ context.Context, key Key) wfv1.NodePhase {
	// Check workflow nodes first (source of truth for actual execution state)
	nodeID := s.taskNodeID(key)
	node, err := s.nodes.Get(nodeID)
	if err == nil {
		// Node exists — use its phase
		if node.IsDaemoned() && node.Phase == wfv1.NodeRunning {
			return wfv1.NodeSucceeded
		}
		return node.Phase
	}

	// No workflow node. Check internal phases map for evaluator-managed states
	// (e.g. Omitted marking from unreachable depends conditions).
	s.mu.RLock()
	if phase, ok := s.phases[key]; ok && isTerminalPhase(phase) {
		s.mu.RUnlock()
		return phase
	}
	s.mu.RUnlock()

	return wfv1.NodePending
}

// setPhase updates the phase of a task.
// This is primarily used by the evaluator to track internal state;
// actual node phase updates happen through the workflow controller.
func (s *workflowStore) setPhase(_ context.Context, key Key, phase wfv1.NodePhase) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.phases[key] = phase
}

// getNode returns the raw node status for a task.
func (s *workflowStore) getNode(taskName string) *wfv1.NodeStatus {
	nodeID := s.taskNodeID(taskName)
	node, err := s.nodes.Get(nodeID)
	if err != nil {
		return nil
	}
	return node
}

// getTaskChildNodes returns the child nodes for a task.
func (s *workflowStore) getTaskChildNodes(taskName string) []*wfv1.NodeStatus {
	node := s.getNode(taskName)
	if node == nil {
		return nil
	}

	children := make([]*wfv1.NodeStatus, 0, len(node.Children))
	for _, childID := range node.Children {
		child, err := s.nodes.Get(childID)
		if err == nil {
			children = append(children, child)
		}
	}
	return children
}

// getRetryChildren returns the non-hook child nodes of a Retry node,
// ordered by their position in the Children slice (attempt order).
func (s *workflowStore) getRetryChildren(taskName string) []*wfv1.NodeStatus {
	node := s.getNode(taskName)
	if node == nil || node.Type != wfv1.NodeTypeRetry {
		return nil
	}
	var children []*wfv1.NodeStatus
	for _, childID := range node.Children {
		child, err := s.nodes.Get(childID)
		if err != nil {
			continue
		}
		if child.NodeFlag != nil && child.NodeFlag.Hooked {
			continue
		}
		children = append(children, child)
	}
	return children
}

// getTaskGroupChildren returns the schedulable expanded children of a TaskGroup
// node (i.e. excludes hook and retry-attempt scaffolding nodes). Returns nil if
// the named task has no node, or its node isn't a TaskGroup.
func (s *workflowStore) getTaskGroupChildren(taskName string) []*wfv1.NodeStatus {
	node := s.getNode(taskName)
	if node == nil || node.Type != wfv1.NodeTypeTaskGroup {
		return nil
	}
	var children []*wfv1.NodeStatus
	for _, childID := range node.Children {
		child, err := s.nodes.Get(childID)
		if err != nil {
			continue
		}
		if child.NodeFlag != nil && (child.NodeFlag.Hooked || child.NodeFlag.Retried) {
			continue
		}
		children = append(children, child)
	}
	return children
}

// areHooksFulfilled checks if all lifecycle hooks for a task are fulfilled.
func (s *workflowStore) areHooksFulfilled(taskName string) bool {
	node := s.getNode(taskName)
	if node == nil {
		return true
	}

	for _, childID := range node.Children {
		childNode, err := s.nodes.Get(childID)
		if err != nil {
			continue
		}
		if childNode.NodeFlag != nil && childNode.NodeFlag.Hooked && !childNode.Fulfilled() {
			return false
		}
	}

	return true
}
