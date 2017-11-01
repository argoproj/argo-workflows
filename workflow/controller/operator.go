package controller

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasttemplate"
)

// wfOperationCtx is the context for evaluation and operation of a single workflow
type wfOperationCtx struct {
	// wf is the workflow object
	wf *wfv1.Workflow
	// updated indicates whether or not the workflow object itself was updated
	// and needs to be persisted back to kubernetes
	updated bool
	// log is an logrus logging context to corrolate logs with a workflow
	log *log.Entry
	// controller reference to workflow controller
	controller *WorkflowController
	// NOTE: eventually we may need to store additional metadata state to
	// understand how to proceed in workflows with more complex control flows.
	// (e.g. workflow failed in step 1 of 3 but has finalizer steps)
}

// wfScope contains the current scope of variables available when iterating steps in a workflow
type wfScope struct {
	scope map[string]interface{}
}

// operateWorkflow is the operator logic of a workflow
// It evaluates the current state of the workflow and decides how to proceed down the execution path
func (wfc *WorkflowController) operateWorkflow(wf *wfv1.Workflow) {
	if wf.Completed() {
		return
	}
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance

	woc := wfOperationCtx{
		wf:      wf.DeepCopyObject().(*wfv1.Workflow),
		updated: false,
		log: log.WithFields(log.Fields{
			"workflow":  wf.ObjectMeta.Name,
			"namespace": wf.ObjectMeta.Namespace,
		}),
		controller: wfc,
	}
	defer func() {
		if woc.updated {
			_, err := wfc.WorkflowClient.UpdateWorkflow(woc.wf)
			if err != nil {
				woc.log.Errorf("Error updating %s status: %v", woc.wf.ObjectMeta.SelfLink, err)
			} else {
				woc.log.Infof("Workflow %s updated", woc.wf.ObjectMeta.SelfLink)
			}
		}
	}()
	if woc.wf.Status.Nodes == nil {
		woc.wf.Status.Nodes = make(map[string]wfv1.NodeStatus)
		woc.updated = true
	}

	err := woc.executeTemplate(wf.Spec.Entrypoint, wf.Spec.Arguments, wf.ObjectMeta.Name)
	if err != nil {
		woc.log.Errorf("%s error: %+v", wf.ObjectMeta.Name, err)
	}
}

func (woc *wfOperationCtx) executeTemplate(templateName string, args wfv1.Arguments, nodeName string) error {
	woc.log.Infof("Evaluating node %s: %v, args: %#v", nodeName, templateName, args)
	nodeID := woc.wf.NodeID(nodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if ok && node.Completed() {
		woc.log.Infof("Node %s already completed", nodeName)
		return nil
	}
	tmpl := woc.wf.GetTemplate(templateName)
	if tmpl == nil {
		err := errors.Errorf(errors.CodeBadRequest, "Node %s error: template '%s' undefined", nodeName, templateName)
		woc.markNodeStatus(nodeName, wfv1.NodeStatusError)
		return err
	}
	if len(args) > 0 {
		var err error
		tmpl, err = substituteArgs(tmpl, args)
		if err != nil {
			woc.markNodeStatus(nodeName, wfv1.NodeStatusError)
			return err
		}
	}

	if tmpl.Container != nil {
		if ok {
			// There's already a node entry for the container. This means the container was already
			// scheduled (or had a create pod error). Nothing to more to do with this node.
			return nil
		}
		// We have not yet created the pod
		return woc.executeContainer(nodeName, tmpl)

	} else if len(tmpl.Steps) > 0 {
		if !ok {
			node = *woc.markNodeStatus(nodeName, wfv1.NodeStatusRunning)
			woc.log.Infof("Initialized workflow node %v", node)
		}
		return woc.executeSteps(nodeName, tmpl.Steps)

	} else if tmpl.Script != nil {
		return woc.executeScript(nodeName, tmpl)
	}

	woc.markNodeStatus(nodeName, wfv1.NodeStatusError)
	return errors.Errorf("Template '%s' missing specification", tmpl.Name)
}

// markNodeError marks a node with the given status, creating the node if necessary
func (woc *wfOperationCtx) markNodeStatus(nodeName string, status string) *wfv1.NodeStatus {
	nodeID := woc.wf.NodeID(nodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if !ok {
		node = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: status}
	} else {
		node.Status = status
	}
	woc.wf.Status.Nodes[nodeID] = node
	woc.updated = true
	return &node
}

func (woc *wfOperationCtx) executeContainer(nodeName string, tmpl *wfv1.Template) error {
	err := woc.createWorkflowPod(nodeName, tmpl)
	if err != nil {
		// TODO: may need to query pod status if we hit already exists error
		woc.markNodeStatus(nodeName, wfv1.NodeStatusError)
		return err
	}
	node := woc.markNodeStatus(nodeName, wfv1.NodeStatusRunning)
	woc.log.Infof("Initialized container node %v", node)
	return nil
}

func (woc *wfOperationCtx) executeSteps(nodeName string, steps []map[string]wfv1.WorkflowStep) error {
	var scope wfScope
	for i, stepGroup := range steps {
		sgNodeName := fmt.Sprintf("%s[%d]", nodeName, i)
		err := woc.executeStepGroup(stepGroup, sgNodeName, &scope)
		if err != nil {
			woc.markNodeStatus(nodeName, wfv1.NodeStatusError)
			return err
		}
		sgNodeID := woc.wf.NodeID(sgNodeName)
		if !woc.wf.Status.Nodes[sgNodeID].Completed() {
			woc.log.Infof("Workflow step group node %v not yet completed", woc.wf.Status.Nodes[sgNodeID])
			return nil
		}
		if !woc.wf.Status.Nodes[sgNodeID].Successful() {
			woc.log.Infof("Workflow step group %v not successful", woc.wf.Status.Nodes[sgNodeID])
			woc.markNodeStatus(nodeName, wfv1.NodeStatusFailed)
			return nil
		}
	}
	woc.markNodeStatus(nodeName, wfv1.NodeStatusSucceeded)
	return nil
}

// executeStepGroup examines a map of parallel workflows steps and executes them in parallel.
// It first expands any `withItem` clauses, then evaluates any `when` expressions for each step
// to decide if execution is required.
func (woc *wfOperationCtx) executeStepGroup(stepGroup map[string]wfv1.WorkflowStep, nodeName string, scope *wfScope) error {
	nodeID := woc.wf.NodeID(nodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if ok && node.Completed() {
		woc.log.Infof("Step group node %v already marked completed", node)
		return nil
	}
	if !ok {
		node = *woc.markNodeStatus(nodeName, wfv1.NodeStatusRunning)
		woc.log.Infof("Initializing step group node %v", node)
	}
	stepGroup, err := woc.expandStepGroup(stepGroup)
	if err != nil {
		woc.markNodeStatus(nodeName, wfv1.NodeStatusError)
		return err
	}

	nodeIDtoStepName := make(map[string]string)

	childNodeIDs := make([]string, 0)
	// First kick off all parallel steps in the group
	for stepName, step := range stepGroup {
		childNodeName := fmt.Sprintf("%s.%s", nodeName, stepName)
		childNodeIDs = append(childNodeIDs, woc.wf.NodeID(childNodeName))

		// Check the step's when clause to decide if it should execute
		proceed, err := shouldExecute(step.When)
		if err != nil {
			woc.markNodeStatus(nodeName, wfv1.NodeStatusError)
			return err
		}
		if !proceed {
			woc.markNodeStatus(nodeName, wfv1.NodeStatusSkipped)
			continue
		}
		err = woc.executeTemplate(step.Template, step.Arguments, childNodeName)
		if err != nil {
			woc.markNodeStatus(nodeName, wfv1.NodeStatusError)
			return err
		}
	}
	// Return if not all children completed
	for _, childNodeID := range childNodeIDs {
		if !woc.wf.Status.Nodes[childNodeID].Completed() {
			return nil
		}
	}
	// All children completed. Determine step group status as a whole, and add any outputs to the scope
	for _, childNodeID := range childNodeIDs {
		childNode := woc.wf.Status.Nodes[childNodeID]
		if !childNode.Successful() {
			woc.markNodeStatus(nodeName, wfv1.NodeStatusFailed)
			woc.log.Infof("Step group node %s deemed failed due to failure of %s", nodeID, childNodeID)
			return nil
		}
		stepName := nodeIDtoStepName[childNodeID]
		for outName, outParam := range childNode.Outputs.Parameters {
			key := fmt.Sprintf("steps.%s.outputs.parameters.%s", stepName, outName)
			scope.addParamToScope(key, outParam.Value)
		}
		for outName, outArt := range childNode.Outputs.Artifacts {
			key := fmt.Sprintf("steps.%s.outputs.artifacts.%s", stepName, outName)
			scope.addArtifactToScope(key, outArt)
		}
	}
	woc.markNodeStatus(node.Name, wfv1.NodeStatusSucceeded)
	woc.log.Infof("Step group node %s successful", nodeID)
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

func (woc *wfOperationCtx) expandStepGroup(stepGroup map[string]wfv1.WorkflowStep) (map[string]wfv1.WorkflowStep, error) {
	newStepGroup := make(map[string]wfv1.WorkflowStep)
	for stepName, step := range stepGroup {
		if len(step.WithItems) == 0 {
			newStepGroup[stepName] = step
			continue
		}
		expandedStep, err := woc.expandStep(stepName, step)
		if err != nil {
			return nil, err
		}
		for newStepName, newStep := range expandedStep {
			newStepGroup[newStepName] = newStep
		}
	}
	return newStepGroup, nil
}

func (woc *wfOperationCtx) expandStep(stepName string, step wfv1.WorkflowStep) (map[string]wfv1.WorkflowStep, error) {
	stepBytes, err := json.Marshal(step)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	fstTmpl := fasttemplate.New(string(stepBytes), "{{", "}}")

	expandedStep := make(map[string]wfv1.WorkflowStep)
	for i, item := range step.WithItems {
		replaceMap := make(map[string]interface{})
		var newStepName string
		switch val := item.(type) {
		case string:
			replaceMap["item"] = val
			newStepName = fmt.Sprintf("%s(%s)", stepName, val)
		case map[string]interface{}:
			// Handle the case when withItems is a list of maps.
			// vals holds stringified versions of the map items which are incorporated as part of the step name.
			// For example if the item is: {"name": "jesse","group":"developer"}
			// the vals would be: ["name:jesse", "group:developer"]
			// This would eventually be part of the step name (group:developer,name:jesse)
			vals := make([]string, 0)
			for itemKey, itemValIf := range val {
				itemVal, ok := itemValIf.(string)
				if !ok {
					return nil, errors.Errorf(errors.CodeBadRequest, "withItems[%d][%s] expected string. received: %s", i, itemKey, itemVal)
				}
				replaceMap[fmt.Sprintf("item.%s", itemKey)] = itemVal
				vals = append(vals, fmt.Sprintf("%s:%s", itemKey, itemVal))
			}
			// sort the values so that the name is deterministic
			sort.Strings(vals)
			newStepName = fmt.Sprintf("%s(%s)", stepName, strings.Join(vals, ","))
		default:
			return nil, errors.Errorf(errors.CodeBadRequest, "withItems[%d] expected string or map. received: %s", i, val)
		}
		newStepStr := fstTmpl.ExecuteString(replaceMap)
		var newStep wfv1.WorkflowStep
		err = json.Unmarshal([]byte(newStepStr), &newStep)
		if err != nil {
			return nil, errors.InternalWrapError(err)
		}
		expandedStep[newStepName] = newStep
	}
	return expandedStep, nil
}

func (woc *wfOperationCtx) executeScript(nodeName string, tmpl *wfv1.Template) error {
	err := woc.createWorkflowPod(nodeName, tmpl)
	if err != nil {
		// TODO: may need to query pod status if we hit already exists error
		woc.markNodeStatus(nodeName, wfv1.NodeStatusError)
		return err
	}
	node := woc.markNodeStatus(nodeName, wfv1.NodeStatusRunning)
	woc.log.Infof("Initialized container node %v", node)
	return nil
}

// substituteArgs returns a new copy of the template with all input parameters substituted
func substituteArgs(tmpl *wfv1.Template, args wfv1.Arguments) (*wfv1.Template, error) {
	tmplBytes, err := json.Marshal(tmpl)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	fstTmpl := fasttemplate.New(string(tmplBytes), "{{", "}}")
	replaceMap := make(map[string]interface{})
	for argName, argVal := range args {
		if strings.HasPrefix(argName, "parameters.") {
			_, ok := argVal.(string)
			if !ok {
				return nil, errors.Errorf("argument '%s' expected to be string. received: %s", argName, argVal)
			}
			replaceMap["inputs."+argName] = argVal
		}
	}
	s := fstTmpl.ExecuteString(replaceMap)

	var newTmpl wfv1.Template
	err = json.Unmarshal([]byte(s), &newTmpl)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &newTmpl, nil
}

func (wfs *wfScope) addParamToScope(key, val string) {
	wfs.scope[key] = val
}

func (wfs *wfScope) addArtifactToScope(key string, artifact wfv1.OutputArtifact) {
	wfs.scope[key] = artifact
}

func (wfs *wfScope) resolveVar(v string) (interface{}, error) {
	val, ok := wfs.scope[v]
	if !ok {
		return nil, errors.Errorf("Unable to resolve: {{%s}}", v)
	}
	return val, nil
}

func (wfs *wfScope) resolveStringVar(v string) (string, error) {
	val, err := wfs.resolveVar(v)
	if err != nil {
		return "", err
	}
	valStr, ok := val.(string)
	if !ok {
		return "", errors.Errorf("Variable {{%s}} is not a string", v)
	}
	return valStr, nil
}
