package controller

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
)

func (woc *wfOperationCtx) executeSteps(nodeName string, tmpl *wfv1.Template) error {
	nodeID := woc.wf.NodeID(nodeName)
	defer func() {
		if woc.wf.Status.Nodes[nodeID].Completed() {
			_ = woc.killDeamonedChildren(nodeID)
		}
	}()
	scope := wfScope{
		tmpl:  tmpl,
		scope: make(map[string]interface{}),
	}
	for i, stepGroup := range tmpl.Steps {
		sgNodeName := fmt.Sprintf("%s[%d]", nodeName, i)
		woc.addChildNode(nodeName, sgNodeName)
		err := woc.executeStepGroup(stepGroup, sgNodeName, &scope)
		if err != nil {
			if errors.IsCode(errors.CodeTimeout, err) {
				return err
			}
			woc.markNodeError(nodeName, err)
			return err
		}
		sgNodeID := woc.wf.NodeID(sgNodeName)
		if !woc.wf.Status.Nodes[sgNodeID].Completed() {
			woc.log.Infof("Workflow step group node %v not yet completed", woc.wf.Status.Nodes[sgNodeID])
			return nil
		}

		if !woc.wf.Status.Nodes[sgNodeID].Successful() {
			failMessage := fmt.Sprintf("step group %s was unsuccessful", sgNodeName)
			woc.log.Info(failMessage)
			woc.markNodePhase(nodeName, wfv1.NodeFailed, failMessage)
			return nil
		}

		// HACK: need better way to add children to scope
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
			if childNode.PodIP != "" {
				key := fmt.Sprintf("steps.%s.ip", step.Name)
				scope.addParamToScope(key, childNode.PodIP)
			}
			if childNode.Outputs != nil {
				if childNode.Outputs.Result != nil {
					key := fmt.Sprintf("steps.%s.outputs.result", step.Name)
					scope.addParamToScope(key, *childNode.Outputs.Result)
				}
				for _, outParam := range childNode.Outputs.Parameters {
					key := fmt.Sprintf("steps.%s.outputs.parameters.%s", step.Name, outParam.Name)
					scope.addParamToScope(key, *outParam.Value)
				}
				for _, outArt := range childNode.Outputs.Artifacts {
					key := fmt.Sprintf("steps.%s.outputs.artifacts.%s", step.Name, outArt.Name)
					scope.addArtifactToScope(key, outArt)
				}
			}
		}
	}
	outputs, err := getTemplateOutputsFromScope(tmpl, &scope)
	if err != nil {
		woc.markNodeError(nodeName, err)
		return err
	}
	if outputs != nil {
		node := woc.wf.Status.Nodes[nodeID]
		node.Outputs = outputs
		woc.wf.Status.Nodes[nodeID] = node
	}
	woc.markNodePhase(nodeName, wfv1.NodeSucceeded)
	return nil
}

// executeStepGroup examines a map of parallel steps and executes them in parallel.
// Handles referencing of variables in scope, expands `withItem` clauses, and evaluates `when` expressions
func (woc *wfOperationCtx) executeStepGroup(stepGroup []wfv1.WorkflowStep, sgNodeName string, scope *wfScope) error {
	nodeID := woc.wf.NodeID(sgNodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if ok && node.Completed() {
		woc.log.Debugf("Step group node %v already marked completed", node)
		return nil
	}
	if !ok {
		node = *woc.markNodePhase(sgNodeName, wfv1.NodeRunning)
		woc.log.Infof("Initializing step group node %v", node)
	}

	// First, resolve any references to outputs from previous steps, and perform substitution
	stepGroup, err := woc.resolveReferences(stepGroup, scope)
	if err != nil {
		woc.markNodeError(sgNodeName, err)
		return err
	}

	// Next, expand the step's withItems (if any)
	stepGroup, err = woc.expandStepGroup(stepGroup)
	if err != nil {
		woc.markNodeError(sgNodeName, err)
		return err
	}

	// Kick off all parallel steps in the group
	for _, step := range stepGroup {
		childNodeName := fmt.Sprintf("%s.%s", sgNodeName, step.Name)
		woc.addChildNode(sgNodeName, childNodeName)

		// Check the step's when clause to decide if it should execute
		proceed, err := shouldExecute(step.When)
		if err != nil {
			woc.markNodeError(childNodeName, err)
			woc.markNodeError(sgNodeName, err)
			return err
		}
		if !proceed {
			skipReason := fmt.Sprintf("when '%s' evaluated false", step.When)
			woc.log.Infof("Skipping %s: %s", childNodeName, skipReason)
			woc.markNodePhase(childNodeName, wfv1.NodeSkipped, skipReason)
			continue
		}
		err = woc.executeTemplate(step.Template, step.Arguments, childNodeName)
		if err != nil {
			if !errors.IsCode(errors.CodeTimeout, err) {
				woc.markNodeError(childNodeName, err)
				woc.markNodeError(sgNodeName, err)
			}
			return err
		}
	}

	node = woc.wf.Status.Nodes[nodeID]
	// Return if not all children completed
	for _, childNodeID := range node.Children {
		if !woc.wf.Status.Nodes[childNodeID].Completed() {
			return nil
		}
	}
	// All children completed. Determine step group status as a whole
	for _, childNodeID := range node.Children {
		childNode := woc.wf.Status.Nodes[childNodeID]
		if !childNode.Successful() {
			failMessage := fmt.Sprintf("child '%s' failed", childNodeID)
			woc.markNodePhase(sgNodeName, wfv1.NodeFailed, failMessage)
			woc.log.Infof("Step group node %s deemed failed: %s", childNode, failMessage)
			return nil
		}
	}
	woc.markNodePhase(node.Name, wfv1.NodeSucceeded)
	woc.log.Infof("Step group node %v successful", woc.wf.Status.Nodes[nodeID])
	return nil
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
		newStepStr, err := common.Replace(fstTmpl, replaceMap, true, "")
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
			return nil, errors.Errorf(errors.CodeBadRequest, "withParam value not be parsed as a JSON list: %s", step.WithParam)
		}
	} else {
		// this should have been prevented in expandStepGroup()
		return nil, errors.InternalError("expandStep() was called with withItems and withParam empty")
	}

	for i, item := range items {
		replaceMap := make(map[string]string)
		var newStepName string
		switch val := item.(type) {
		case string, int32, int64, float32, float64:
			replaceMap["item"] = fmt.Sprintf("%v", val)
			newStepName = fmt.Sprintf("%s(%v)", step.Name, val)
		case map[string]interface{}:
			// Handle the case when withItems is a list of maps.
			// vals holds stringified versions of the map items which are incorporated as part of the step name.
			// For example if the item is: {"name": "jesse","group":"developer"}
			// the vals would be: ["name:jesse", "group:developer"]
			// This would eventually be part of the step name (group:developer,name:jesse)
			vals := make([]string, 0)
			for itemKey, itemValIf := range val {
				switch itemVal := itemValIf.(type) {
				case string, int32, int64, float32, float64:
					replaceMap[fmt.Sprintf("item.%s", itemKey)] = fmt.Sprintf("%v", itemVal)
					vals = append(vals, fmt.Sprintf("%s:%s", itemKey, itemVal))
				default:
					return nil, errors.Errorf(errors.CodeBadRequest, "withItems[%d][%s] expected string or number. received: %s", i, itemKey, itemVal)
				}
			}
			// sort the values so that the name is deterministic
			sort.Strings(vals)
			newStepName = fmt.Sprintf("%s(%v)", step.Name, strings.Join(vals, ","))
		default:
			return nil, errors.Errorf(errors.CodeBadRequest, "withItems[%d] expected string, number, or map. received: %s", i, val)
		}
		newStepStr, err := common.Replace(fstTmpl, replaceMap, false, "")
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

// killDeamonedChildren kill any granchildren of a step template node, which have been daemoned.
// We only need to check grandchildren instead of children because the direct children of a step
// template are actually stepGroups, which are nodes that cannot represent actual containers.
// Returns the first error that occurs (if any)
// TODO(jessesuen): this logic will need to change with DAGs
func (woc *wfOperationCtx) killDeamonedChildren(nodeID string) error {
	woc.log.Infof("Checking deamon children of %s", nodeID)
	var firstErr error
	execCtl := common.ExecutionControl{
		Deadline: &time.Time{},
	}
	for _, childNodeID := range woc.wf.Status.Nodes[nodeID].Children {
		for _, grandChildID := range woc.wf.Status.Nodes[childNodeID].Children {
			gcNode := woc.wf.Status.Nodes[grandChildID]
			if gcNode.Daemoned == nil || !*gcNode.Daemoned {
				continue
			}
			err := woc.updateExecutionControl(gcNode.ID, execCtl)
			if err != nil {
				woc.log.Errorf("Failed to update execution control of %s: %+v", gcNode, err)
				if firstErr == nil {
					firstErr = err
				}
			}
		}
	}
	return firstErr
}

// updateExecutionControl updates the execution control parameters
func (woc *wfOperationCtx) updateExecutionControl(podName string, execCtl common.ExecutionControl) error {
	execCtlBytes, err := json.Marshal(execCtl)
	if err != nil {
		return errors.InternalWrapError(err)
	}

	woc.log.Infof("Updating execution control of %s: %s", podName, execCtlBytes)
	err = common.AddPodAnnotation(
		woc.controller.kubeclientset,
		podName,
		woc.wf.ObjectMeta.Namespace,
		common.AnnotationKeyExecutionControl,
		string(execCtlBytes),
	)
	if err != nil {
		return err
	}

	// Ideally we would simply annotate the pod with the updates and be done with it, allowing
	// the executor to notice the updates naturally via the Downward API annotations volume
	// mounted file. However, updates to the Downward API volumes take a very long time to
	// propagate (minutes). The following code fast-tracks this by signaling the executor
	// using SIGUSR2 that something changed.
	woc.log.Infof("Signalling %s of updates", podName)
	exec, err := common.ExecPodContainer(
		woc.controller.restConfig, woc.wf.ObjectMeta.Namespace, podName,
		common.WaitContainerName, true, true, "sh", "-c", "kill -s USR2 1",
	)
	if err != nil {
		return err
	}
	go func() {
		// This call is necessary to actually send the exec. Since signalling is best effort,
		// it is launched as a goroutine and the error is discarded
		_, _, err = common.GetExecutorOutput(exec)
		if err != nil {
			log.Warnf("Signal command failed: %v", err)
			return
		}
		log.Infof("Signal of %s (%s) successfully issued", podName, common.WaitContainerName)
	}()

	return nil
}
