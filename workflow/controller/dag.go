package controller

import (
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
)

// dagContext holds context information about this context's DAG
type dagContext struct {
	// encompasser is the node name of the encompassing node to this DAG.
	// This is used to incorporate into each of the task's node names.
	encompasser string

	// tasks are all the tasks in the template
	tasks []wfv1.DAGTask

	wf *wfv1.Workflow
}

func (d *dagContext) getTask(taskName string) *wfv1.DAGTask {
	for _, task := range d.tasks {
		if task.Name == taskName {
			return &task
		}
	}
	panic("target " + taskName + " does not exist")
}

// taskNodeName formulates the nodeName for a dag task
func (d *dagContext) taskNodeName(taskName string) string {
	return fmt.Sprintf("%s.%s", d.encompasser, taskName)
}

// taskNodeID formulates the node ID for a dag task
func (d *dagContext) taskNodeID(taskName string) string {
	nodeName := d.taskNodeName(taskName)
	return d.wf.NodeID(nodeName)
}

func (d *dagContext) getTaskNode(taskName string) *wfv1.NodeStatus {
	nodeID := d.taskNodeID(taskName)
	node, ok := d.wf.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return &node
}

func (woc *wfOperationCtx) executeDAG(nodeName string, tmpl *wfv1.Template) *wfv1.NodeStatus {
	nodeID := woc.wf.NodeID(nodeName)
	node, nodeInitialized := woc.wf.Status.Nodes[nodeID]
	if nodeInitialized && node.Completed() {
		return &node
	}
	dagCtx := &dagContext{
		encompasser: nodeName,
		tasks:       tmpl.DAG.Tasks,
		wf:          woc.wf,
	}
	var targetTasks []string
	if tmpl.DAG.Targets == "" {
		targetTasks = findLeafTaskNames(tmpl.DAG.Tasks)
	} else {
		targetTasks = strings.Split(tmpl.DAG.Targets, " ")
	}
	if !nodeInitialized {
		node = *woc.initializeNodes(dagCtx, targetTasks)
	}
	// kick off execution of each target task asynchronously
	for _, taskNames := range targetTasks {
		woc.executeDAGTask(dagCtx, taskNames)
	}
	// return early if we have yet to complete execution of any one of our dependencies
	for _, depName := range targetTasks {
		depNode := dagCtx.getTaskNode(depName)
		if !depNode.Completed() {
			return &node
		}
	}
	// all desired tasks completed. now it is time to assess state
	for _, depName := range targetTasks {
		depNode := dagCtx.getTaskNode(depName)
		if !depNode.Successful() {
			// TODO: need to create virtual fan-in and fan-out node and mark them accordingly
			// For now we use the dag node
			return woc.markNodePhase(nodeName, depNode.Phase)
		}
	}
	return woc.markNodePhase(nodeName, wfv1.NodeSucceeded)
}

// initializeNodes performs a one time initialization of the DAG nodes and edges
func (woc *wfOperationCtx) initializeNodes(dagCtx *dagContext, targetTasks []string) *wfv1.NodeStatus {
	node := woc.markNodePhase(dagCtx.encompasser, wfv1.NodeRunning)
	woc.log.Infof("Initializing nodes for tasks %s", targetTasks)
	// rootTaskNames contains all the taskNames of "root" nodes of the DAG
	// (tasks with zero dependencies). These tasks are considered the
	// immediate children of the DAG entrypoint
	rootTaskNames := make(map[string]bool)
	for _, taskName := range targetTasks {
		woc.initializeTask(dagCtx, taskName, rootTaskNames)
	}
	for rootTaskName := range rootTaskNames {
		woc.addChildNode(dagCtx.encompasser, dagCtx.taskNodeName(rootTaskName))
	}
	return node
}

// initializeTask creates the initial node entry for a task and its dependencies
func (woc *wfOperationCtx) initializeTask(dagCtx *dagContext, taskName string, roots map[string]bool) {
	nodeName := dagCtx.taskNodeName(taskName)
	nodeID := woc.wf.NodeID(nodeName)
	_, ok := woc.wf.Status.Nodes[nodeID]
	if ok {
		return
	}

	_ = woc.markNodePhase(nodeName, wfv1.NodePending)
	task := dagCtx.getTask(taskName)
	for _, depName := range task.Dependencies {
		woc.initializeTask(dagCtx, depName, roots)
		woc.addChildNode(dagCtx.taskNodeName(depName), nodeName)
	}
	if len(task.Dependencies) == 0 {
		roots[taskName] = true
	}
}

// executeDAGTask traverses and executes the upward chain of dependencies of a task
func (woc *wfOperationCtx) executeDAGTask(dagCtx *dagContext, taskName string) {
	node := dagCtx.getTaskNode(taskName)
	if node.Completed() {
		return
	}
	// Check if our dependencies completed. If not, recurse our parents
	// executing them if necessary
	task := dagCtx.getTask(taskName)
	dependenciesCompleted := true
	for _, depName := range task.Dependencies {
		depNode := dagCtx.getTaskNode(depName)
		if depNode.Completed() {
			if !depNode.Successful() {
				woc.log.Infof("Target %s marked %s due to dependency %s %s", taskName, wfv1.NodeSkipped, depNode.Name, depNode.Phase)
				woc.markNodePhase(node.Name, wfv1.NodeSkipped)
				return
			}
			continue
		}
		dependenciesCompleted = false
		// recurse our dependency
		woc.executeDAGTask(dagCtx, depName)
	}
	if !dependenciesCompleted {
		return
	}

	// All our dependencies were satisifed and successful. It's our turn to run
	if node.Phase == wfv1.NodePending {
		woc.log.Infof("All of node %s dependencies %s satisfied", node, task.Dependencies)
	}
	woc.executeTemplate(task.Template, task.Arguments, node.Name)
}

// findLeafTaskNames finds the names of all tasks whom no other nodes depend on.
// When targets is omitted, the return value is used as the the default list of targets.
func findLeafTaskNames(tasks []wfv1.DAGTask) []string {
	taskIsLeaf := make(map[string]bool)
	for _, task := range tasks {
		if _, ok := taskIsLeaf[task.Name]; !ok {
			taskIsLeaf[task.Name] = true
		}
		for _, dependency := range task.Dependencies {
			taskIsLeaf[dependency] = false
		}
	}
	leafTaskNames := make([]string, 0)
	for taskName, isLeaf := range taskIsLeaf {
		if isLeaf {
			leafTaskNames = append(leafTaskNames, taskName)
		}
	}
	return leafTaskNames
}
