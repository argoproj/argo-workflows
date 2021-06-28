package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Knetic/govaluate"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/template"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

// stepsContext holds context information about this context's steps
type stepsContext struct {
	// boundaryID is the node ID of the boundary which all immediate child steps are bound to
	boundaryID string

	// scope holds parameter and artifacts which are referenceable in scope during execution
	scope *wfScope

	// tmplCtx is the context of template search.
	tmplCtx *templateresolution.Context

	// onExitTemplate is a flag denoting this template as part of an onExit handler. This is necessary to ensure that
	// further nodes stemming from this template are allowed to run when using "ShutdownStrategy: Stop"
	onExitTemplate bool
}

func (woc *wfOperationCtx) executeSteps(ctx context.Context, nodeName string, tmplCtx *templateresolution.Context, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypeSteps, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodeRunning)
	}

	defer func() {
		if woc.wf.Status.Nodes[node.ID].Fulfilled() {
			woc.killDaemonedChildren(node.ID)
		}
	}()

	// The template scope of this step.
	stepTemplateScope := tmplCtx.GetTemplateScope()

	stepsCtx := stepsContext{
		boundaryID:     node.ID,
		scope:          createScope(tmpl),
		tmplCtx:        tmplCtx,
		onExitTemplate: opts.onExitTemplate,
	}
	woc.addOutputsToLocalScope("workflow", woc.wf.Status.Outputs, stepsCtx.scope)

	for i, stepGroup := range tmpl.Steps {
		sgNodeName := fmt.Sprintf("%s[%d]", nodeName, i)
		{
			sgNode := woc.wf.GetNodeByName(sgNodeName)
			if sgNode == nil {
				_ = woc.initializeNode(sgNodeName, wfv1.NodeTypeStepGroup, stepTemplateScope, &wfv1.WorkflowStep{}, stepsCtx.boundaryID, wfv1.NodeRunning)
			} else if !sgNode.Fulfilled() {
				_ = woc.markNodePhase(sgNodeName, wfv1.NodeRunning)
			}
		}
		// The following will connect the step group node to its parents.
		if i == 0 {
			// If this is the first step group, the boundary node is the parent
			woc.addChildNode(nodeName, sgNodeName)
		} else {
			// Otherwise connect all the outbound nodes of the previous step group as parents to
			// the current step group node.
			prevStepGroupName := fmt.Sprintf("%s[%d]", nodeName, i-1)
			prevStepGroupNode := woc.wf.GetNodeByName(prevStepGroupName)
			if len(prevStepGroupNode.Children) == 0 {
				// corner case which connects an empty StepGroup (e.g. due to empty withParams) to
				// the previous StepGroup node
				woc.addChildNode(prevStepGroupName, sgNodeName)
			} else {
				for _, childID := range prevStepGroupNode.Children {
					outboundNodeIDs := woc.getOutboundNodes(childID)
					woc.log.Infof("SG Outbound nodes of %s are %s", childID, outboundNodeIDs)
					for _, outNodeID := range outboundNodeIDs {
						woc.addChildNode(woc.wf.Status.Nodes[outNodeID].Name, sgNodeName)
					}
				}
			}
		}

		sgNode := woc.executeStepGroup(ctx, stepGroup.Steps, sgNodeName, &stepsCtx)

		if !sgNode.Fulfilled() {
			woc.log.Infof("Workflow step group node %s not yet completed", sgNode.ID)
			return node, nil
		}

		if sgNode.FailedOrError() {
			failMessage := fmt.Sprintf("step group %s was unsuccessful: %s", sgNode.ID, sgNode.Message)
			woc.log.Info(failMessage)
			woc.updateOutboundNodes(nodeName, tmpl)
			return woc.markNodePhase(nodeName, wfv1.NodeFailed, sgNode.Message), nil
		}

		// Add all outputs of each step in the group to the scope
		for _, step := range stepGroup.Steps {
			childNodeName := fmt.Sprintf("%s.%s", sgNodeName, step.Name)
			childNode := woc.wf.GetNodeByName(childNodeName)
			prefix := fmt.Sprintf("steps.%s", step.Name)
			if childNode == nil {
				// This happens when there was `withItem/withParam` expansion.
				// We add the aggregate outputs of our children to the scope as a JSON list
				var childNodes []wfv1.NodeStatus
				for _, node := range woc.wf.Status.Nodes {
					if node.BoundaryID == stepsCtx.boundaryID && strings.HasPrefix(node.Name, childNodeName+"(") && node.Type != wfv1.NodeTypeSkipped {
						childNodes = append(childNodes, node)
					}
				}
				if len(childNodes) > 0 {
					// Expanded child nodes should be created from the same template.
					_, tmpl, templateStored, err := stepsCtx.tmplCtx.ResolveTemplate(&childNodes[0])
					if err != nil {
						return node, err
					}
					// A new template was stored during resolution, persist it
					if templateStored {
						woc.updated = true
					}

					err = woc.processAggregateNodeOutputs(tmpl, stepsCtx.scope, prefix, childNodes)
					if err != nil {
						return node, err
					}
				} else {
					woc.log.Infof("Step '%s' has no expanded child nodes", childNode)
				}
			} else {
				woc.buildLocalScope(stepsCtx.scope, prefix, childNode)
			}
		}
	}

	woc.updateOutboundNodes(nodeName, tmpl)
	// If this template has outputs from any of its steps, copy them to this node here
	outputs, err := getTemplateOutputsFromScope(tmpl, stepsCtx.scope)
	if err != nil {
		return node, err
	}
	if outputs != nil {
		node := woc.wf.GetNodeByName(nodeName)
		node.Outputs = outputs
		woc.addOutputsToGlobalScope(node.Outputs)
		woc.wf.Status.Nodes[node.ID] = *node
	}
	return woc.markNodePhase(nodeName, wfv1.NodeSucceeded), nil
}

// updateOutboundNodes set the outbound nodes from the last step group
func (woc *wfOperationCtx) updateOutboundNodes(nodeName string, tmpl *wfv1.Template) {
	outbound := make([]string, 0)
	// Find the last, initialized stepgroup node
	var lastSGNode *wfv1.NodeStatus
	for i := len(tmpl.Steps) - 1; i >= 0; i-- {
		sgNode := woc.wf.GetNodeByName(fmt.Sprintf("%s[%d]", nodeName, i))
		if sgNode != nil {
			lastSGNode = sgNode
			break
		}
	}
	if lastSGNode == nil {
		woc.log.Warnf("node '%s' had no initialized StepGroup nodes", nodeName)
		return
	}
	for _, childID := range lastSGNode.Children {
		outboundNodeIDs := woc.getOutboundNodes(childID)
		woc.log.Infof("Outbound nodes of %s is %s", childID, outboundNodeIDs)
		outbound = append(outbound, outboundNodeIDs...)
	}
	node := woc.wf.GetNodeByName(nodeName)
	woc.log.Infof("Outbound nodes of %s is %s", node.ID, outbound)
	node.OutboundNodes = outbound
	woc.wf.Status.Nodes[node.ID] = *node
}

// executeStepGroup examines a list of parallel steps and executes them in parallel.
// Handles referencing of variables in scope, expands `withItem` clauses, and evaluates `when` expressions
func (woc *wfOperationCtx) executeStepGroup(ctx context.Context, stepGroup []wfv1.WorkflowStep, sgNodeName string, stepsCtx *stepsContext) *wfv1.NodeStatus {
	node := woc.wf.GetNodeByName(sgNodeName)
	if node.Fulfilled() {
		woc.log.Debugf("Step group node %v already marked completed", node)
		return node
	}

	// First, resolve any references to outputs from previous steps, and perform substitution
	stepGroup, err := woc.resolveReferences(stepGroup, stepsCtx.scope)
	if err != nil {
		return woc.markNodeError(sgNodeName, err)
	}

	// Next, expand the step's withItems (if any)
	stepGroup, err = woc.expandStepGroup(sgNodeName, stepGroup, stepsCtx)
	if err != nil {
		return woc.markNodeError(sgNodeName, err)
	}

	// Maps nodes to their steps
	nodeSteps := make(map[string]wfv1.WorkflowStep)

	// The template scope of this step group.
	stepTemplateScope := stepsCtx.tmplCtx.GetTemplateScope()

	// Kick off all parallel steps in the group
	for _, step := range stepGroup {
		childNodeName := fmt.Sprintf("%s.%s", sgNodeName, step.Name)

		// Check the step's when clause to decide if it should execute
		proceed, err := shouldExecute(step.When)
		if err != nil {
			woc.initializeNode(childNodeName, wfv1.NodeTypeSkipped, stepTemplateScope, &step, stepsCtx.boundaryID, wfv1.NodeError, err.Error())
			woc.addChildNode(sgNodeName, childNodeName)
			woc.markNodeError(childNodeName, err)
			return woc.markNodeError(sgNodeName, err)
		}
		if !proceed {
			if woc.wf.GetNodeByName(childNodeName) == nil {
				skipReason := fmt.Sprintf("when '%s' evaluated false", step.When)
				woc.log.Infof("Skipping %s: %s", childNodeName, skipReason)
				woc.initializeNode(childNodeName, wfv1.NodeTypeSkipped, stepTemplateScope, &step, stepsCtx.boundaryID, wfv1.NodeSkipped, skipReason)
				woc.addChildNode(sgNodeName, childNodeName)
			}
			continue
		}

		childNode, err := woc.executeTemplate(ctx, childNodeName, &step, stepsCtx.tmplCtx, step.Arguments, &executeTemplateOpts{boundaryID: stepsCtx.boundaryID, onExitTemplate: stepsCtx.onExitTemplate})
		if err != nil {
			switch err {
			case ErrDeadlineExceeded:
				return node
			case ErrParallelismReached:
			case ErrTimeout:
				return woc.markNodePhase(node.Name, wfv1.NodeFailed, fmt.Sprintf("child '%s' timedout", childNodeName))
			default:
				woc.addChildNode(sgNodeName, childNodeName)
				return woc.markNodeError(node.Name, fmt.Errorf("step group deemed errored due to child %s error: %w", childNodeName, err))
			}
		}
		if childNode != nil {
			nodeSteps[childNodeName] = step
			woc.addChildNode(sgNodeName, childNodeName)
		}
	}

	node = woc.wf.GetNodeByName(sgNodeName)
	// Return if not all children completed
	completed := true
	for _, childNodeID := range node.Children {
		childNode := woc.wf.Status.Nodes[childNodeID]
		step := nodeSteps[childNode.Name]
		if !childNode.Fulfilled() {
			completed = false
		} else if childNode.Completed() {
			hasOnExitNode, onExitNode, err := woc.runOnExitNode(ctx, step.GetExitHook(woc.execWf.Spec.Arguments), step.Name, childNode.Name, stepsCtx.boundaryID, stepsCtx.tmplCtx, "steps."+step.Name, childNode.Outputs)
			if hasOnExitNode && (onExitNode == nil || !onExitNode.Fulfilled() || err != nil) {
				// The onExit node is either not complete or has errored out, return.
				completed = false
			}
		}
	}
	if !completed {
		return node
	}

	woc.addOutputsToGlobalScope(node.Outputs)

	// All children completed. Determine step group status as a whole
	for _, childNodeID := range node.Children {
		childNode := woc.wf.Status.Nodes[childNodeID]
		step := nodeSteps[childNode.Name]
		if childNode.FailedOrError() && !step.ContinuesOn(childNode.Phase) {
			failMessage := fmt.Sprintf("child '%s' failed", childNodeID)
			woc.log.Infof("Step group node %s deemed failed: %s", node.ID, failMessage)
			return woc.markNodePhase(node.Name, wfv1.NodeFailed, failMessage)
		}
	}
	woc.log.Infof("Step group node %v successful", node.ID)
	return woc.markNodePhase(node.Name, wfv1.NodeSucceeded)
}

// shouldExecute evaluates a already substituted when expression to decide whether or not a step should execute
func shouldExecute(when string) (bool, error) {
	if when == "" {
		return true, nil
	}
	expression, err := govaluate.NewEvaluableExpression(when)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid token") {
			return false, errors.Errorf(errors.CodeBadRequest, "Invalid 'when' expression '%s': %v (hint: try wrapping the affected expression in quotes (\"))", when, err)
		}
		return false, errors.Errorf(errors.CodeBadRequest, "Invalid 'when' expression '%s': %v", when, err)
	}
	// The following loop converts govaluate variables (which we don't use), into strings. This
	// allows us to have expressions like: "foo != bar" without requiring foo and bar to be quoted.
	tokens := expression.Tokens()
	for i, tok := range tokens {
		switch tok.Kind {
		case govaluate.VARIABLE:
			tok.Kind = govaluate.STRING
		default:
			continue
		}
		tokens[i] = tok
	}
	expression, err = govaluate.NewEvaluableExpressionFromTokens(tokens)
	if err != nil {
		return false, errors.InternalWrapErrorf(err, "Failed to parse 'when' expression '%s': %v", when, err)
	}
	result, err := expression.Evaluate(nil)
	if err != nil {
		return false, errors.InternalWrapErrorf(err, "Failed to evaluate 'when' expresion '%s': %v", when, err)
	}
	boolRes, ok := result.(bool)
	if !ok {
		return false, errors.Errorf(errors.CodeBadRequest, "Expected boolean evaluation for '%s'. Got %v", when, result)
	}
	return boolRes, nil
}

// resolveReferences replaces any references to outputs of previous steps, or artifacts in the inputs
// NOTE: by now, input parameters should have been substituted throughout the template, so we only
// are concerned with:
// 1) dereferencing output.parameters from previous steps
// 2) dereferencing output.result from previous steps
// 3) dereferencing output.exitCode from previous steps
// 4) dereferencing artifacts from previous steps
// 5) dereferencing artifacts from inputs
func (woc *wfOperationCtx) resolveReferences(stepGroup []wfv1.WorkflowStep, scope *wfScope) ([]wfv1.WorkflowStep, error) {
	newStepGroup := make([]wfv1.WorkflowStep, len(stepGroup))

	// Step 0: replace all parameter scope references for volumes
	err := woc.substituteParamsInVolumes(scope.getParameters())
	if err != nil {
		return nil, err
	}

	for i, step := range stepGroup {
		// Step 1: replace all parameter scope references in the step
		// TODO: improve this
		stepBytes, err := json.Marshal(step)
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}
		newStepStr, err := template.Replace(string(stepBytes), woc.globalParams.Merge(scope.getParameters()), true)
		if err != nil {
			return nil, err
		}
		var newStep wfv1.WorkflowStep
		err = json.Unmarshal([]byte(newStepStr), &newStep)
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}

		// If we are not executing, don't attempt to resolve any artifact references. We only check if we are executing after
		// the initial parameter resolution, since it's likely that the "when" clause will contain parameter references.
		proceed, err := shouldExecute(newStep.When)
		if err != nil {
			// If we got an error, it might be because our "when" clause contains a task-expansion parameter (e.g. {{item}}).
			// Since we don't perform task-expansion until later and task-expansion parameters won't get resolved here,
			// we continue execution as normal
			if newStep.ShouldExpand() {
				proceed = true
			} else {
				return nil, err
			}
		}
		if !proceed {
			// We can simply return this WorkflowStep; the fact that it won't execute will be reconciled later on in execution
			newStepGroup[i] = newStep
			continue
		}

		// Step 2: replace all artifact references
		for j, art := range newStep.Arguments.Artifacts {
			if art.From == "" && art.FromExpression == "" {
				continue
			}

			resolvedArt, err := scope.resolveArtifact(&art)
			if err != nil {
				if art.Optional {
					continue
				}
				return nil, fmt.Errorf("unable to resolve references: %s", err)
			}
			resolvedArt.Name = art.Name
			newStep.Arguments.Artifacts[j] = *resolvedArt
		}

		newStepGroup[i] = newStep
	}
	return newStepGroup, nil
}

// expandStepGroup looks at each step in a collection of parallel steps, and expands all steps using withItems/withParam
func (woc *wfOperationCtx) expandStepGroup(sgNodeName string, stepGroup []wfv1.WorkflowStep, stepsCtx *stepsContext) ([]wfv1.WorkflowStep, error) {
	newStepGroup := make([]wfv1.WorkflowStep, 0)
	for _, step := range stepGroup {
		if !step.ShouldExpand() {
			newStepGroup = append(newStepGroup, step)
			continue
		}
		expandedStep, err := woc.expandStep(step)
		if err != nil {
			return nil, err
		}
		if len(expandedStep) == 0 {
			// Empty list
			childNodeName := fmt.Sprintf("%s.%s", sgNodeName, step.Name)
			if woc.wf.GetNodeByName(childNodeName) == nil {
				stepTemplateScope := stepsCtx.tmplCtx.GetTemplateScope()
				skipReason := "Skipped, empty params"
				woc.log.Infof("Skipping %s: %s", childNodeName, skipReason)
				woc.initializeNode(childNodeName, wfv1.NodeTypeSkipped, stepTemplateScope, &step, stepsCtx.boundaryID, wfv1.NodeSkipped, skipReason)
				woc.addChildNode(sgNodeName, childNodeName)
			}
		}
		newStepGroup = append(newStepGroup, expandedStep...)
	}
	return newStepGroup, nil
}

// expandStep expands a step containing withItems or withParams into multiple parallel steps
func (woc *wfOperationCtx) expandStep(step wfv1.WorkflowStep) ([]wfv1.WorkflowStep, error) {
	var err error
	expandedStep := make([]wfv1.WorkflowStep, 0)
	var items []wfv1.Item
	if len(step.WithItems) > 0 {
		items = step.WithItems
	} else if step.WithParam != "" {
		err = json.Unmarshal([]byte(step.WithParam), &items)
		if err != nil {
			return nil, errors.Errorf(errors.CodeBadRequest, "withParam value could not be parsed as a JSON list: %s", strings.TrimSpace(step.WithParam))
		}
	} else if step.WithSequence != nil {
		items, err = expandSequence(step.WithSequence)
		if err != nil {
			return nil, err
		}
	} else {
		// this should have been prevented in expandStepGroup()
		return nil, errors.InternalError("expandStep() was called with withItems and withParam empty")
	}

	// these fields can be very large (>100m) and marshalling 10k x 100m = 6GB of memory used and
	// very poor performance, so we just nil them out
	step.WithItems = nil
	step.WithParam = ""
	step.WithSequence = nil

	stepBytes, err := json.Marshal(step)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	t, err := template.NewTemplate(string(stepBytes))
	if err != nil {
		return nil, fmt.Errorf("unable to parse argo variable: %w", err)
	}

	for i, item := range items {
		var newStep wfv1.WorkflowStep
		newStepName, err := processItem(t, step.Name, i, item, &newStep)
		if err != nil {
			return nil, err
		}
		newStep.Name = newStepName
		newStep.Template = step.Template
		expandedStep = append(expandedStep, newStep)
	}
	return expandedStep, nil
}

func (woc *wfOperationCtx) prepareMetricScope(node *wfv1.NodeStatus) (map[string]string, map[string]func() float64) {
	realTimeScope := make(map[string]func() float64)
	localScope := woc.globalParams.DeepCopy()

	if node.Fulfilled() {
		localScope[common.LocalVarDuration] = fmt.Sprintf("%f", node.FinishedAt.Sub(node.StartedAt.Time).Seconds())
		realTimeScope[common.LocalVarDuration] = func() float64 {
			return node.FinishedAt.Sub(node.StartedAt.Time).Seconds()
		}
	} else {
		localScope[common.LocalVarDuration] = fmt.Sprintf("%f", time.Since(node.StartedAt.Time).Seconds())
		realTimeScope[common.LocalVarDuration] = func() float64 {
			return time.Since(node.StartedAt.Time).Seconds()
		}
	}

	if node.Phase != "" {
		localScope[common.LocalVarStatus] = string(node.Phase)
	}

	if node.Inputs != nil {
		for _, param := range node.Inputs.Parameters {
			key := fmt.Sprintf("inputs.parameters.%s", param.Name)
			localScope[key] = param.Value.String()
		}
	}

	if node.Outputs != nil {
		if node.Outputs.Result != nil {
			localScope["outputs.result"] = *node.Outputs.Result
		}
		if node.Outputs.ExitCode != nil {
			localScope[common.LocalVarExitCode] = *node.Outputs.ExitCode
		}
		for _, param := range node.Outputs.Parameters {
			key := fmt.Sprintf("outputs.parameters.%s", param.Name)
			localScope[key] = param.Value.String()
		}
	}

	if node.ResourcesDuration != nil {
		for name, duration := range node.ResourcesDuration {
			localScope[fmt.Sprintf("%s.%s", common.LocalVarResourcesDuration, name)] = fmt.Sprint(duration.Duration().Seconds())
		}
	}

	return localScope, realTimeScope
}
