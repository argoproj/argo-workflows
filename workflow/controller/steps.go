package controller

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/valyala/fasttemplate"
)

// stepsContext holds context information about this context's steps
type stepsContext struct {
	// boundaryID is the node ID of the boundary which all immediate child steps are bound to
	boundaryID string

	// scope holds parameter and artifacts which are referenceable in scope during execution
	scope *wfScope
}

func (woc *wfOperationCtx) executeSteps(nodeName string, tmpl *wfv1.Template, boundaryID string) *wfv1.NodeStatus {
	node := woc.getNodeByName(nodeName)
	if node == nil {
		node = woc.initializeNode(nodeName, wfv1.NodeTypeSteps, tmpl.Name, boundaryID, wfv1.NodeRunning)
	}
	defer func() {
		if woc.wf.Status.Nodes[node.ID].Completed() {
			_ = woc.killDaemonedChildren(node.ID)
		}
	}()
	stepsCtx := stepsContext{
		boundaryID: node.ID,
		scope: &wfScope{
			tmpl:  tmpl,
			scope: make(map[string]interface{}),
		},
	}
	woc.addOutputsToScope("workflow", woc.wf.Status.Outputs, stepsCtx.scope)

	for i, stepGroup := range tmpl.Steps {
		sgNodeName := fmt.Sprintf("%s[%d]", nodeName, i)
		sgNode := woc.getNodeByName(sgNodeName)
		if sgNode == nil {
			sgNode = woc.initializeNode(sgNodeName, wfv1.NodeTypeStepGroup, "", stepsCtx.boundaryID, wfv1.NodeRunning)
		}
		// The following will connect the step group node to its parents.
		if i == 0 {
			// If this is the first step group, the boundary node is the parent
			woc.addChildNode(nodeName, sgNodeName)
			node = woc.getNodeByName(nodeName)
		} else {
			// Otherwise connect all the outbound nodes of the previous step group as parents to
			// the current step group node.
			prevStepGroupName := fmt.Sprintf("%s[%d]", nodeName, i-1)
			prevStepGroupNode := woc.getNodeByName(prevStepGroupName)
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
		sgNode = woc.executeStepGroup(stepGroup, sgNodeName, &stepsCtx)
		if !sgNode.Completed() {
			woc.log.Infof("Workflow step group node %v not yet completed", sgNode)
			return node
		}

		if !sgNode.Successful() {
			failMessage := fmt.Sprintf("step group %s was unsuccessful: %s", sgNode.ID, sgNode.Message)
			woc.log.Info(failMessage)
			woc.updateOutboundNodes(nodeName, tmpl)
			return woc.markNodePhase(nodeName, wfv1.NodeFailed, sgNode.Message)
		}

		// Add all outputs of each step in the group to the scope
		for _, step := range stepGroup {
			childNodeName := fmt.Sprintf("%s.%s", sgNodeName, step.Name)
			childNode := woc.getNodeByName(childNodeName)
			prefix := fmt.Sprintf("steps.%s", step.Name)
			if childNode == nil {
				// This happens when there was `withItem/withParam` expansion.
				// We add the aggregate outputs of our children to the scope as a JSON list
				var childNodes []wfv1.NodeStatus
				for _, node := range woc.wf.Status.Nodes {
					if node.BoundaryID == stepsCtx.boundaryID && strings.HasPrefix(node.Name, childNodeName+"(") {
						childNodes = append(childNodes, node)
					}
				}
				woc.processAggregateNodeOutputs(step.Template, stepsCtx.scope, prefix, childNodes)
			} else {
				woc.processNodeOutputs(stepsCtx.scope, prefix, childNode)
			}
		}
	}
	woc.updateOutboundNodes(nodeName, tmpl)
	// If this template has outputs from any of its steps, copy them to this node here
	outputs, err := getTemplateOutputsFromScope(tmpl, stepsCtx.scope)
	if err != nil {
		return woc.markNodeError(nodeName, err)
	}
	if outputs != nil {
		node = woc.getNodeByName(nodeName)
		node.Outputs = outputs
		woc.wf.Status.Nodes[node.ID] = *node
	}

	return woc.markNodePhase(nodeName, wfv1.NodeSucceeded)
}

// updateOutboundNodes set the outbound nodes from the last step group
func (woc *wfOperationCtx) updateOutboundNodes(nodeName string, tmpl *wfv1.Template) {
	outbound := make([]string, 0)
	// Find the last, initialized stepgroup node
	var lastSGNode *wfv1.NodeStatus
	for i := len(tmpl.Steps) - 1; i >= 0; i-- {
		sgNode := woc.getNodeByName(fmt.Sprintf("%s[%d]", nodeName, i))
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
		for _, outNodeID := range outboundNodeIDs {
			outbound = append(outbound, outNodeID)
		}
	}
	node := woc.getNodeByName(nodeName)
	woc.log.Infof("Outbound nodes of %s is %s", node.ID, outbound)
	node.OutboundNodes = outbound
	woc.wf.Status.Nodes[node.ID] = *node
}

// executeStepGroup examines a list of parallel steps and executes them in parallel.
// Handles referencing of variables in scope, expands `withItem` clauses, and evaluates `when` expressions
func (woc *wfOperationCtx) executeStepGroup(stepGroup []wfv1.WorkflowStep, sgNodeName string, stepsCtx *stepsContext) *wfv1.NodeStatus {
	node := woc.getNodeByName(sgNodeName)
	if node.Completed() {
		woc.log.Debugf("Step group node %v already marked completed", node)
		return node
	}

	// First, resolve any references to outputs from previous steps, and perform substitution
	stepGroup, err := woc.resolveReferences(stepGroup, stepsCtx.scope)
	if err != nil {
		return woc.markNodeError(sgNodeName, err)
	}

	// Next, expand the step's withItems (if any)
	stepGroup, err = woc.expandStepGroup(stepGroup)
	if err != nil {
		return woc.markNodeError(sgNodeName, err)
	}

	// Maps nodes to their steps
	nodeSteps := make(map[string]wfv1.WorkflowStep)

	// Kick off all parallel steps in the group
	for _, step := range stepGroup {
		childNodeName := fmt.Sprintf("%s.%s", sgNodeName, step.Name)

		// Check the step's when clause to decide if it should execute
		proceed, err := shouldExecute(step.When)
		if err != nil {
			woc.initializeNode(childNodeName, wfv1.NodeTypeSkipped, "", stepsCtx.boundaryID, wfv1.NodeError, err.Error())
			woc.addChildNode(sgNodeName, childNodeName)
			woc.markNodeError(childNodeName, err)
			return woc.markNodeError(sgNodeName, err)
		}
		if !proceed {
			if woc.getNodeByName(childNodeName) == nil {
				skipReason := fmt.Sprintf("when '%s' evaluated false", step.When)
				woc.log.Infof("Skipping %s: %s", childNodeName, skipReason)
				woc.initializeNode(childNodeName, wfv1.NodeTypeSkipped, "", stepsCtx.boundaryID, wfv1.NodeSkipped, skipReason)
				woc.addChildNode(sgNodeName, childNodeName)
			}
			continue
		}
		childNode, err := woc.executeTemplate(step.Template, step.Arguments, childNodeName, stepsCtx.boundaryID)
		if err != nil {
			switch err {
			case ErrDeadlineExceeded:
				return node
			case ErrParallelismReached:
			default:
				errMsg := fmt.Sprintf("child '%s' errored", childNode.ID)
				woc.log.Infof("Step group node %s deemed errored due to child %s error: %s", node, childNodeName, err.Error())
				woc.addChildNode(sgNodeName, childNodeName)
				return woc.markNodePhase(node.Name, wfv1.NodeError, errMsg)
			}
		}
		if childNode != nil {
			nodeSteps[childNodeName] = step
			woc.addChildNode(sgNodeName, childNodeName)
		}
	}

	node = woc.getNodeByName(sgNodeName)
	// Return if not all children completed
	for _, childNodeID := range node.Children {
		if !woc.wf.Status.Nodes[childNodeID].Completed() {
			return node
		}
	}
	// All children completed. Determine step group status as a whole
	for _, childNodeID := range node.Children {
		childNode := woc.wf.Status.Nodes[childNodeID]
		step := nodeSteps[childNode.Name]
		if !childNode.Successful() && !step.ContinuesOn(childNode.Phase) {
			failMessage := fmt.Sprintf("child '%s' failed", childNodeID)
			woc.log.Infof("Step group node %s deemed failed: %s", node, failMessage)
			return woc.markNodePhase(node.Name, wfv1.NodeFailed, failMessage)
		}
	}
	woc.log.Infof("Step group node %v successful", node)
	return woc.markNodePhase(node.Name, wfv1.NodeSucceeded)
}

// shouldExecute evaluates a already substituted when expression to decide whether or not a step should execute
func shouldExecute(when string) (bool, error) {
	if when == "" {
		return true, nil
	}
	expression, err := govaluate.NewEvaluableExpression(when)
	if err != nil {
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
// 2) dereferencing artifacts from previous steps
// 3) dereferencing artifacts from inputs
func (woc *wfOperationCtx) resolveReferences(stepGroup []wfv1.WorkflowStep, scope *wfScope) ([]wfv1.WorkflowStep, error) {
	newStepGroup := make([]wfv1.WorkflowStep, len(stepGroup))

	// Step 0: replace all parameter scope references for volumes
	err := woc.substituteParamsInVolumes(scope.replaceMap())
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
		fstTmpl := fasttemplate.New(string(stepBytes), "{{", "}}")
		newStepStr, err := common.Replace(fstTmpl, scope.replaceMap(), true)
		if err != nil {
			return nil, err
		}
		var newStep wfv1.WorkflowStep
		err = json.Unmarshal([]byte(newStepStr), &newStep)
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}

		// Step 2: replace all artifact references
		for j, art := range newStep.Arguments.Artifacts {
			if art.From == "" {
				continue
			}
			resolvedArt, err := scope.resolveArtifact(art.From)
			if err != nil {
				return nil, err
			}
			resolvedArt.Name = art.Name
			newStep.Arguments.Artifacts[j] = *resolvedArt
		}

		newStepGroup[i] = newStep
	}
	return newStepGroup, nil
}

// expandStepGroup looks at each step in a collection of parallel steps, and expands all steps using withItems/withParam
func (woc *wfOperationCtx) expandStepGroup(stepGroup []wfv1.WorkflowStep) ([]wfv1.WorkflowStep, error) {
	newStepGroup := make([]wfv1.WorkflowStep, 0)
	for _, step := range stepGroup {
		if len(step.WithItems) == 0 && step.WithParam == "" && step.WithSequence == nil {
			newStepGroup = append(newStepGroup, step)
			continue
		}
		expandedStep, err := woc.expandStep(step)
		if err != nil {
			return nil, err
		}
		for _, newStep := range expandedStep {
			newStepGroup = append(newStepGroup, newStep)
		}
	}
	return newStepGroup, nil
}

// expandStep expands a step containing withItems or withParams into multiple parallel steps
func (woc *wfOperationCtx) expandStep(step wfv1.WorkflowStep) ([]wfv1.WorkflowStep, error) {
	stepBytes, err := json.Marshal(step)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	fstTmpl := fasttemplate.New(string(stepBytes), "{{", "}}")
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

	for i, item := range items {
		var newStep wfv1.WorkflowStep
		newStepName, err := processItem(fstTmpl, step.Name, i, item, &newStep)
		if err != nil {
			return nil, err
		}
		newStep.Name = newStepName
		newStep.Template = step.Template
		expandedStep = append(expandedStep, newStep)
	}
	return expandedStep, nil
}
