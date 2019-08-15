package controller

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/valyala/fasttemplate"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/templateresolution"
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

	// tmplCtx is the context of template search.
	tmplCtx *templateresolution.Context
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

// GetTaskNode returns the node status of a task.
func (d *dagContext) GetTaskNode(taskName string) *wfv1.NodeStatus {
	nodeID := d.taskNodeID(taskName)
	node, ok := d.wf.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return &node
}

// Assert all branch finished for failFast:disable function
func (d *dagContext) assertBranchFinished(targetTaskName string) bool {
	// We should ensure that from the bottom to the top,
	// all the nodes of this branch have at least one failure.
	// If successful, we should continue to run down until the leaf node
	taskNode := d.GetTaskNode(targetTaskName)
	if taskNode == nil {
		taskObject := d.getTask(targetTaskName)
		if taskObject != nil {
			// Make sure all the dependency node have one failed
			for _, tmpTaskName := range taskObject.Dependencies {
				// Recursive check until top root node
				return d.assertBranchFinished(tmpTaskName)
			}
		}
	} else if !taskNode.Successful() {
		return true
	}

	// In failFast situation, if node is successful, it will run to leaf node, above
	// the function, we have already check the leaf node status
	return false
}

// assessDAGPhase assesses the overall DAG status
func (d *dagContext) assessDAGPhase(targetTasks []string, nodes map[string]wfv1.NodeStatus) wfv1.NodePhase {
	// First check all our nodes to see if anything is still running. If so, then the DAG is
	// considered still running (even if there are failures). Remember any failures and if retry
	// nodes have been exhausted.
	var unsuccessfulPhase wfv1.NodePhase
	retriesExhausted := true
	for _, node := range nodes {
		if node.BoundaryID != d.boundaryID {
			continue
		}
		if !node.Completed() {
			return wfv1.NodeRunning
		}
		if node.Successful() {
			continue
		}
		// failed retry attempts should not factor into the overall unsuccessful phase of the dag
		// because the subsequent attempt may have succeeded
		if unsuccessfulPhase == "" && !isRetryAttempt(node, nodes) {
			unsuccessfulPhase = node.Phase
		}
		if node.Type == wfv1.NodeTypeRetry && d.hasMoreRetries(&node) {
			retriesExhausted = false
		}
	}

	if unsuccessfulPhase != "" {
		// If failFast set to false, we should return Running to continue this workflow for other DAG branch
		if d.tmpl.DAG.FailFast != nil && !*d.tmpl.DAG.FailFast {
			tmpOverAllFinished := true
			// If all the nodes have finished, we should mark the failed node to finish overall workflow
			// So we should check all the targetTasks branch have finished
			for _, tmpDepName := range targetTasks {
				tmpDepNode := d.GetTaskNode(tmpDepName)
				if tmpDepNode == nil {
					// If leaf node is nil, we should check it's parent node and recursive check
					if !d.assertBranchFinished(tmpDepName) {
						tmpOverAllFinished = false
					}
				} else if tmpDepNode.Type == wfv1.NodeTypeRetry && d.hasMoreRetries(tmpDepNode) {
					tmpOverAllFinished = false
					break
				}

				//If leaf node has finished, we should mark the error workflow
			}
			if !tmpOverAllFinished {
				return wfv1.NodeRunning
			}
		}

		// if we were unsuccessful, we can return *only* if all retry nodes have ben exhausted.
		if retriesExhausted {
			return unsuccessfulPhase
		}
	}
	// There are no currently running tasks. Now check if our dependencies were met
	for _, depName := range targetTasks {
		depNode := d.GetTaskNode(depName)
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

// isRetryAttempt detects if a node is part of a retry
func isRetryAttempt(node wfv1.NodeStatus, nodes map[string]wfv1.NodeStatus) bool {
	for _, potentialParent := range nodes {
		if potentialParent.Type == wfv1.NodeTypeRetry {
			for _, child := range potentialParent.Children {
				if child == node.ID {
					return true
				}
			}
		}
	}
	return false
}

func (d *dagContext) hasMoreRetries(node *wfv1.NodeStatus) bool {
	if node.Phase == wfv1.NodeSucceeded {
		return false
	}

	if len(node.Children) == 0 {
		return true
	}
	// pick the first child to determine it's template type
	childNode, ok := d.wf.Status.Nodes[node.Children[0]]
	if !ok {
		return false
	}
	tmpl := childNode.ResolvedTemplate
	if tmpl.RetryStrategy.Limit != nil && int32(len(node.Children)) > *tmpl.RetryStrategy.Limit {
		return false
	}
	return true
}

func (woc *wfOperationCtx) executeDAG(nodeName string, tmplCtx *templateresolution.Context, tmpl *wfv1.Template, orgTmpl wfv1.TemplateHolder, boundaryID string) *wfv1.NodeStatus {
	node := woc.getNodeByName(nodeName)
	if node != nil && node.Completed() {
		return node
	}
	if node == nil {
		node = woc.initializeNode(nodeName, wfv1.NodeTypeDAG, tmplCtx, tmpl, orgTmpl, boundaryID, wfv1.NodeRunning)
	} else if node.CanRerun() {
		node = woc.markNodePhase(nodeName, wfv1.NodeRunning)
	}

	defer func() {
		if node != nil && woc.wf.Status.Nodes[node.ID].Completed() {
			_ = woc.killDaemonedChildren(node.ID)
		}
	}()

	dagCtx := &dagContext{
		boundaryName: nodeName,
		boundaryID:   woc.wf.NodeID(nodeName),
		tasks:        tmpl.DAG.Tasks,
		visited:      make(map[string]bool),
		tmpl:         tmpl,
		wf:           woc.wf,
		tmplCtx:      tmplCtx,
	}

	// Identify our target tasks. If user did not specify any, then we choose all tasks which have
	// no dependants.
	var targetTasks []string
	if tmpl.DAG.Target == "" {
		targetTasks = findLeafTaskNames(tmpl.DAG.Tasks)
	} else {
		targetTasks = strings.Split(tmpl.DAG.Target, " ")
	}

	// kick off execution of each target task asynchronously
	for _, taskNames := range targetTasks {
		woc.executeDAGTask(dagCtx, taskNames)
	}
	// check if we are still running any tasks in this dag and return early if we do
	dagPhase := dagCtx.assessDAGPhase(targetTasks, woc.wf.Status.Nodes)
	switch dagPhase {
	case wfv1.NodeRunning:
		return woc.getNodeByName(nodeName)
	case wfv1.NodeError, wfv1.NodeFailed:
		return woc.markNodePhase(nodeName, dagPhase)
	}

	// set outputs from tasks in order for DAG templates to support outputs
	scope := wfScope{
		tmpl:  tmpl,
		scope: make(map[string]interface{}),
	}
	for _, task := range tmpl.DAG.Tasks {
		taskNode := dagCtx.GetTaskNode(task.Name)
		if taskNode == nil {
			// Can happen when dag.target was specified
			continue
		}
		woc.processNodeOutputs(&scope, fmt.Sprintf("tasks.%s", task.Name), taskNode)
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
		depNode := dagCtx.GetTaskNode(depName)
		if depNode == nil {
			woc.log.Println(depName)
		}
		outboundNodeIDs := woc.getOutboundNodes(depNode.ID)
		outbound = append(outbound, outboundNodeIDs...)
	}
	woc.log.Infof("Outbound nodes of %s set to %s", node.ID, outbound)
	node.OutboundNodes = outbound
	woc.wf.Status.Nodes[node.ID] = *node

	return woc.markNodePhase(nodeName, wfv1.NodeSucceeded)
}

// executeDAGTask traverses and executes the upward chain of dependencies of a task
func (woc *wfOperationCtx) executeDAGTask(dagCtx *dagContext, taskName string) {
	if _, ok := dagCtx.visited[taskName]; ok {
		return
	}
	dagCtx.visited[taskName] = true

	node := dagCtx.GetTaskNode(taskName)
	if node != nil && node.Completed() {
		return
	}
	// Check if our dependencies completed. If not, recurse our parents executing them if necessary
	task := dagCtx.getTask(taskName)
	dependenciesCompleted := true
	dependenciesSuccessful := true
	nodeName := dagCtx.taskNodeName(taskName)
	for _, depName := range task.Dependencies {
		depNode := dagCtx.GetTaskNode(depName)
		if depNode != nil {
			if depNode.Completed() {
				if !depNode.Successful() && !dagCtx.getTask(depName).ContinuesOn(depNode.Phase) {
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

	// All our dependencies were satisfied and successful. It's our turn to run

	taskGroupNode := woc.getNodeByName(nodeName)
	if taskGroupNode != nil && taskGroupNode.Type != wfv1.NodeTypeTaskGroup {
		taskGroupNode = nil
	}
	// connectDependencies is a helper to connect our dependencies to current task as children
	connectDependencies := func(taskNodeName string) {
		if len(task.Dependencies) == 0 || taskGroupNode != nil {
			// if we had no dependencies, then we are a root task, and we should connect the
			// boundary node as our parent
			if taskGroupNode == nil {
				woc.addChildNode(dagCtx.boundaryName, taskNodeName)
			} else {
				woc.addChildNode(taskGroupNode.Name, taskNodeName)
			}

		} else {
			// Otherwise, add all outbound nodes of our dependencies as parents to this node
			for _, depName := range task.Dependencies {
				depNode := dagCtx.GetTaskNode(depName)
				outboundNodeIDs := woc.getOutboundNodes(depNode.ID)
				woc.log.Infof("DAG outbound nodes of %s are %s", depNode, outboundNodeIDs)
				for _, outNodeID := range outboundNodeIDs {
					woc.addChildNode(woc.wf.Status.Nodes[outNodeID].Name, taskNodeName)
				}
			}
		}
	}

	// First resolve/substitute params/artifacts from our dependencies
	newTask, err := woc.resolveDependencyReferences(dagCtx, task)
	if err != nil {
		woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, nil, nil, task, dagCtx.boundaryID, wfv1.NodeError, err.Error())
		connectDependencies(nodeName)
		return
	}

	// Next, expand the DAG's withItems/withParams/withSequence (if any). If there was none, then
	// expandedTasks will be a single element list of the same task
	expandedTasks, err := woc.expandTask(*newTask)
	if err != nil {
		woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, nil, nil, task, dagCtx.boundaryID, wfv1.NodeError, err.Error())
		connectDependencies(nodeName)
		return
	}

	// If DAG task has withParam of with withSequence then we need to create virtual node of type TaskGroup.
	// For example, if we had task A with withItems of ['foo', 'bar'] which expanded to ['A(0:foo)', 'A(1:bar)'], we still
	// need to create a node for A.
	if len(task.WithItems) > 0 || task.WithParam != "" || task.WithSequence != nil {
		if taskGroupNode == nil {
			connectDependencies(nodeName)
			taskGroupNode = woc.initializeNode(nodeName, wfv1.NodeTypeTaskGroup, nil, nil, task, dagCtx.boundaryID, wfv1.NodeRunning, "")
		}
	}

	for _, t := range expandedTasks {
		node = dagCtx.GetTaskNode(t.Name)
		taskNodeName := dagCtx.taskNodeName(t.Name)
		if node == nil {
			woc.log.Infof("All of node %s dependencies %s completed", taskNodeName, task.Dependencies)
			// Add the child relationship from our dependency's outbound nodes to this node.
			connectDependencies(taskNodeName)

			// Check the task's when clause to decide if it should execute
			proceed, err := shouldExecute(t.When)
			if err != nil {
				woc.initializeNode(taskNodeName, wfv1.NodeTypeSkipped, nil, nil, task, dagCtx.boundaryID, wfv1.NodeError, err.Error())
				continue
			}
			if !proceed {
				skipReason := fmt.Sprintf("when '%s' evaluated false", t.When)
				woc.initializeNode(taskNodeName, wfv1.NodeTypeSkipped, nil, nil, task, dagCtx.boundaryID, wfv1.NodeSkipped, skipReason)
				continue
			}
		}

		// Finally execute the template
		_, _ = woc.executeTemplate(taskNodeName, &t, dagCtx.tmplCtx, t.Arguments, dagCtx.boundaryID)
	}

	if taskGroupNode != nil {
		groupPhase := wfv1.NodeSucceeded
		for _, t := range expandedTasks {
			// Add the child relationship from our dependency's outbound nodes to this node.
			node := dagCtx.GetTaskNode(t.Name)
			if node == nil || !node.Completed() {
				return
			}
			if !node.Successful() {
				groupPhase = node.Phase
			}
		}
		woc.markNodePhase(taskGroupNode.Name, groupPhase)
	}
}

// resolveDependencyReferences replaces any references to outputs of task dependencies, or artifacts in the inputs
// NOTE: by now, input parameters should have been substituted throughout the template
func (woc *wfOperationCtx) resolveDependencyReferences(dagCtx *dagContext, task *wfv1.DAGTask) (*wfv1.DAGTask, error) {
	// build up the scope
	scope := wfScope{
		tmpl:  dagCtx.tmpl,
		scope: make(map[string]interface{}),
	}
	woc.addOutputsToScope("workflow", woc.wf.Status.Outputs, &scope)

	ancestors := common.GetTaskAncestry(dagCtx, task.Name, dagCtx.tasks)
	for _, ancestor := range ancestors {
		ancestorNode := dagCtx.GetTaskNode(ancestor)
		prefix := fmt.Sprintf("tasks.%s", ancestor)
		if ancestorNode.Type == wfv1.NodeTypeTaskGroup {
			var ancestorNodes []wfv1.NodeStatus
			for _, node := range woc.wf.Status.Nodes {
				if node.BoundaryID == dagCtx.boundaryID && strings.HasPrefix(node.Name, ancestorNode.Name+"(") {
					ancestorNodes = append(ancestorNodes, node)
				}
			}
			err := woc.processAggregateNodeOutputs(ancestorNode.ResolvedTemplate, &scope, prefix, ancestorNodes)
			if err != nil {
				return nil, errors.InternalWrapError(err)
			}
		} else {
			woc.processNodeOutputs(&scope, prefix, ancestorNode)
		}
	}

	// Perform replacement
	// Replace woc.volumes
	err := woc.substituteParamsInVolumes(scope.replaceMap())
	if err != nil {
		return nil, err
	}

	// Replace task's parameters
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

// expandTask expands a single DAG task containing withItems, withParams, withSequence into multiple parallel tasks
func (woc *wfOperationCtx) expandTask(task wfv1.DAGTask) ([]wfv1.DAGTask, error) {
	taskBytes, err := json.Marshal(task)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	var items []wfv1.Item
	if len(task.WithItems) > 0 {
		items = task.WithItems
	} else if task.WithParam != "" {
		err = json.Unmarshal([]byte(task.WithParam), &items)
		if err != nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "withParam value could not be parsed as a JSON list: %s", strings.TrimSpace(task.WithParam))
		}
	} else if task.WithSequence != nil {
		items, err = expandSequence(task.WithSequence)
		if err != nil {
			return nil, err
		}
	} else {
		return []wfv1.DAGTask{task}, nil
	}

	fstTmpl := fasttemplate.New(string(taskBytes), "{{", "}}")
	expandedTasks := make([]wfv1.DAGTask, 0)
	for i, item := range items {
		var newTask wfv1.DAGTask
		newTaskName, err := processItem(fstTmpl, task.Name, i, item, &newTask)
		if err != nil {
			return nil, err
		}
		newTask.Name = newTaskName
		newTask.Template = task.Template
		expandedTasks = append(expandedTasks, newTask)
	}
	return expandedTasks, nil
}
