package controller

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/antonmedv/expr"
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

	// onExitTemplate is a flag denoting this template as part of an onExit handler. This is necessary to ensure that
	// further nodes stemming from this template are allowed to run when using "ShutdownStrategy: Stop"
	onExitTemplate bool

	// dependencies is a list of all the tasks a specific task depends on. Because dependencies are computed using regex
	// and regex is expensive, we cache the results so that they are only computed once per operation
	dependencies map[string][]string

	// dependsLogic is the resolved "depends" string of a particular task. A resolved "depends" simply contains
	// task with their explicit results since we allow them to be omitted for convinience
	// (i.e., "A || B.Completed" -> "(A.Succeeded || A.Skipped || A.Daemoned) || B.Completed").
	// Because this resolved "depends" is computed using regex and regex is expensive, we cache the results so that they
	// are only computed once per operation
	dependsLogic map[string]string
}

func (d *dagContext) GetTaskDependencies(taskName string) []string {
	if dependencies, ok := d.dependencies[taskName]; ok {
		return dependencies
	}
	d.resolveDependencies(taskName)
	return d.dependencies[taskName]
}

func (d *dagContext) GetTaskFinishedAtTime(taskName string) time.Time {
	node := d.getTaskNode(taskName)
	if !node.FinishedAt.IsZero() {
		return node.FinishedAt.Time
	}
	return node.StartedAt.Time
}

func (d *dagContext) GetTask(taskName string) *wfv1.DAGTask {
	for _, task := range d.tasks {
		if task.Name == taskName {
			return &task
		}
	}
	panic("target " + taskName + " does not exist")
}

func (d *dagContext) GetTaskDependsLogic(taskName string) string {
	if logic, ok := d.dependsLogic[taskName]; ok {
		return logic
	}
	d.resolveDependencies(taskName)
	return d.dependsLogic[taskName]
}

func (d *dagContext) resolveDependencies(taskName string) {
	dependencies, resolvedDependsLogic := common.GetTaskDependencies(d.GetTask(taskName), d)
	d.dependencies[taskName] = dependencies
	d.dependsLogic[taskName] = resolvedDependsLogic
}

// taskNodeName formulates the nodeName for a dag task
func (d *dagContext) taskNodeName(taskName string) string {
	return fmt.Sprintf("%s.%s", d.boundaryName, taskName)
}

// nodeTaskName formulates the corresponding task name for a dag node. Note that this is not simply the inverse of
// taskNodeName. A task name might be from an expanded task, in which case it will not have an explicit task defined for it.
// When that is the case, we formulate the name of the original expanded task by removing the fields after "("
func (d *dagContext) taskNameFromNodeName(nodeName string) string {
	nodeName = strings.TrimPrefix(nodeName, fmt.Sprintf("%s.", d.boundaryName))
	// Check if this nodeName comes from an expanded task. If it does, return the original parent task
	if index := strings.Index(nodeName, "("); index != -1 {
		nodeName = nodeName[:index]
	}
	return nodeName
}

func (d *dagContext) getTaskFromNode(node *wfv1.NodeStatus) *wfv1.DAGTask {
	return d.GetTask(d.taskNameFromNodeName(node.Name))
}

// taskNodeID formulates the node ID for a dag task
func (d *dagContext) taskNodeID(taskName string) string {
	nodeName := d.taskNodeName(taskName)
	return d.wf.NodeID(nodeName)
}

// getTaskNode returns the node status of a task.
func (d *dagContext) getTaskNode(taskName string) *wfv1.NodeStatus {
	nodeID := d.taskNodeID(taskName)
	node, ok := d.wf.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return &node
}

// Assert all branch finished for failFast:disable function
func (d *dagContext) assertBranchFinished(targetTaskNames []string) bool {
	// We should ensure that from the bottom to the top,
	// all the nodes of this branch have at least one failure.
	// If successful, we should continue to run down until the leaf node
	flag := false
	for _, targetTaskName := range targetTaskNames {
		taskNode := d.getTaskNode(targetTaskName)
		if taskNode == nil {
			taskObject := d.GetTask(targetTaskName)
			if taskObject != nil {
				// Make sure all the dependency node have one failed
				// Recursive check until top root node
				return d.assertBranchFinished(d.GetTaskDependencies(taskObject.Name))
			}
		} else if !taskNode.Successful() {
			taskObject := d.GetTask(targetTaskName)
			if !taskObject.ContinuesOn(taskNode.Phase) {
				flag = true
			}
		}

		// In failFast situation, if node is successful, it will run to leaf node, above
		// the function, we have already check the leaf node status
	}
	return flag
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
		// Failed retry attempts should not factor into the overall unsuccessful phase of the dag
		// because the subsequent attempt may have succeeded
		// Furthermore, if the node failed but ContinuesOn its phase, it should also not factor into the overall phase of the dag
		if unsuccessfulPhase == "" && !(isRetryAttempt(node, nodes) || d.getTaskFromNode(&node).ContinuesOn(node.Phase)) {
			unsuccessfulPhase = node.Phase
		}
		// If the node is a Retry node and has more retry attempts and is not shutting down, do not fail the task as a whole
		// and allow the remaining retries to be executed
		if node.Type == wfv1.NodeTypeRetry && d.hasMoreRetries(&node) && d.wf.Spec.Shutdown.ShouldExecute(d.onExitTemplate) {
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
				tmpDepNode := d.getTaskNode(tmpDepName)
				if tmpDepNode == nil {
					// If leaf node is nil, we should check it's parent node and recursive check
					if !d.assertBranchFinished([]string{tmpDepName}) {
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
		depNode := d.getTaskNode(depName)
		depTask := d.GetTask(depName)
		if depNode == nil {
			return wfv1.NodeRunning
		}
		if !depNode.Successful() && !depTask.ContinuesOn(depNode.Phase) {
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
	_, tmpl, _, err := d.tmplCtx.ResolveTemplate(&childNode)
	if err != nil {
		return false
	}
	if tmpl.RetryStrategy != nil && tmpl.RetryStrategy.Limit != nil && int32(len(node.Children)) > *tmpl.RetryStrategy.Limit {
		return false
	}
	return true
}

func (woc *wfOperationCtx) executeDAG(nodeName string, tmplCtx *templateresolution.Context, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node := woc.getNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypeDAG, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodeRunning)
	}

	defer func() {
		if woc.wf.Status.Nodes[node.ID].Completed() {
			_ = woc.killDaemonedChildren(node.ID)
		}
	}()

	dagCtx := &dagContext{
		boundaryName:   nodeName,
		boundaryID:     node.ID,
		tasks:          tmpl.DAG.Tasks,
		visited:        make(map[string]bool),
		tmpl:           tmpl,
		wf:             woc.wf,
		tmplCtx:        tmplCtx,
		onExitTemplate: opts.onExitTemplate,
		dependencies:   make(map[string][]string),
		dependsLogic:   make(map[string]string),
	}

	// Identify our target tasks. If user did not specify any, then we choose all tasks which have
	// no dependants.
	var targetTasks []string
	if tmpl.DAG.Target == "" {
		targetTasks = dagCtx.findLeafTaskNames(tmpl.DAG.Tasks)
	} else {
		targetTasks = strings.Split(tmpl.DAG.Target, " ")
	}

	// kick off execution of each target task asynchronously
	for _, taskName := range targetTasks {
		woc.executeDAGTask(dagCtx, taskName)
	}

	// check if we are still running any tasks in this dag and return early if we do
	dagPhase := dagCtx.assessDAGPhase(targetTasks, woc.wf.Status.Nodes)
	switch dagPhase {
	case wfv1.NodeRunning:
		return node, nil
	case wfv1.NodeError, wfv1.NodeFailed:
		woc.updateOutboundNodesForTargetTasks(dagCtx, targetTasks, nodeName)
		_ = woc.markNodePhase(nodeName, dagPhase)
		return node, nil
	}

	// set outputs from tasks in order for DAG templates to support outputs
	scope := wfScope{
		tmpl:  tmpl,
		scope: make(map[string]interface{}),
	}
	for _, task := range tmpl.DAG.Tasks {
		taskNode := dagCtx.getTaskNode(task.Name)
		if taskNode == nil {
			// Can happen when dag.target was specified
			continue
		}
		woc.buildLocalScope(&scope, fmt.Sprintf("tasks.%s", task.Name), taskNode)
		woc.addOutputsToGlobalScope(taskNode.Outputs)
	}
	outputs, err := getTemplateOutputsFromScope(tmpl, &scope)
	if err != nil {
		return node, err
	}
	if outputs != nil {
		node = woc.getNodeByName(nodeName)
		node.Outputs = outputs
		woc.wf.Status.Nodes[node.ID] = *node
	}

	woc.updateOutboundNodesForTargetTasks(dagCtx, targetTasks, nodeName)

	return woc.markNodePhase(nodeName, wfv1.NodeSucceeded), nil
}

func (woc *wfOperationCtx) updateOutboundNodesForTargetTasks(dagCtx *dagContext, targetTasks []string, nodeName string) {
	// set the outbound nodes from the target tasks
	outbound := make([]string, 0)
	for _, depName := range targetTasks {
		depNode := dagCtx.getTaskNode(depName)
		if depNode == nil {
			woc.log.Println(depName)
			continue
		}
		outboundNodeIDs := woc.getOutboundNodes(depNode.ID)
		outbound = append(outbound, outboundNodeIDs...)
	}
	node := woc.getNodeByName(nodeName)
	node.OutboundNodes = outbound
	woc.wf.Status.Nodes[node.ID] = *node
	woc.log.Infof("Outbound nodes of %s set to %s", node.ID, outbound)
}

// executeDAGTask traverses and executes the upward chain of dependencies of a task
func (woc *wfOperationCtx) executeDAGTask(dagCtx *dagContext, taskName string) {
	if _, ok := dagCtx.visited[taskName]; ok {
		return
	}
	dagCtx.visited[taskName] = true

	node := dagCtx.getTaskNode(taskName)
	task := dagCtx.GetTask(taskName)
	if node != nil && node.Completed() {
		// Run the node's onExit node, if any.
		hasOnExitNode, onExitNode, err := woc.runOnExitNode(task.Name, task.OnExit, dagCtx.boundaryID, dagCtx.tmplCtx)
		if hasOnExitNode && (onExitNode == nil || !onExitNode.Completed() || err != nil) {
			// The onExit node is either not complete or has errored out, return.
			return
		}
		return
	}

	// The template scope of this dag.
	dagTemplateScope := dagCtx.tmplCtx.GetTemplateScope()

	// Check if our dependencies completed. If not, recurse our parents executing them if necessary
	nodeName := dagCtx.taskNodeName(taskName)
	taskDependencies := dagCtx.GetTaskDependencies(taskName)

	taskGroupNode := woc.getNodeByName(nodeName)
	if taskGroupNode != nil && taskGroupNode.Type != wfv1.NodeTypeTaskGroup {
		taskGroupNode = nil
	}
	// connectDependencies is a helper to connect our dependencies to current task as children
	connectDependencies := func(taskNodeName string) {
		if len(taskDependencies) == 0 || taskGroupNode != nil {
			// if we had no dependencies, then we are a root task, and we should connect the
			// boundary node as our parent
			if taskGroupNode == nil {
				woc.addChildNode(dagCtx.boundaryName, taskNodeName)
			} else {
				woc.addChildNode(taskGroupNode.Name, taskNodeName)
			}

		} else {
			// Otherwise, add all outbound nodes of our dependencies as parents to this node
			for _, depName := range taskDependencies {
				depNode := dagCtx.getTaskNode(depName)
				outboundNodeIDs := woc.getOutboundNodes(depNode.ID)
				for _, outNodeID := range outboundNodeIDs {
					woc.addChildNode(woc.wf.Status.Nodes[outNodeID].Name, taskNodeName)
				}
			}
		}
	}

	if dagCtx.GetTaskDependsLogic(taskName) != "" {
		execute, proceed, err := dagCtx.evaluateDependsLogic(taskName)
		if err != nil {
			woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeError, err.Error())
			connectDependencies(nodeName)
			return
		}
		if !proceed {
			// This node's dependencies are not completed yet, recurse into them, then return
			for _, dep := range taskDependencies {
				woc.executeDAGTask(dagCtx, dep)
			}
			return
		}
		if !execute {
			// Given the results of this node's dependencies, this node should not be executed. Mark it skipped
			woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeSkipped, "depends condition not met")
			connectDependencies(nodeName)
			return
		}
	}

	// All our dependencies were satisfied and successful. It's our turn to run
	// First resolve/substitute params/artifacts from our dependencies
	newTask, err := woc.resolveDependencyReferences(dagCtx, task)
	if err != nil {
		woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeError, err.Error())
		connectDependencies(nodeName)
		return
	}

	// Next, expand the DAG's withItems/withParams/withSequence (if any). If there was none, then
	// expandedTasks will be a single element list of the same task
	expandedTasks, err := expandTask(*newTask)
	if err != nil {
		woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeError, err.Error())
		connectDependencies(nodeName)
		return
	}

	// If DAG task has withParam of with withSequence then we need to create virtual node of type TaskGroup.
	// For example, if we had task A with withItems of ['foo', 'bar'] which expanded to ['A(0:foo)', 'A(1:bar)'], we still
	// need to create a node for A.
	if task.ShouldExpand() {
		if taskGroupNode == nil {
			connectDependencies(nodeName)
			taskGroupNode = woc.initializeNode(nodeName, wfv1.NodeTypeTaskGroup, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeRunning, "")
		}
	}

	for _, t := range expandedTasks {
		taskNodeName := dagCtx.taskNodeName(t.Name)
		// Ensure that the generated taskNodeName can be reversed into the original (not expanded) task name
		if dagCtx.taskNameFromNodeName(taskNodeName) != task.Name {
			panic("unreachable: task node name cannot be reversed into tag name; please file a bug on GitHub")
		}

		node = dagCtx.getTaskNode(t.Name)
		if node == nil {
			woc.log.Infof("All of node %s dependencies %s completed", taskNodeName, taskDependencies)
			// Add the child relationship from our dependency's outbound nodes to this node.
			connectDependencies(taskNodeName)

			// Check the task's when clause to decide if it should execute
			proceed, err := shouldExecute(t.When)
			if err != nil {
				woc.initializeNode(taskNodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeError, err.Error())
				continue
			}
			if !proceed {
				skipReason := fmt.Sprintf("when '%s' evaluated false", t.When)
				woc.initializeNode(taskNodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeSkipped, skipReason)
				continue
			}
		}

		// Finally execute the template
		_, _ = woc.executeTemplate(taskNodeName, &t, dagCtx.tmplCtx, t.Arguments, &executeTemplateOpts{boundaryID: dagCtx.boundaryID, onExitTemplate: dagCtx.onExitTemplate})
	}

	if taskGroupNode != nil {
		groupPhase := wfv1.NodeSucceeded
		for _, t := range expandedTasks {
			// Add the child relationship from our dependency's outbound nodes to this node.
			node := dagCtx.getTaskNode(t.Name)
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

func (woc *wfOperationCtx) buildLocalScopeFromTask(dagCtx *dagContext, task *wfv1.DAGTask) (*wfScope, error) {
	// build up the scope
	scope := wfScope{
		tmpl:  dagCtx.tmpl,
		scope: make(map[string]interface{}),
	}
	woc.addOutputsToLocalScope("workflow", woc.wf.Status.Outputs, &scope)

	ancestors := common.GetTaskAncestry(dagCtx, task.Name)
	for _, ancestor := range ancestors {
		ancestorNode := dagCtx.getTaskNode(ancestor)
		if ancestorNode == nil {
			return nil, errors.InternalErrorf("Ancestor task node %s not found", ancestor)
		}
		prefix := fmt.Sprintf("tasks.%s", ancestor)
		if ancestorNode.Type == wfv1.NodeTypeTaskGroup {
			var ancestorNodes []wfv1.NodeStatus
			for _, node := range woc.wf.Status.Nodes {
				if node.BoundaryID == dagCtx.boundaryID && strings.HasPrefix(node.Name, ancestorNode.Name+"(") {
					ancestorNodes = append(ancestorNodes, node)
				}
			}
			_, tmpl, templateStored, err := dagCtx.tmplCtx.ResolveTemplate(ancestorNode)
			if err != nil {
				return nil, errors.InternalWrapError(err)
			}
			// A new template was stored during resolution, persist it
			if templateStored {
				woc.updated = true
			}

			err = woc.processAggregateNodeOutputs(tmpl, &scope, prefix, ancestorNodes)
			if err != nil {
				return nil, errors.InternalWrapError(err)
			}
		} else {
			woc.buildLocalScope(&scope, prefix, ancestorNode)
		}
	}
	return &scope, nil
}

// resolveDependencyReferences replaces any references to outputs of task dependencies, or artifacts in the inputs
// NOTE: by now, input parameters should have been substituted throughout the template
func (woc *wfOperationCtx) resolveDependencyReferences(dagCtx *dagContext, task *wfv1.DAGTask) (*wfv1.DAGTask, error) {

	scope, err := woc.buildLocalScopeFromTask(dagCtx, task)
	if err != nil {
		return nil, err
	}

	// Perform replacement
	// Replace woc.volumes
	err = woc.substituteParamsInVolumes(scope.getParameters())
	if err != nil {
		return nil, err
	}

	// Replace task's parameters
	taskBytes, err := json.Marshal(task)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	fstTmpl := fasttemplate.New(string(taskBytes), "{{", "}}")

	newTaskStr, err := common.Replace(fstTmpl, woc.globalParams.Merge(scope.getParameters()), true)
	if err != nil {
		return nil, err
	}
	var newTask wfv1.DAGTask
	err = json.Unmarshal([]byte(newTaskStr), &newTask)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}

	// If we are not executing, don't attempt to resolve any artifact references. We only check if we are executing after
	// the initial parameter resolution, since it's likely that the "when" clause will contain parameter references.
	proceed, err := shouldExecute(newTask.When)
	if err != nil {
		// If we got an error, it might be because our "when" clause contains a task-expansion parameter (e.g. {{item}}).
		// Since we don't perform task-expansion until later and task-expansion parameters won't get resolved here,
		// we continue execution as normal
		if newTask.ShouldExpand() {
			proceed = true
		} else {
			return nil, err
		}
	}
	if !proceed {
		// We can simply return here; the fact that this task won't execute will be reconciled later on in execution
		return &newTask, nil
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
func (d *dagContext) findLeafTaskNames(tasks []wfv1.DAGTask) []string {
	taskIsLeaf := make(map[string]bool)
	for _, task := range tasks {
		if _, ok := taskIsLeaf[task.Name]; !ok {
			taskIsLeaf[task.Name] = true
		}
		for _, dependency := range d.GetTaskDependencies(task.Name) {
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
func expandTask(task wfv1.DAGTask) ([]wfv1.DAGTask, error) {
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

type TaskResults struct {
	Succeeded bool `json:"Succeeded"`
	Failed    bool `json:"Failed"`
	Errored   bool `json:"Errored"`
	Skipped   bool `json:"Skipped"`
	Completed bool `json:"Completed"`
	Daemoned  bool `json:"Daemoned"`
}

// evaluateDependsLogic returns whether a node should execute and proceed. proceed means that all of its dependencies are
// completed and execute means that given the results of its dependencies, this node should execute.
func (d *dagContext) evaluateDependsLogic(taskName string) (bool, bool, error) {
	evalScope := make(map[string]TaskResults)

	for _, taskName := range d.GetTaskDependencies(taskName) {

		// If the task is still running, we should not proceed.
		depNode := d.getTaskNode(taskName)
		if depNode == nil || !depNode.Completed() {
			return false, false, nil
		}

		evalTaskName := strings.Replace(taskName, "-", "_", -1)
		if _, ok := evalScope[evalTaskName]; ok {
			continue
		}

		evalScope[evalTaskName] = TaskResults{
			Succeeded: depNode.Phase == wfv1.NodeSucceeded,
			Failed:    depNode.Phase == wfv1.NodeFailed,
			Errored:   depNode.Phase == wfv1.NodeError,
			Skipped:   depNode.Phase == wfv1.NodeSkipped,
			Completed: depNode.Phase == wfv1.NodeSucceeded || depNode.Phase == wfv1.NodeFailed,
			Daemoned:  depNode.IsDaemoned() && depNode.Phase != wfv1.NodePending,
		}
	}

	evalLogic := strings.Replace(d.GetTaskDependsLogic(taskName), "-", "_", -1)
	result, err := expr.Eval(evalLogic, evalScope)
	if err != nil {
		return false, false, fmt.Errorf("unable to evaluate expression '%s': %s", evalLogic, err)
	}
	execute, ok := result.(bool)
	if !ok {
		return false, false, fmt.Errorf("unable to cast expression result '%s': %s", result, err)
	}

	return execute, true, nil
}
