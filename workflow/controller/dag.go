package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v3/util/template"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	controllercache "github.com/argoproj/argo-workflows/v3/workflow/controller/cache"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
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
	// task with their explicit results since we allow them to be omitted for convenience
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
	node, err := d.wf.Status.Nodes.Get(nodeID)
	if err != nil {
		log.Warnf("was unable to obtain the node for %s, taskName %s", nodeID, taskName)
		return nil
	}
	return node
}

// assessDAGPhase assesses the overall DAG status
func (d *dagContext) assessDAGPhase(targetTasks []string, nodes wfv1.Nodes, isShutdown bool) (wfv1.NodePhase, error) {
	// We cannot only rely on the DAG traversal. Conditionals, self-references,
	// and ContinuesOn (every one of those features in unison) make this an undecidable problem.
	// However, we can just use isShutdown to automatically fail the DAG.
	if isShutdown {
		return wfv1.NodeFailed, nil
	}
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

	boundaryNode, err := nodes.Get(d.boundaryID)
	if err != nil {
		return "", err
	}
	// BFS over the children of the DAG
	uniqueQueue := newUniquePhaseNodeQueue(generatePhaseNodes(boundaryNode.Children, wfv1.NodeSucceeded)...)
	for !uniqueQueue.empty() {
		curr := uniqueQueue.pop()

		node, err := nodes.Get(curr.nodeID)
		if err != nil {
			// this is okay, this means that
			// we are still running
			return wfv1.NodeRunning, nil
		}
		// We need to store the current branchPhase to remember the last completed phase in this branch so that we can apply it to omitted nodes
		branchPhase := curr.phase

		if !node.Fulfilled() {
			return wfv1.NodeRunning, nil
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
			uniqueQueue.add(generatePhaseNodes(getRetryNodeChildrenIds(node, nodes), branchPhase)...)
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

	return result, nil
}

func (woc *wfOperationCtx) executeDAG(ctx context.Context, nodeName string, tmplCtx *templateresolution.Context, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {

	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypeDAG, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodeRunning, opts.nodeFlag)
	}

	defer func() {
		node, err := woc.wf.Status.Nodes.Get(node.ID)
		if err != nil {
			// CRITICAL ERROR IF THIS BRANCH IS REACHED -> PANIC
			panic(fmt.Sprintf("expected node for %s due to preceded initializeExecutableNode but couldn't find it", node.ID))
		}
		if node.Fulfilled() {
			woc.killDaemonedChildren(node.ID)
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

	// pre-execute daemoned tasks
	for _, task := range tmpl.DAG.Tasks {
		taskNode := dagCtx.getTaskNode(task.Name)
		if err != nil {
			continue
		}
		if taskNode != nil && taskNode.IsDaemoned() {
			woc.executeDAGTask(ctx, dagCtx, task.Name)
		}
	}

	// kick off execution of each target task asynchronously
	onExitCompleted := true
	for _, taskName := range targetTasks {
		woc.executeDAGTask(ctx, dagCtx, taskName)

		// It is possible that target tasks are not reconsidered (i.e. executeDAGTask is not called on them) once they are
		// complete (since the DAG itself will have succeeded). To ensure that their exit handlers are run we also run them here. Note that
		// calls to runOnExitNode are idempotent: it is fine if they are called more than once for the same task.
		taskNode := dagCtx.getTaskNode(taskName)

		if taskNode != nil {
			task := dagCtx.GetTask(taskName)
			scope, err := woc.buildLocalScopeFromTask(dagCtx, task)
			if err != nil {
				woc.markNodeError(node.Name, err)
				return node, err
			}
			scope.addParamToScope(fmt.Sprintf("tasks.%s.status", task.Name), string(taskNode.Phase))
			_, err = woc.executeTmplLifeCycleHook(ctx, scope, dagCtx.GetTask(taskName).Hooks, taskNode, dagCtx.boundaryID, dagCtx.tmplCtx, "tasks."+taskName)
			if err != nil {
				woc.markNodeError(node.Name, err)
				return node, err
			}
			if taskNode.Fulfilled() {
				if taskNode.Completed() {
					hasOnExitNode, onExitNode, err := woc.runOnExitNode(ctx, dagCtx.GetTask(taskName).GetExitHook(woc.execWf.Spec.Arguments), taskNode, dagCtx.boundaryID, dagCtx.tmplCtx, "tasks."+taskName, scope)
					if err != nil {
						return node, err
					}
					if hasOnExitNode && (onExitNode == nil || !onExitNode.Fulfilled()) {
						onExitCompleted = false
					}
				}
			}
		}
	}

	// Check if we are still running any tasks in this dag and return early if we do
	// We should wait for onExit nodes even if ShutdownStrategy is enabled.
	dagPhase, err := dagCtx.assessDAGPhase(targetTasks, woc.wf.Status.Nodes, woc.GetShutdownStrategy().Enabled() && onExitCompleted)
	if err != nil {
		return nil, err
	}

	switch dagPhase {
	case wfv1.NodeRunning:
		return node, nil
	case wfv1.NodeError, wfv1.NodeFailed:
		err = woc.updateOutboundNodesForTargetTasks(dagCtx, targetTasks, nodeName)
		if err != nil {
			return nil, err
		}
		_ = woc.markNodePhase(nodeName, dagPhase)
		return node, nil
	}

	// set outputs from tasks in order for DAG templates to support outputs
	scope := createScope(tmpl)
	for _, task := range tmpl.DAG.Tasks {
		taskNode := dagCtx.getTaskNode(task.Name)
		if taskNode == nil {
			// Can happen when dag.target was specified
			continue
		}

		prefix := fmt.Sprintf("tasks.%s", task.Name)
		if taskNode.Type == wfv1.NodeTypeTaskGroup {
			childNodes := make([]wfv1.NodeStatus, len(taskNode.Children))
			for i, childID := range taskNode.Children {
				childNode, err := woc.wf.Status.Nodes.Get(childID)
				if err != nil {
					woc.log.Errorf("was unable to obtain node for %s", childID)
					return nil, fmt.Errorf("Critical error, unable to find %s", childID)
				}
				childNodes[i] = *childNode
			}
			err := woc.processAggregateNodeOutputs(scope, prefix, childNodes)
			if err != nil {
				woc.log.Errorf("unable to processAggregateNodeOutputs")
				return nil, errors.InternalWrapError(err)
			}
		}
		woc.buildLocalScope(scope, prefix, taskNode)
		woc.addOutputsToGlobalScope(taskNode.Outputs)
	}
	outputs, err := getTemplateOutputsFromScope(tmpl, scope)
	if err != nil {
		woc.log.Errorf("unable to get outputs")
		return node, err
	}
	if outputs != nil {
		node, err = woc.wf.GetNodeByName(nodeName)
		if err != nil {
			woc.log.Errorf("unable to get node by name for %s", nodeName)
			return nil, err
		}
		node.Outputs = outputs
		woc.wf.Status.Nodes.Set(node.ID, *node)
	}
	if node.MemoizationStatus != nil {
		c := woc.controller.cacheFactory.GetCache(controllercache.ConfigMapCache, node.MemoizationStatus.CacheName)
		err := c.Save(ctx, node.MemoizationStatus.Key, node.ID, node.Outputs)
		if err != nil {
			woc.log.WithFields(log.Fields{"nodeID": node.ID}).WithError(err).Error("Failed to save node outputs to cache")
			node.Phase = wfv1.NodeError
		}
	}

	err = woc.updateOutboundNodesForTargetTasks(dagCtx, targetTasks, nodeName)
	if err != nil {
		return nil, err
	}
	return woc.markNodePhase(nodeName, wfv1.NodeSucceeded), nil
}

func (woc *wfOperationCtx) updateOutboundNodesForTargetTasks(dagCtx *dagContext, targetTasks []string, nodeName string) error {
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
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		woc.log.Warnf("was unable to obtain node by name for %s", nodeName)
		return err
	}
	node.OutboundNodes = outbound
	woc.wf.Status.Nodes.Set(node.ID, *node)
	woc.log.Infof("Outbound nodes of %s set to %s", node.ID, outbound)
	return nil
}

// executeDAGTask traverses and executes the upward chain of dependencies of a task
func (woc *wfOperationCtx) executeDAGTask(ctx context.Context, dagCtx *dagContext, taskName string) {
	if _, ok := dagCtx.visited[taskName]; ok {
		return
	}
	dagCtx.visited[taskName] = true

	node := dagCtx.getTaskNode(taskName)
	task := dagCtx.GetTask(taskName)
	log := woc.log.WithField("taskName", taskName)
	if node != nil && (node.Fulfilled() || node.Phase == wfv1.NodeRunning) {
		scope, err := woc.buildLocalScopeFromTask(dagCtx, task)
		if err != nil {
			log.Error("Failed to build local scope from task")
			woc.markNodeError(node.Name, err)
			return
		}
		scope.addParamToScope(fmt.Sprintf("tasks.%s.status", task.Name), string(node.Phase))
		hookCompleted, err := woc.executeTmplLifeCycleHook(ctx, scope, dagCtx.GetTask(taskName).Hooks, node, dagCtx.boundaryID, dagCtx.tmplCtx, "tasks."+taskName)
		if err != nil {
			woc.markNodeError(node.Name, err)
		}
		// Check all hooks are completes
		if !hookCompleted {
			return
		}
	}

	if node != nil && node.Phase.Fulfilled() {
		// Collect the completed task metrics
		_, tmpl, _, tmplErr := dagCtx.tmplCtx.ResolveTemplate(task)
		if tmplErr != nil {
			woc.markNodeError(node.Name, tmplErr)
			return
		}
		if err := woc.mergedTemplateDefaultsInto(tmpl); err != nil {
			woc.markNodeError(node.Name, err)
			return
		}
		if tmpl != nil && tmpl.Metrics != nil {
			if prevNodeStatus, ok := woc.preExecutionNodePhases[node.ID]; ok && !prevNodeStatus.Fulfilled() {
				localScope, realTimeScope := woc.prepareMetricScope(node)
				woc.computeMetrics(ctx, tmpl.Metrics.Prometheus, localScope, realTimeScope, false)
			}
		}

		processedTmpl, err := common.ProcessArgs(tmpl, &task.Arguments, woc.globalParams, map[string]string{}, true, woc.wf.Namespace, woc.controller.configMapInformer.GetIndexer())
		if err != nil {
			woc.markNodeError(node.Name, err)
		}

		// Release acquired lock completed task.
		if processedTmpl != nil {
			woc.controller.syncManager.Release(ctx, woc.wf, node.ID, processedTmpl.Synchronization)
		}

		scope, err := woc.buildLocalScopeFromTask(dagCtx, task)
		if err != nil {
			woc.markNodeError(node.Name, err)
			log.Error("Failed to build local scope from task")
			return
		}
		scope.addParamToScope(fmt.Sprintf("tasks.%s.status", task.Name), string(node.Phase))

		if node.Completed() {
			// Run the node's onExit node, if any.
			hasOnExitNode, onExitNode, err := woc.runOnExitNode(ctx, task.GetExitHook(woc.execWf.Spec.Arguments), node, dagCtx.boundaryID, dagCtx.tmplCtx, "tasks."+taskName, scope)
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
	// error condition taken care of via a nil check
	taskGroupNode, _ := woc.wf.GetNodeByName(nodeName)

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
					nodeName, err := woc.wf.Status.Nodes.GetName(outNodeID)
					if err != nil {
						woc.log.Errorf("was unable to obtain node for %s", outNodeID)
						return
					}
					woc.addChildNode(nodeName, taskNodeName)
				}
			}
		}
	}

	if dagCtx.GetTaskDependsLogic(taskName) != "" {
		// Recurse into all of this node's dependencies
		for _, dep := range taskDependencies {
			woc.executeDAGTask(ctx, dagCtx, dep)
		}
		execute, proceed, err := dagCtx.evaluateDependsLogic(taskName)
		if err != nil {
			woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeError, &wfv1.NodeFlag{}, err.Error())
			connectDependencies(nodeName)
			return
		}
		if !proceed {
			// This node's dependencies are not completed yet, return
			return
		}
		if !execute {
			// Given the results of this node's dependencies, this node should not be executed. Mark it omitted
			woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeOmitted, &wfv1.NodeFlag{}, "omitted: depends condition not met")
			connectDependencies(nodeName)
			return
		}
	}

	// All our dependencies were satisfied and successful. It's our turn to run
	// First resolve/substitute params/artifacts from our dependencies
	newTask, err := woc.resolveDependencyReferences(dagCtx, task)
	if err != nil {
		woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeError, &wfv1.NodeFlag{}, err.Error())
		connectDependencies(nodeName)
		return
	}

	// Next, expand the DAG's withItems/withParams/withSequence (if any). If there was none, then
	// expandedTasks will be a single element list of the same task
	expandedTasks, err := expandTask(*newTask)
	if err != nil {
		woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeError, &wfv1.NodeFlag{}, err.Error())
		connectDependencies(nodeName)
		return
	}

	// If DAG task has withParam of with withSequence then we need to create virtual node of type TaskGroup.
	// For example, if we had task A with withItems of ['foo', 'bar'] which expanded to ['A(0:foo)', 'A(1:bar)'], we still
	// need to create a node for A.
	if task.ShouldExpand() {
		// DAG task with empty withParams list should be skipped
		if len(expandedTasks) == 0 {
			skipReason := "Skipped, empty params"
			woc.initializeNode(nodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeSkipped, &wfv1.NodeFlag{}, skipReason)
			connectDependencies(nodeName)
		} else if taskGroupNode == nil {
			connectDependencies(nodeName)
			taskGroupNode = woc.initializeNode(nodeName, wfv1.NodeTypeTaskGroup, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeRunning, &wfv1.NodeFlag{}, "")
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
				woc.initializeNode(taskNodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeError, &wfv1.NodeFlag{}, err.Error())
				continue
			}
			if !proceed {
				skipReason := fmt.Sprintf("when '%s' evaluated false", t.When)
				woc.initializeNode(taskNodeName, wfv1.NodeTypeSkipped, dagTemplateScope, task, dagCtx.boundaryID, wfv1.NodeSkipped, &wfv1.NodeFlag{}, skipReason)
				continue
			}
		}

		// Finally execute the template
		node, err = woc.executeTemplate(ctx, taskNodeName, &t, dagCtx.tmplCtx, t.Arguments, &executeTemplateOpts{boundaryID: dagCtx.boundaryID, onExitTemplate: dagCtx.onExitTemplate})
		if err != nil {
			switch err {
			case ErrDeadlineExceeded:
				return
			case ErrParallelismReached:
			case ErrMaxDepthExceeded:
			case ErrTimeout:
				_ = woc.markNodePhase(taskNodeName, wfv1.NodeFailed, err.Error())
				return
			default:
				_ = woc.markNodeError(taskNodeName, fmt.Errorf("task '%s' errored: %v", taskNodeName, err))
				return
			}
		}
		// Some scenario, Node will be nil e.g: when parallelism reached.
		if node == nil {
			return
		}
		if node.Completed() {
			scope, err := woc.buildLocalScopeFromTask(dagCtx, task)
			if err != nil {
				woc.markNodeError(node.Name, err)
			}
			scope.addParamToScope(fmt.Sprintf("tasks.%s.status", task.Name), string(node.Phase))
			// if the node type is NodeTypeRetry, and its last child is completed, it will be completed after woc.executeTemplate;
			hasOnExitNode, onExitNode, err := woc.runOnExitNode(ctx, task.GetExitHook(woc.execWf.Spec.Arguments), node, dagCtx.boundaryID, dagCtx.tmplCtx, "tasks."+taskName, scope)
			if hasOnExitNode && (onExitNode == nil || !onExitNode.Fulfilled() || err != nil) {
				// The onExit node is either not complete or has errored out, return.
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
	scope := createScope(dagCtx.tmpl)
	woc.addOutputsToLocalScope("workflow", woc.wf.Status.Outputs, scope)

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
					// Filter retried nodes and only aggregate outputs of their parent nodes.
					if node.NodeFlag != nil && node.NodeFlag.Retried {
						continue
					}
					ancestorNodes = append(ancestorNodes, node)
				}
			}
			_, _, templateStored, err := dagCtx.tmplCtx.ResolveTemplate(ancestorNode)
			if err != nil {
				return nil, errors.InternalWrapError(err)
			}
			// A new template was stored during resolution, persist it
			if templateStored {
				woc.updated = true
			}

			err = woc.processAggregateNodeOutputs(scope, prefix, ancestorNodes)
			if err != nil {
				return nil, errors.InternalWrapError(err)
			}
		} else {
			woc.buildLocalScope(scope, prefix, ancestorNode)
		}
	}
	return scope, nil
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
	newTaskStr, err := template.Replace(string(taskBytes), woc.globalParams.Merge(scope.getParameters()), true)
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

	artifacts := wfv1.Artifacts{}
	// replace all artifact references
	for _, art := range newTask.Arguments.Artifacts {
		if art.From == "" && art.FromExpression == "" {
			artifacts = append(artifacts, art)
			continue
		}
		resolvedArt, err := scope.resolveArtifact(&art)
		if err != nil {
			if strings.Contains(err.Error(), "Unable to resolve") && art.Optional {
				woc.log.Warnf("Optional artifact '%s' was not found; it won't be available as an input", art.Name)
				continue
			}
			return nil, err
		}
		resolvedArt.Name = art.Name
		artifacts = append(artifacts, *resolvedArt)
	}
	newTask.Arguments.Artifacts = artifacts
	return &newTask, nil
}

// findLeafTaskNames finds the names of all tasks whom no other nodes depend on.
// This list of tasks is used as the default list of targets when dag.targets is omitted.
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
	sort.Strings(leafTaskNames) // execute tasks in a predictable order
	return leafTaskNames
}

// expandTask expands a single DAG task containing withItems, withParams, withSequence into multiple parallel tasks
// We want to be lazy with expanding. Unfortunately this is not quite possible as the When field might rely on
// expansion to work with the shouldExecute function. To address this we apply a trick, we try to expand, if we fail, we then
// check shouldExecute, if shouldExecute returns false, we continue on as normal else error out
func expandTask(task wfv1.DAGTask) ([]wfv1.DAGTask, error) {
	var err error
	var items []wfv1.Item
	if len(task.WithItems) > 0 {
		items = task.WithItems
	} else if task.WithParam != "" {
		err = json.Unmarshal([]byte(task.WithParam), &items)
		if err != nil {
			mustExec, mustExecErr := shouldExecute(task.When)
			if mustExecErr != nil || mustExec {
				return nil, errors.Errorf(errors.CodeBadRequest, "withParam value could not be parsed as a JSON list: %s: %v", strings.TrimSpace(task.WithParam), err)
			}
		}
	} else if task.WithSequence != nil {
		items, err = expandSequence(task.WithSequence)
		if err != nil {
			mustExec, mustExecErr := shouldExecute(task.When)
			if mustExecErr != nil || mustExec {
				return nil, err
			}
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

	tmpl, err := template.NewTemplate(string(taskBytes))
	if err != nil {
		return nil, fmt.Errorf("unable to parse argo variable: %w", err)
	}
	expandedTasks := make([]wfv1.DAGTask, 0)
	for i, item := range items {
		var newTask wfv1.DAGTask
		newTaskName, err := processItem(tmpl, task.Name, i, item, &newTask, task.When)
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
	Omitted      bool `json:"Omitted"`
	Daemoned     bool `json:"Daemoned"`
	AnySucceeded bool `json:"AnySucceeded"`
	AllFailed    bool `json:"AllFailed"`
}

// evaluateDependsLogic returns whether a node should execute and proceed. proceed means that all of its dependencies are
// completed and execute means that given the results of its dependencies, this node should execute.
func (d *dagContext) evaluateDependsLogic(taskName string) (bool, bool, error) {
	node := d.getTaskNode(taskName)
	if node != nil {
		return true, true, nil
	}

	evalScope := make(map[string]TaskResults)

	for _, taskName := range d.GetTaskDependencies(taskName) {

		// If the task is still running, we should not proceed.
		depNode := d.getTaskNode(taskName)
		if depNode == nil || !depNode.Fulfilled() || !common.CheckAllHooksFullfilled(depNode, d.wf.Status.Nodes) {
			return false, false, nil
		}

		evalTaskName := strings.ReplaceAll(taskName, "-", "_")
		if _, ok := evalScope[evalTaskName]; ok {
			continue
		}

		anySucceeded := false
		allFailed := false

		if depNode.Type == wfv1.NodeTypeTaskGroup {

			allFailed = len(depNode.Children) > 0

			for _, childNodeID := range depNode.Children {
				childNodePhase, err := d.wf.Status.Nodes.GetPhase(childNodeID)
				if err != nil {
					log.Warnf("was unable to obtain node for %s", childNodeID)
					allFailed = false // we don't know if all failed
					continue
				}
				anySucceeded = anySucceeded || *childNodePhase == wfv1.NodeSucceeded
				allFailed = allFailed && *childNodePhase == wfv1.NodeFailed
			}
		}

		evalScope[evalTaskName] = TaskResults{
			Succeeded:    depNode.Phase == wfv1.NodeSucceeded,
			Failed:       depNode.Phase == wfv1.NodeFailed,
			Errored:      depNode.Phase == wfv1.NodeError,
			Skipped:      depNode.Phase == wfv1.NodeSkipped,
			Omitted:      depNode.Phase == wfv1.NodeOmitted,
			Daemoned:     depNode.IsDaemoned() && depNode.Phase != wfv1.NodePending,
			AnySucceeded: anySucceeded,
			AllFailed:    allFailed,
		}
	}

	evalLogic := strings.ReplaceAll(d.GetTaskDependsLogic(taskName), "-", "_")
	execute, err := argoexpr.EvalBool(evalLogic, evalScope)
	if err != nil {
		return false, false, fmt.Errorf("unable to evaluate expression '%s': %s", evalLogic, err)
	}
	return execute, true, nil
}
