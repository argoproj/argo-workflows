package controller

import (
	"encoding/json"
	"fmt"
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
				woc.log.Errorf("ERROR updating status: %v", err)
			} else {
				woc.log.Infof("UPDATED: %#v", woc.wf.Status)
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

	switch tmpl.Type {
	case wfv1.TypeContainer:
		if ok {
			// There's already a node entry for the container. This means the container was already
			// scheduled (or had a create pod error). Nothing to more to do with this node.
			return nil
		}
		// We have not yet created the pod
		status := wfv1.NodeStatusRunning
		err := woc.createWorkflowPod(nodeName, tmpl, args)
		if err != nil {
			// TODO: may need to query pod status if we hit already exists error
			status = wfv1.NodeStatusError
		}
		node = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: status}
		woc.wf.Status.Nodes[nodeID] = node
		woc.log.Infof("Initialized container node %v", node)
		woc.updated = true
		return err

	case wfv1.TypeWorkflow:
		if !ok {
			node = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: wfv1.NodeStatusRunning}
			woc.log.Infof("Initialized workflow node %v", node)
			woc.wf.Status.Nodes[nodeID] = node
			woc.updated = true
		}
		for i, stepGroup := range tmpl.Steps {
			sgNodeName := fmt.Sprintf("%s[%d]", nodeName, i)
			err := woc.executeStepGroup(stepGroup, sgNodeName)
			if err != nil {
				node.Status = wfv1.NodeStatusError
				woc.wf.Status.Nodes[nodeID] = node
				woc.updated = true
				return err
			}
			sgNodeID := woc.wf.NodeID(sgNodeName)
			if !woc.wf.Status.Nodes[sgNodeID].Completed() {
				woc.log.Infof("Workflow step group node %v not yet completed", woc.wf.Status.Nodes[sgNodeID])
				return nil
			}
			if !woc.wf.Status.Nodes[sgNodeID].Successful() {
				woc.log.Infof("Workflow step group %v not successful", woc.wf.Status.Nodes[sgNodeID])
				node.Status = wfv1.NodeStatusFailed
				woc.wf.Status.Nodes[nodeID] = node
				woc.updated = true
				return nil
			}
		}
		node.Status = wfv1.NodeStatusSucceeded
		woc.wf.Status.Nodes[nodeID] = node
		woc.updated = true
		return nil

	default:
		woc.wf.Status.Nodes[nodeID] = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: wfv1.NodeStatusError}
		woc.updated = true
		return errors.Errorf("Unknown type: %s", tmpl.Type)
	}
}

// markNodeError marks a node with the given status, creating the node if necessary
func (woc *wfOperationCtx) markNodeStatus(nodeName string, status string) {
	nodeID := woc.wf.NodeID(nodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if !ok {
		node = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: status}
	} else {
		node.Status = status
	}
	woc.wf.Status.Nodes[nodeID] = node
	woc.updated = true
}

func (woc *wfOperationCtx) executeStepGroup(stepGroup map[string]wfv1.WorkflowStep, nodeName string) error {
	nodeID := woc.wf.NodeID(nodeName)
	node, ok := woc.wf.Status.Nodes[nodeID]
	if ok && node.Completed() {
		woc.log.Infof("Step group node %v already marked completed", node)
		return nil
	}
	if !ok {
		node = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: "Running"}
		woc.wf.Status.Nodes[nodeID] = node
		woc.log.Infof("Initializing step group node %v", node)
		woc.updated = true
	}
	stepGroup, err := woc.expandStepGroup(stepGroup)
	if err != nil {
		woc.markNodeStatus(nodeName, wfv1.NodeStatusError)
		return err
	}

	childNodeIDs := make([]string, 0)
	// First kick off all parallel steps in the group
	for stepName, step := range stepGroup {
		childNodeName := fmt.Sprintf("%s.%s", nodeName, stepName)
		childNodeIDs = append(childNodeIDs, woc.wf.NodeID(childNodeName))
		err := woc.executeTemplate(step.Template, step.Arguments, childNodeName)
		if err != nil {
			node.Status = wfv1.NodeStatusError
			woc.wf.Status.Nodes[nodeID] = node
			woc.updated = true
			return err
		}
	}
	// Return if not all children completed
	for _, childNodeID := range childNodeIDs {
		if !woc.wf.Status.Nodes[childNodeID].Completed() {
			return nil
		}
	}
	// All children completed. Determine status
	for _, childNodeID := range childNodeIDs {
		if !woc.wf.Status.Nodes[childNodeID].Successful() {
			node.Status = wfv1.NodeStatusFailed
			woc.wf.Status.Nodes[nodeID] = node
			woc.updated = true
			woc.log.Infof("Step group node %s deemed failed due to failure of %s", nodeID, childNodeID)
			return nil
		}
	}
	woc.markNodeStatus(node.Name, wfv1.NodeStatusSucceeded)
	woc.log.Infof("Step group node %s successful", nodeID)
	return nil
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
	for _, item := range step.WithItems {
		switch val := item.(type) {
		case string:
			replaceMap := map[string]interface{}{
				"item": val,
			}
			newStepStr := fstTmpl.ExecuteString(replaceMap)
			var newStep wfv1.WorkflowStep
			err = json.Unmarshal([]byte(newStepStr), &newStep)
			if err != nil {
				return nil, errors.InternalWrapError(err)
			}
			newStepName := fmt.Sprintf("%s(%s)", stepName, val)
			expandedStep[newStepName] = newStep
		}
	}
	return expandedStep, nil
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
