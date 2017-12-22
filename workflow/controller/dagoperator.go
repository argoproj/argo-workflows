package controller

import (
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1alpha1"
)

func (woc *wfOperationCtx) operateDAG() (bool, wfv1.NodePhase) {
	var targetNames []string
	if woc.wf.Spec.Target == "" {
		targetNames = findLeafTargetNames(woc.wf.Spec.Targets)
	} else {
		targetNames = strings.Split(woc.wf.Spec.Target, ",")
	}
	// one time initialization of the DAG nodes
	if len(woc.wf.Status.Nodes) == 0 {
		for _, targetName := range targetNames {
			woc.initializeTarget(targetName)
		}
		woc.updated = true
	}
	// operate on each target
	for _, targetName := range targetNames {
		woc.operateTarget(targetName)
	}
	// return whether or not we completed execution
	for _, depName := range targetNames {
		depNode := woc.getNodeFromTarget(depName)
		if !depNode.Completed() {
			return false, wfv1.NodeRunning
		}
	}
	// if all targets completed, return overall status
	for _, depName := range targetNames {
		depNode := woc.getNodeFromTarget(depName)
		if !depNode.Successful() {
			return true, depNode.Phase
		}
	}
	return true, wfv1.NodeSucceeded
}

// initializeTarget creates the initial node entry for a target and parents
func (woc *wfOperationCtx) initializeTarget(targetName string) {
	nodeName := fmt.Sprintf("%s.%s", woc.wf.ObjectMeta.Name, targetName)
	nodeID := woc.wf.NodeID(nodeName)
	_, ok := woc.wf.Status.Nodes[nodeID]
	if ok {
		return
	}
	_ = woc.markNodePhase(nodeName, wfv1.NodePending)
	target := woc.getTargetByName(targetName)
	for _, depName := range target.Dependencies {
		woc.initializeTarget(depName)
	}
}

func (woc *wfOperationCtx) operateTarget(targetName string) {
	node := woc.getNodeFromTarget(targetName)
	if node.Completed() {
		return
	}
	//woc.log.Infof("Operating target %s", targetName)
	// Check if our dependencies completed. If not, recurse our parents
	// executing them if necessary
	target := woc.getTargetByName(targetName)
	dependenciesCompleted := true
	for _, depName := range target.Dependencies {
		depNode := woc.getNodeFromTarget(depName)
		if depNode.Completed() {
			if !depNode.Successful() {
				woc.log.Infof("Target %s marked %s due to failure of dependency %s", targetName, depNode.Phase, depNode.Name)
				woc.markNodePhase(node.Name, depNode.Phase, fmt.Sprintf("dependency %s %s", depNode.Name, depNode.Phase))
				return
			}
			continue
		}
		dependenciesCompleted = false
		woc.operateTarget(depName)
	}
	if !dependenciesCompleted {
		return
	}

	// All our dependencies were satisifed and successful. It's our turn to run
	if node.Phase == wfv1.NodePending {
		woc.log.Infof("All of target %s dependencies satisfied %s", target.Name, target.Dependencies)
	}
	woc.executeTemplate(target.Template, target.Arguments, node.Name)
}

func (woc *wfOperationCtx) getNodeFromTarget(targetName string) wfv1.NodeStatus {
	nodeName := fmt.Sprintf("%s.%s", woc.wf.ObjectMeta.Name, targetName)
	nodeID := woc.wf.NodeID(nodeName)
	return woc.wf.Status.Nodes[nodeID]
}

func (woc *wfOperationCtx) getTargetByName(targetName string) *wfv1.Target {
	for _, target := range woc.wf.Spec.Targets {
		if target.Name == targetName {
			return &target
		}
	}
	panic("target " + targetName + " does not exist")
}

// findLeafTargetNames finds all target names who no other nodes depend on.
// This is used as the the default list of targets.
func findLeafTargetNames(targets []wfv1.Target) []string {
	targetIsLeaf := make(map[string]bool)
	for _, target := range targets {
		if _, ok := targetIsLeaf[target.Name]; !ok {
			targetIsLeaf[target.Name] = true
		}
		for _, dependency := range target.Dependencies {
			targetIsLeaf[dependency] = false
		}
	}
	leafTargets := make([]string, 0)
	for target, isLeaf := range targetIsLeaf {
		if isLeaf {
			leafTargets = append(leafTargets, target)
		}
	}
	return leafTargets
}
