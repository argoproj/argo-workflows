package controller

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

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
			_ = woc.killDeamonedChildren(node.ID)
		}
	}()
	stepsCtx := stepsContext{
		boundaryID: node.ID,
		scope: &wfScope{
			tmpl:  tmpl,
			scope: make(map[string]interface{}),
		},
	}
	for i, stepGroup := range tmpl.Steps {
		sgNodeName := fmt.Sprintf("%s[%d]", nodeName, i)
		sgNode := woc.getNodeByName(sgNodeName)
		if sgNode == nil {
			// initialize the step group
			sgNode = woc.initializeNode(sgNodeName, wfv1.NodeTypeStepGroup, "", stepsCtx.boundaryID, wfv1.NodeRunning)
			if i == 0 {
				// Connect the boundary node with the first step group
				woc.addChildNode(nodeName, sgNodeName)
				node = woc.getNodeByName(nodeName)
			} else {
				// Otherwise connect all the outbound nodes of the previous
				// step group as parents to the current step group node
				prevStepGroupName := fmt.Sprintf("%s[%d]", nodeName, i-1)
				prevStepGroupNode := woc.getNodeByName(prevStepGroupName)
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
			failMessage := fmt.Sprintf("step group %s was unsuccessful: %s", sgNode, sgNode.Message)
			woc.log.Info(failMessage)
			woc.updateOutboundNodes(nodeName, tmpl)
			return woc.markNodePhase(nodeName, wfv1.NodeFailed, sgNode.Message)
		}

		for _, step := range stepGroup {
			childNodeName := fmt.Sprintf("%s.%s", sgNodeName, step.Name)
			childNodeID := woc.wf.NodeID(childNodeName)
			childNode, ok := woc.wf.Status.Nodes[childNodeID]
			if !ok {
				// This can happen if there was `withItem` expansion
				// it is okay to ignore this because these expanded steps
				// are not easily referenceable by user.
				continue
			}
			prefix := fmt.Sprintf("steps.%s", step.Name)
			woc.processNodeOutputs(stepsCtx.scope, prefix, &childNode)
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

	// Kick off all parallel steps in the group
	for _, step := range stepGroup {
		childNodeName := fmt.Sprintf("%s.%s", sgNodeName, step.Name)

		// Check the step's when clause to decide if it should execute
		proceed, err := shouldExecute(step.When)
		if err != nil {
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
				errMsg := fmt.Sprintf("child '%s' errored", childNode)
				woc.log.Infof("Step group node %s deemed errored due to child %s error: %s", node, childNodeName, err.Error())
				woc.addChildNode(sgNodeName, childNodeName)
				return woc.markNodePhase(node.Name, wfv1.NodeError, errMsg)
			}
		}
		if childNode != nil {
			woc.addChildNode(sgNodeName, childNodeName)
			if childNode.Completed() && !childNode.Successful() {
				break
			}
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
		if !childNode.Successful() {
			failMessage := fmt.Sprintf("child '%s' failed", childNodeID)
			woc.log.Infof("Step group node %s deemed failed: %s", node, failMessage)
			return woc.markNodePhase(node.Name, wfv1.NodeFailed, failMessage)
		}
	}
	woc.log.Infof("Step group node %v successful", node)
	return woc.markNodePhase(node.Name, wfv1.NodeSucceeded)
}

var whenExpression = regexp.MustCompile("^(.*)(==|!=)(.*)$")

// shouldExecute evaluates a already substituted when expression to decide whether or not a step should execute
func shouldExecute(when string) (bool, error) {
	if when == "" {
		return true, nil
	}
	parts := whenExpression.FindStringSubmatch(when)
	if len(parts) == 0 {
		return false, errors.Errorf(errors.CodeBadRequest, "Invalid 'when' expression: %s", when)
	}
	var1 := strings.TrimSpace(parts[1])
	operator := parts[2]
	var2 := strings.TrimSpace(parts[3])
	switch operator {
	case "==":
		return var1 == var2, nil
	case "!=":
		return var1 != var2, nil
	default:
		return false, errors.Errorf(errors.CodeBadRequest, "Unknown operator: %s", operator)
	}
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

	for i, step := range stepGroup {
		// Step 1: replace all parameter scope references in the step
		// TODO: improve this
		stepBytes, err := json.Marshal(step)
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}
		replaceMap := make(map[string]string)
		for key, val := range scope.scope {
			valStr, ok := val.(string)
			if ok {
				replaceMap[key] = valStr
			}
		}
		fstTmpl := fasttemplate.New(string(stepBytes), "{{", "}}")
		newStepStr, err := common.Replace(fstTmpl, replaceMap, true)
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
		if len(step.WithItems) == 0 && step.WithParam == "" {
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
	} else {
		// this should have been prevented in expandStepGroup()
		return nil, errors.InternalError("expandStep() was called with withItems and withParam empty")
	}

	for i, item := range items {
		replaceMap := make(map[string]string)
		var newStepName string
		switch val := item.Value.(type) {
		case string, int32, int64, float32, float64, bool:
			replaceMap["item"] = fmt.Sprintf("%v", val)
			newStepName = fmt.Sprintf("%s(%d:%v)", step.Name, i, val)
		case map[string]interface{}:
			// Handle the case when withItems is a list of maps.
			// vals holds stringified versions of the map items which are incorporated as part of the step name.
			// For example if the item is: {"name": "jesse","group":"developer"}
			// the vals would be: ["name:jesse", "group:developer"]
			// This would eventually be part of the step name (group:developer,name:jesse)
			vals := make([]string, 0)
			for itemKey, itemValIf := range val {
				switch itemVal := itemValIf.(type) {
				case string, int32, int64, float32, float64, bool:
					replaceMap[fmt.Sprintf("item.%s", itemKey)] = fmt.Sprintf("%v", itemVal)
					vals = append(vals, fmt.Sprintf("%s:%s", itemKey, itemVal))
				default:
					return nil, errors.Errorf(errors.CodeBadRequest, "withItems[%d][%s] expected string or number. received: %s", i, itemKey, itemVal)
				}
			}
			// sort the values so that the name is deterministic
			sort.Strings(vals)
			newStepName = fmt.Sprintf("%s(%d:%v)", step.Name, i, strings.Join(vals, ","))
		default:
			return nil, errors.Errorf(errors.CodeBadRequest, "withItems[%d] expected string, number, or map. received: %s", i, val)
		}
		newStepStr, err := common.Replace(fstTmpl, replaceMap, false)
		if err != nil {
			return nil, err
		}
		var newStep wfv1.WorkflowStep
		err = json.Unmarshal([]byte(newStepStr), &newStep)
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}
		newStep.Name = newStepName
		expandedStep = append(expandedStep, newStep)
	}
	return expandedStep, nil
}
