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
	// (i.e., "A || (B.Succeeded || B.Failed)" -> "(A.Succeeded || A.Skipped || A.Daemoned) || (B.Succeeded || B.Failed)").
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
	if node == nil {
		return time.Time{}
	}
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
	var dependencyTasks []string
	for dep := range dependencies {
		dependencyTasks = append(dependencyTasks, dep)
	}

	d.dependencies[taskName] = dependencyTasks
	d.dependsLogic[taskName] = resolvedDependsLogic
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

// getTaskNode returns the node status of a task.
func (d *dagContext) getTaskNode(taskName string) *wfv1.NodeStatus {
	nodeID := d.taskNodeID(taskName)
	node, ok := d.wf.Status.Nodes[nodeID]
	if !ok {
		return nil
	}
	return &node
}

// assessDAGPhase assesses the overall DAG status
func (d *dagContext) assessDAGPhase(targetTasks []string, nodes wfv1.Nodes) wfv1.NodePhase {

	// targetTaskPhases keeps track of all the phases of the target tasks. This is necessary because some target tasks may
	// be omitted and will not have an explicit phase. We would still like to deduce a phase for those tasks in order to
	// determine the overall phase of the DAG. To do so, an omitted task always inherits the phase of its parents, with
	// preference of Failed or Error phases over Succeeded. This means that if a task in a branch fails, all of its descendents
	// will be considered Failed unless they themselves complete with a different phase, in which case that different phase
	// will take precedence as the branch phase for their descendents.
	targetTaskPhases := make(map[string]wfv1.NodePhase)
	for _, task := range targetTasks {
		targetTaskPhases[d.taskNodeID(task)] = ""
	}

	// BFS over the children of the DAG
	uniqueQueue := newUniquePhaseNodeQueue(generatePhaseNodes(nodes[d.boundaryID].Children, wfv1.NodeSucceeded)...)
	for !uniqueQueue.empty() {
		curr := uniqueQueue.pop()
		// We need to store the current branchPhase to remember the last completed phase in this branch so that we can apply it to omitted nodes
		node, branchPhase := nodes[curr.nodeId], curr.phase

		if !node.Fulfilled() {
			return wfv1.NodeRunning
		}

		// Only overwrite the branchPhase if this node completed. (If it didn't we can just inherit our parent's branchPhase).
		if node.Completed() {
			branchPhase = node.Phase
		}

		// This node is a target task, so it will not have any children. Store or deduce its phase
		if previousPhase, isTargetTask := targetTaskPhases[node.ID]; isTargetTask {
			// Since we want Failed or Errored phases to have preference over Succeeded in case of ambiguity, only update
			// the deduced phase of the target task if it is not already Failed or Errored.
			// Note that if the target task is NOT omitted (i.e. it Completed), then this check is moot, because every time
			// we arrive at said target task it will have the same branchPhase.
			if !previousPhase.FailedOrError() {
				targetTaskPhases[node.ID] = branchPhase
			}
		}

		if node.Type == wfv1.NodeTypeRetry {
			// A fulfilled Retry node will always reflect the status of its last child node, so its individual attempts don't interest us.
			// To resume the traversal, we look at the children of the last child node.
			if childNode := getChildNodeIndex(&node, nodes, -1); childNode != nil {
				uniqueQueue.add(generatePhaseNodes(childNode.Children, branchPhase)...)
			}
		} else {
			uniqueQueue.add(generatePhaseNodes(node.Children, branchPhase)...)
		}
	}

	// We only succeed if all the target tasks have been considered (i.e. its nodes created) and there are no failures
	failFast := d.tmpl.DAG.FailFast == nil || *d.tmpl.DAG.FailFast
	result := wfv1.NodeSucceeded
	for _, depName := range targetTasks {
		branchPhase := targetTaskPhases[d.taskNodeID(depName)]
		if branchPhase == "" {
			result = wfv1.NodeRunning
			// If failFast is disabled, we will want to let all tasks complete before checking for failures
			if !failFast {
				break
			}
		} else if branchPhase.FailedOrError() {
			// If this target task has continueOn set for its current phase, then don't treat it as failed for the purposes
			// of determining DAG status. This is so that target tasks with said continueOn do not fail the overall DAG.
			// For non-leaf tasks, this is done by setting all of its dependents to allow for their failure or error in
			// their "depends" clause during their respective "dependencies" to "depends" conversion. See "expandDependency"
			// in ancestry.go
			if task := d.GetTask(depName); task.ContinuesOn(branchPhase) {
				continue
			}

			result = branchPhase
			// If failFast is enabled, don't check to see if other target tasks are complete and fail now instead
			if failFast {
				break
			}
		}
	}

	return result
}

func (woc *wfOperationCtx) executeDAG(nodeName string, tmplCtx *templateresolution.Context, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypeDAG, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodeRunning)
	}

	defer func() {
		if woc.wf.Status.Nodes[node.ID].Fulfilled() {
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

		// It is possible that target tasks are not reconsidered (i.e. executeDAGTask is not called on them) once they are
		// complete (since the DAG itself will have succeeded). To ensure that their exit handlers are run we also run them here. Note that
		// calls to runOnExitNode are idempotent: it is fine if they are called more than once for the same task.
		taskNode := dagCtx.getTaskNode(taskName)
		if taskNode != nil && taskNode.Fulfilled() {
			if taskNode.Completed() {
				// Run the node's onExit node, if any. Since this is a target task, we don't need to consider the status
				// of the onExit node before continuing. That will be done in assesDAGPhase
				_, _, err := woc.runOnExitNode(dagCtx.GetTask(taskName).OnExit, taskName, taskNode.Name, dagCtx.boundaryID, dagCtx.tmplCtx)
				if err != nil {
					return node, err
				}
			}
		}
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
		node = woc.wf.GetNodeByName(nodeName)
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
	node := woc.wf.GetNodeByName(nodeName)
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
	if node != nil && node.Fulfilled() {
		// Collect the completed task metrics
		_, tmpl, _, _ := dagCtx.tmplCtx.ResolveTemplate(task)
		if tmpl != nil && tmpl.Metrics != nil {
			if prevNodeStatus, ok := woc.preExecutionNodePhases[node.ID]; ok && !prevNodeStatus.Fulfilled() {
				localScope, realTimeScope := woc.prepareMetricScope(node)
				woc.computeMetrics(tmpl.Metrics.Prometheus, localScope, realTimeScope, false)
			}
		}

		// Release acquired lock completed task.
		if tmpl != nil && tmpl.Synchronization != nil {
			woc.controller.syncManager.Release(woc.wf, node.ID, tmpl.Synchronization)
		}

		if node.Completed() {
			// Run the node's onExit node, if any.
			hasOnExitNode, onExitNode, err := woc.runOnExitNode(task.OnExit, task.Name, node.Name, dagCtx.boundaryID, dagCtx.tmplCtx)
			if hasOnExitNode && (onExitNode == nil || !onExitNode.Fulfilled() || err != nil) {
				// The onExit node is either not complete or has errored out, return.
				return
			}
		}
		return
	}

	// The template scope of this dag.
	dagTemplateScope := dagCtx.tmplCtx.GetTemplateScope()

	// Check if our dependencies completed. If not, recurse our parents executing them if necessary
	nodeName := dagCtx.taskNodeName(taskName)
	taskDependencies := dagCtx.GetTaskDependencies(taskName)

	taskGroupNode := woc.wf.GetNodeByName(nodeName)
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
		// Recurse into all of this node's dependencies
		for _, dep := range taskDependencies {
			woc.executeDAGTask(dagCtx, dep)
		}
		execute, proceed, err := dagCtx.evaluateDependsLogic(taskName)
		if err != nil {
			woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeError, err.Error())
			connectDependencies(nodeName)
			return
		}
		if !proceed {
			// This node's dependencies are not completed yet, return
			return
		}
		if !execute {
			// Given the results of this node's dependencies, this node should not be executed. Mark it omitted
			woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeOmitted, "omitted: depends condition not met")
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
		node = dagCtx.getTaskNode(t.Name)
		if node == nil {
			woc.log.Infof("All of node %s dependencies %v completed", taskNodeName, taskDependencies)
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
		node, err = woc.executeTemplate(taskNodeName, &t, dagCtx.tmplCtx, t.Arguments, &executeTemplateOpts{boundaryID: dagCtx.boundaryID, onExitTemplate: dagCtx.onExitTemplate})
		if err != nil {
			switch err {
			case ErrDeadlineExceeded:
				return
			case ErrParallelismReached:
			case ErrTimeout:
				_ = woc.markNodePhase(taskNodeName, wfv1.NodeFailed, err.Error())
				return
			default:
				woc.log.Infof("DAG %s deemed errored due to task %s error: %s", node.ID, taskNodeName, err.Error())
				_ = woc.markNodePhase(taskNodeName, wfv1.NodeError, fmt.Sprintf("task '%s' errored", taskNodeName))
				return
			}
		}
	}

	if taskGroupNode != nil {
		groupPhase := wfv1.NodeSucceeded
		for _, t := range expandedTasks {
			// Add the child relationship from our dependency's outbound nodes to this node.
			node := dagCtx.getTaskNode(t.Name)
			if node == nil || !node.Fulfilled() {
				return
			}
			if node.FailedOrError() {
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
	fstTmpl, err := fasttemplate.NewTemplate(string(taskBytes), "{{", "}}")
	if err != nil {
		return nil, fmt.Errorf("unable to parse argo varaible: %w", err)
	}

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
		resolvedArt, err := scope.resolveArtifact(art.From, art.SubPath)
		if err != nil {
			if strings.Contains(err.Error(), "Unable to resolve") && art.Optional {
				woc.log.Warnf("Optional artifact '%s' was not found; it won't be available as an input", art.Name)
				continue
			}
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
	var err error
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

	taskBytes, err := json.Marshal(task)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}

	// these fields can be very large (>100m) and marshalling 10k x 100m = 6GB of memory used and
	// very poor performance, so we just nil them out
	task.WithItems = nil
	task.WithParam = ""
	task.WithSequence = nil

	fstTmpl, err := fasttemplate.NewTemplate(string(taskBytes), "{{", "}}")
	if err != nil {
		return nil, fmt.Errorf("unable to parse argo varaible: %w", err)
	}
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
	Succeeded    bool `json:"Succeeded"`
	Failed       bool `json:"Failed"`
	Errored      bool `json:"Errored"`
	Skipped      bool `json:"Skipped"`
	Daemoned     bool `json:"Daemoned"`
	AnySucceeded bool `json:"AnySucceeded"`
	AllFailed    bool `json:"AllFailed"`
}

// evaluateDependsLogic returns whether a node should execute and proceed. proceed means that all of its dependencies are
// completed and execute means that given the results of its dependencies, this node should execute.
func (d *dagContext) evaluateDependsLogic(taskName string) (bool, bool, error) {
	evalScope := make(map[string]TaskResults)

	for _, taskName := range d.GetTaskDependencies(taskName) {

		// If the task is still running, we should not proceed.
		depNode := d.getTaskNode(taskName)
		if depNode == nil || !depNode.Fulfilled() {
			return false, false, nil
		}

		// If a task happens to have an onExit node, don't proceed until the onExit node is fulfilled
		if onExitNode := d.wf.GetNodeByName(common.GenerateOnExitNodeName(taskName)); onExitNode != nil {
			if !onExitNode.Fulfilled() {
				return false, false, nil
			}
		}

		evalTaskName := strings.Replace(taskName, "-", "_", -1)
		if _, ok := evalScope[evalTaskName]; ok {
			continue
		}

		anySucceeded := false
		allFailed := false

		if depNode.Type == wfv1.NodeTypeTaskGroup {

			allFailed = len(depNode.Children) > 0

			for _, childNodeID := range depNode.Children {
				childNode := d.wf.Status.Nodes[childNodeID]
				anySucceeded = anySucceeded || childNode.Phase == wfv1.NodeSucceeded
				allFailed = allFailed && childNode.Phase == wfv1.NodeFailed
			}
		}

		evalScope[evalTaskName] = TaskResults{
			Succeeded:    depNode.Phase == wfv1.NodeSucceeded,
			Failed:       depNode.Phase == wfv1.NodeFailed,
			Errored:      depNode.Phase == wfv1.NodeError,
			Skipped:      depNode.Phase == wfv1.NodeSkipped,
			Daemoned:     depNode.IsDaemoned() && depNode.Phase != wfv1.NodePending,
			AnySucceeded: anySucceeded,
			AllFailed:    allFailed,
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
