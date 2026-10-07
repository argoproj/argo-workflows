package dag

import (
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
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

// TaskNodeID is the node ID of a task within a DAG/Steps boundary.
func TaskNodeID(wf *wfv1.Workflow, boundaryName, taskName string) string {
	return wf.ResolveNodeID(TaskNodeName(boundaryName, taskName))
}

// TaskNode returns the node of a task within a DAG/Steps boundary, or nil
// when the task has no node.
func TaskNode(wf *wfv1.Workflow, boundaryName, taskName string) *wfv1.NodeStatus {
	node, err := wf.Status.Nodes.Get(TaskNodeID(wf, boundaryName, taskName))
	if err != nil {
		return nil
	}
	return node
}

// getNode returns the raw node status for a task.
func (s *workflowStore) getNode(taskName string) *wfv1.NodeStatus {
	return TaskNode(s.workflow, s.boundaryName, taskName)
}

// TaskGroupItems returns the item nodes of tg, an expanded task's TaskGroup
// node: its children named as its items, "<group>(<index>:<item>)" (see
// expandedTaskName), other than a hook or a retry attempt. Its other
// children are not items: the lifecycle hooks an older controller hung on
// the group itself and, when the group has no items, the nodes that follow
// it (a DAG task's dependants, the next step group), which then hang off the
// group. missing reports whether a child is missing from nodes (pruned or
// offloaded): it may have been an item. It is the single rule for which of a
// TaskGroup's children are its items.
func TaskGroupItems(nodes wfv1.Nodes, tg *wfv1.NodeStatus) (items []*wfv1.NodeStatus, missing bool) {
	for _, childID := range tg.Children {
		child, err := nodes.Get(childID)
		if err != nil {
			missing = true
			continue
		}
		if !strings.HasPrefix(child.Name, tg.Name+"(") || (child.NodeFlag != nil && (child.NodeFlag.Hooked || child.NodeFlag.Retried)) {
			continue
		}
		items = append(items, child)
	}
	return items, missing
}

// getTaskGroupChildren returns the item nodes of a task's TaskGroup node
// (TaskGroupItems). Returns nil if the named task has no node, or its node
// isn't a TaskGroup.
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
	return TaskGroupItems(s.nodes, node)
}

// areHooksFulfilled checks if all lifecycle hooks for a task are fulfilled.
// An expanded task's exit hooks run per item and hang off its item nodes, so
// for a TaskGroup the item nodes' hooks are checked too.
func (s *workflowStore) areHooksFulfilled(taskName string) bool {
	node := s.getNode(taskName)
	if node == nil {
		return true
	}
	if !common.CheckAllHooksFullfilled(node, s.nodes) {
		return false
	}
	if node.Type == wfv1.NodeTypeTaskGroup {
		for _, child := range s.getTaskGroupChildren(taskName) {
			if !common.CheckAllHooksFullfilled(child, s.nodes) {
				return false
			}
		}
	}
	return true
}
