package controller

import (
	"fmt"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	"github.com/argoproj/argo/errors"
	log "github.com/sirupsen/logrus"
)

// operateWorkflow is the operator logic of a workflow
// It evaluates the current state of the workflow and decides how to proceed down the execution path
func (wfc *WorkflowController) operateWorkflow(wf *wfv1.Workflow) {
	if wf.Completed() {
		return
	}
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	wfCopy := wf.DeepCopyObject().(*wfv1.Workflow)
	updated := false

	defer func() {
		if updated {
			_, err := wfc.WorkflowClient.UpdateWorkflow(wfCopy)
			if err != nil {
				log.Errorf("ERROR updating status: %v", err)
			} else {
				log.Infof("UPDATED %s: %#v", wfCopy.ObjectMeta.Name, wfCopy.Status)
			}
		}
	}()
	if wfCopy.Status.Nodes == nil {
		wfCopy.Status.Nodes = make(map[string]wfv1.NodeStatus)
		updated = true
	}

	tmplUpdates, err := wfc.executeTemplate(wfCopy, wfCopy.Spec.Entrypoint, nil, wfCopy.ObjectMeta.Name)
	updated = updated || tmplUpdates
	if err != nil {
		log.Errorf("%s error: %+v", wf.ObjectMeta.Name, err)
	}
}

// Returns tuple of: (workflow was updated, error)
func (wfc *WorkflowController) executeTemplate(wf *wfv1.Workflow, templateName string, args *wfv1.Arguments, nodeName string) (bool, error) {
	log.Infof("Evaluating node %s: %v, args: %#v", nodeName, templateName, args)
	nodeID := wf.NodeID(nodeName)
	node, ok := wf.Status.Nodes[nodeID]
	if ok && node.Completed() {
		log.Infof("Node %s already completed", nodeName)
		return false, nil
	}
	tmpl := wf.GetTemplate(templateName)
	if tmpl == nil {
		err := errors.Errorf(errors.CodeBadRequest, "Node %s error: template '%s' undefined", nodeName, templateName)
		wf.Status.Nodes[nodeID] = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: wfv1.NodeStatusError}
		return true, err
	}

	switch tmpl.Type {
	case wfv1.TypeContainer:
		if !ok {
			// We have not yet created the pod
			status := wfv1.NodeStatusRunning
			err := wfc.createWorkflowPod(wf, nodeName, tmpl, args)
			if err != nil {
				// TODO: may need to query pod status if we hit already exists error
				status = wfv1.NodeStatusError
				return false, err
			}
			node = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: status}
			wf.Status.Nodes[nodeID] = node
			log.Infof("Initialized container node %v", node)
			return true, nil
		}
		return false, nil

	case wfv1.TypeWorkflow:
		updates := false
		if !ok {
			node = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: wfv1.NodeStatusRunning}
			log.Infof("Initialized workflow node %v", node)
			wf.Status.Nodes[nodeID] = node
			updates = true
		}
		for i, stepGroup := range tmpl.Steps {
			sgNodeName := fmt.Sprintf("%s[%d]", nodeName, i)
			sgUpdates, err := wfc.executeStepGroup(wf, stepGroup, sgNodeName)
			if err != nil {
				node.Status = wfv1.NodeStatusError
				wf.Status.Nodes[nodeID] = node
				return true, err
			}
			updates = updates || sgUpdates
			sgNodeID := wf.NodeID(sgNodeName)
			if !wf.Status.Nodes[sgNodeID].Completed() {
				log.Infof("Workflow step group node %v not yet completed", wf.Status.Nodes[sgNodeID])
				return updates, nil
			}
			if !wf.Status.Nodes[sgNodeID].Successful() {
				log.Infof("Workflow step group %v not successful", wf.Status.Nodes[sgNodeID])
				node.Status = wfv1.NodeStatusFailed
				wf.Status.Nodes[nodeID] = node
				return true, nil
			}
		}
		node.Status = wfv1.NodeStatusSucceeded
		wf.Status.Nodes[nodeID] = node
		return true, nil

	default:
		wf.Status.Nodes[nodeID] = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: wfv1.NodeStatusError}
		return true, fmt.Errorf("Unknown type: %s", tmpl.Type)
	}
}

func (wfc *WorkflowController) executeStepGroup(wf *wfv1.Workflow, stepGroup map[string]wfv1.WorkflowStep, nodeName string) (bool, error) {
	nodeID := wf.NodeID(nodeName)
	node, ok := wf.Status.Nodes[nodeID]
	if ok && node.Completed() {
		log.Infof("Step group node %v already marked completed", node)
		return false, nil
	}
	updates := false
	if !ok {
		node = wfv1.NodeStatus{ID: nodeID, Name: nodeName, Status: "Running"}
		wf.Status.Nodes[nodeID] = node
		log.Infof("Initializing step group node %v", node)
		updates = true
	}
	childNodeIDs := make([]string, 0)
	// First kick off all parallel steps in the group
	for stepName, step := range stepGroup {
		childNodeName := fmt.Sprintf("%s.%s", nodeName, stepName)
		childNodeIDs = append(childNodeIDs, wf.NodeID(childNodeName))
		sUpdates, err := wfc.executeTemplate(wf, step.Template, &step.Arguments, childNodeName)
		updates = updates || sUpdates
		if err != nil {
			node.Status = wfv1.NodeStatusError
			wf.Status.Nodes[nodeID] = node
			return true, err
		}
	}
	// Return if not all children completed
	for _, childNodeID := range childNodeIDs {
		if !wf.Status.Nodes[childNodeID].Completed() {
			return updates, nil
		}
	}
	// All children completed. Determine status
	for _, childNodeID := range childNodeIDs {
		if !wf.Status.Nodes[childNodeID].Successful() {
			node.Status = wfv1.NodeStatusFailed
			wf.Status.Nodes[nodeID] = node
			updates = true
			log.Infof("Step group node %s deemed failed due to failure of %s", nodeID, childNodeID)
			return updates, nil
		}
	}
	node.Status = wfv1.NodeStatusSucceeded
	wf.Status.Nodes[nodeID] = node
	log.Infof("Step group node %s successful", nodeID)
	return true, nil
}
