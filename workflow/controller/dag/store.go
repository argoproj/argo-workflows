package dag

import (
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// workflowStore adapts Argo's Workflow.Status.Nodes as a store for task states.
// It provides access to workflow node states using task names as keys.
type workflowStore struct {
	nodes        wfv1.Nodes
	boundaryID   string
	boundaryName string
	workflow     *wfv1.Workflow
}

// newWorkflowStore creates a new workflowStore from a workflow and DAG context.
func newWorkflowStore(wf *wfv1.Workflow, boundaryID, boundaryName string) *workflowStore {
	return &workflowStore{
		nodes:        wf.Status.Nodes,
		boundaryID:   boundaryID,
		boundaryName: boundaryName,
		workflow:     wf,
	}
}

// TaskNodeName is the node name of a task within a DAG/Steps boundary: DAG
// tasks are "<boundary>.<task>", Steps tasks (already named "[i].<step>") are
// "<boundary>[i].<step>". It is the single definition of the convention: the
// Engine creates nodes with it and the evaluator looks them up with it.
func TaskNodeName(boundaryName, taskName string) string {
	if strings.HasPrefix(taskName, "[") {
		return boundaryName + taskName
	}
	return boundaryName + "." + taskName
}

func (s *workflowStore) taskNodeName(taskName string) string {
	return TaskNodeName(s.boundaryName, taskName)
}

// taskNodeID computes the node ID for a task.
func (s *workflowStore) taskNodeID(taskName string) string {
	return s.workflow.ResolveNodeID(s.taskNodeName(taskName))
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

// getTaskGroupChildren returns the schedulable expanded children of a TaskGroup
// node (i.e. excludes hook and retry-attempt scaffolding nodes). Returns nil if
// the named task has no node, or its node isn't a TaskGroup.
func (s *workflowStore) getTaskGroupChildren(taskName string) []*wfv1.NodeStatus {
	children, _ := s.taskGroupChildren(taskName)
	return children
}

// taskGroupChildren is getTaskGroupChildren plus whether any listed child is
// missing from the node map (pruned or offloaded), which the filtered length
// alone cannot tell apart from a hook child.
func (s *workflowStore) taskGroupChildren(taskName string) (children []*wfv1.NodeStatus, missing bool) {
	node := s.getNode(taskName)
	if node == nil || node.Type != wfv1.NodeTypeTaskGroup {
		return nil, false
	}
	for _, childID := range node.Children {
		child, err := s.nodes.Get(childID)
		if err != nil {
			missing = true
			continue
		}
		if child.NodeFlag != nil && (child.NodeFlag.Hooked || child.NodeFlag.Retried) {
			continue
		}
		children = append(children, child)
	}
	return children, missing
}

// areHooksFulfilled checks if all lifecycle hooks for a task are fulfilled.
// An expanded task's exit hooks run per item and hang off its item nodes, so
// for a TaskGroup the item nodes' hooks are checked too.
func (s *workflowStore) areHooksFulfilled(taskName string) bool {
	node := s.getNode(taskName)
	if node == nil {
		return true
	}
	if !s.nodeHooksFulfilled(node) {
		return false
	}
	if node.Type == wfv1.NodeTypeTaskGroup {
		for _, child := range s.getTaskGroupChildren(taskName) {
			if !s.nodeHooksFulfilled(child) {
				return false
			}
		}
	}
	return true
}

func (s *workflowStore) nodeHooksFulfilled(node *wfv1.NodeStatus) bool {
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
