package controller

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/valyala/fasttemplate"
)

// dagContext holds context information about this context's DAG
type dagContext struct {
	// boundaryName is the node name of the boundary node to this DAG.
	// This is used to incorporate into each of the task's node names.
	boundaryName string
	boundaryID   string

	// tasks are all the tasks in the template
	tasks []wfv1.DAGTask

	// visited keeps track of tasks we have already visited during an invocation of executeDAG
	// in order to avoid duplicating work
	visited map[string]bool

	// tmpl is the template spec. it is needed to resolve hard-wired artifacts
	tmpl *wfv1.Template

	// wf is stored to formulate nodeIDs
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
	return fmt.Sprintf("%s.%s", d.boundaryName, taskName)
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

// assessDAGPhase assesses the overall DAG status
func (d *dagContext) assessDAGPhase(targetTasks []string, nodes map[string]wfv1.NodeStatus) wfv1.NodePhase {
	var phase wfv1.NodePhase
	// First check all our nodes to see if any thing is still running. If any thing is still running
	// then the DAG is considered still running (even if there are failures). Remember any failures
	for _, node := range nodes {
		if node.BoundaryID != d.boundaryID {
			continue
		}
		if !node.Completed() {
			return wfv1.NodeRunning
		}
		if !node.Successful() && phase == "" {
			phase = node.Phase
		}
	}
	// If we get here, then there are no running tasks. Propagate the failure/error if one was found
	if phase != "" {
		return phase
	}
	// There are no currently running tasks. Now check if our dependencies were met
	for _, depName := range targetTasks {
		depNode := d.getTaskNode(depName)
		if depNode == nil {
			return wfv1.NodeRunning
		}
		if !depNode.Successful() {
			// we should theoretically never get here since it would have been caught in first loop
			return depNode.Phase
		}
	}
	// If we get here, all our dependencies were completed and successful
	return wfv1.NodeSucceeded
}

func (woc *wfOperationCtx) executeDAG(nodeName string, tmpl *wfv1.Template, boundaryID string) *wfv1.NodeStatus {
	node := woc.getNodeByName(nodeName)
	if node != nil && node.Completed() {
		return node
	}
	dagCtx := &dagContext{
		boundaryName: nodeName,
		boundaryID:   woc.wf.NodeID(nodeName),
		tasks:        tmpl.DAG.Tasks,
		visited:      make(map[string]bool),
		tmpl:         tmpl,
		wf:           woc.wf,
	}
	var targetTasks []string
	if tmpl.DAG.Targets == "" {
		targetTasks = findLeafTaskNames(tmpl.DAG.Tasks)
	} else {
		targetTasks = strings.Split(tmpl.DAG.Targets, " ")
	}

	if node == nil {
		node = woc.initializeNode(nodeName, wfv1.NodeTypeDAG, tmpl.Name, boundaryID, wfv1.NodeRunning)
		rootTasks := findRootTaskNames(dagCtx, targetTasks)
		woc.log.Infof("Root tasks of %s identified as %s", nodeName, rootTasks)
		for _, rootTaskName := range rootTasks {
			woc.addChildNode(node.Name, dagCtx.taskNodeName(rootTaskName))
		}
	}
	// kick off execution of each target task asynchronously
	for _, taskNames := range targetTasks {
		woc.executeDAGTask(dagCtx, taskNames)
	}
	// check if we are still running any tasks in this dag and return early if we do
	dagPhase := dagCtx.assessDAGPhase(targetTasks, woc.wf.Status.Nodes)
	switch dagPhase {
	case wfv1.NodeRunning:
		return node
	case wfv1.NodeError, wfv1.NodeFailed:
		return woc.markNodePhase(nodeName, dagPhase)
	}

	// set outputs from tasks in order for DAG templates to support outputs
	scope := wfScope{
		tmpl:  tmpl,
		scope: make(map[string]interface{}),
	}
	for _, task := range tmpl.DAG.Tasks {
		scope.addNodeOutputsToScope(fmt.Sprintf("tasks.%s", task.Name), dagCtx.getTaskNode(task.Name))
	}
	outputs, err := getTemplateOutputsFromScope(tmpl, &scope)
	if err != nil {
		return woc.markNodeError(nodeName, err)
	}
	if outputs != nil {
		node = woc.getNodeByName(nodeName)
		node.Outputs = outputs
		woc.wf.Status.Nodes[node.ID] = *node
	}

	// set the outbound nodes from the target tasks
	node = woc.getNodeByName(nodeName)
	outbound := make([]string, 0)
	for _, depName := range targetTasks {
		depNode := dagCtx.getTaskNode(depName)
		if depNode == nil {
			woc.log.Println(depName)
		}
		outboundNodeIDs := woc.getOutboundNodes(depNode.ID)
		for _, outNodeID := range outboundNodeIDs {
			outbound = append(outbound, outNodeID)
		}
	}
	woc.log.Infof("Outbound nodes of %s set to %s", node.ID, outbound)
	node.OutboundNodes = outbound
	woc.wf.Status.Nodes[node.ID] = *node

	return woc.markNodePhase(nodeName, wfv1.NodeSucceeded)
}

// findRootTaskNames finds the names of all tasks which have no dependencies.
// Once identified, these root tasks are marked as children to the encompassing node.
func findRootTaskNames(dagCtx *dagContext, targetTasks []string) []string {
	//rootTaskNames := make(map[string]bool)
	rootTaskNames := make([]string, 0)
	visited := make(map[string]bool)
	var findRootHelper func(s string)
	findRootHelper = func(taskName string) {
		if _, ok := visited[taskName]; ok {
			return
		}
		visited[taskName] = true
		task := dagCtx.getTask(taskName)
		if len(task.Dependencies) == 0 {
			rootTaskNames = append(rootTaskNames, taskName)
			return
		}
		for _, depName := range task.Dependencies {
			findRootHelper(depName)
		}
	}
	for _, targetTaskName := range targetTasks {
		findRootHelper(targetTaskName)
	}
	return rootTaskNames
}

// executeDAGTask traverses and executes the upward chain of dependencies of a task
func (woc *wfOperationCtx) executeDAGTask(dagCtx *dagContext, taskName string) {
	if _, ok := dagCtx.visited[taskName]; ok {
		return
	}
	dagCtx.visited[taskName] = true

	node := dagCtx.getTaskNode(taskName)
	if node != nil && node.Completed() {
		return
	}
	// Check if our dependencies completed. If not, recurse our parents executing them if necessary
	task := dagCtx.getTask(taskName)
	dependenciesCompleted := true
	dependenciesSuccessful := true
	nodeName := dagCtx.taskNodeName(taskName)
	for _, depName := range task.Dependencies {
		depNode := dagCtx.getTaskNode(depName)
		if depNode != nil {
			if depNode.Completed() {
				if !depNode.Successful() {
					dependenciesSuccessful = false
				}
				continue
			}
		}
		dependenciesCompleted = false
		dependenciesSuccessful = false
		// recurse our dependency
		woc.executeDAGTask(dagCtx, depName)
	}
	if !dependenciesCompleted {
		return
	}

	if !dependenciesSuccessful {
		// TODO: in the future we may support some more sophisticated syntax for deciding on how
		// to proceed if at least one dependency succeeded, analogous to airflow's trigger rules,
		// (e.g. one_success, all_done, one_failed, etc...). This decision would be made here.
		return
	}

	// All our dependencies completed. Now add the child relationship from our dependency's
	// outbound nodes to this node.
	node = dagCtx.getTaskNode(taskName)
	if node == nil {
		woc.log.Infof("All of node %s dependencies %s completed", nodeName, task.Dependencies)
		// Add all outbound nodes of our dependencies as parents to this node
		for _, depName := range task.Dependencies {
			depNode := dagCtx.getTaskNode(depName)
			woc.log.Infof("node %s outbound nodes: %s", depNode, depNode.OutboundNodes)
			if depNode.Type == wfv1.NodeTypePod {
				woc.addChildNode(depNode.Name, nodeName)
			} else {
				for _, outNodeID := range depNode.OutboundNodes {
					woc.addChildNode(woc.wf.Status.Nodes[outNodeID].Name, nodeName)
				}
			}
		}
	}

	// All our dependencies were satisfied and successful. It's our turn to run
	// Substitute params/artifacts from our dependencies and execute the template
	newTask, err := woc.resolveDependencyReferences(dagCtx, task)
	if err != nil {
		woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, task.Template, dagCtx.boundaryID, wfv1.NodeError, err.Error())
		return
	}
	_, _ = woc.executeTemplate(newTask.Template, newTask.Arguments, nodeName, dagCtx.boundaryID)
}

// resolveDependencyReferences replaces any references to outputs of task dependencies, or artifacts in the inputs
// NOTE: by now, input parameters should have been substituted throughout the template
func (woc *wfOperationCtx) resolveDependencyReferences(dagCtx *dagContext, task *wfv1.DAGTask) (*wfv1.DAGTask, error) {
	// build up the scope
	scope := wfScope{
		tmpl:  dagCtx.tmpl,
		scope: make(map[string]interface{}),
	}
	ancestors := common.GetTaskAncestry(task.Name, dagCtx.tasks)
	for _, ancestor := range ancestors {
		ancestorNode := dagCtx.getTaskNode(ancestor)
		prefix := fmt.Sprintf("tasks.%s", ancestor)
		scope.addNodeOutputsToScope(prefix, ancestorNode)
	}

	// Perform replacement
	taskBytes, err := json.Marshal(task)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	fstTmpl := fasttemplate.New(string(taskBytes), "{{", "}}")
	newTaskStr, err := common.Replace(fstTmpl, scope.replaceMap(), true)
	if err != nil {
		return nil, err
	}
	var newTask wfv1.DAGTask
	err = json.Unmarshal([]byte(newTaskStr), &newTask)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}

	// replace all artifact references
	for j, art := range newTask.Arguments.Artifacts {
		if art.From == "" {
			continue
		}
		resolvedArt, err := scope.resolveArtifact(art.From)
		if err != nil {
			return nil, err
		}
		resolvedArt.Name = art.Name
		newTask.Arguments.Artifacts[j] = *resolvedArt
	}
	return &newTask, nil
}

// findLeafTaskNames finds the names of all tasks whom no other nodes depend on.
// This list of tasks is used as the the default list of targets when dag.targets is omitted.
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
