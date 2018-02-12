package controller

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func (woc *wfOperationCtx) executeSuspend(nodeName string, tmpl *wfv1.Template, boundaryID string) *wfv1.NodeStatus {
	node := woc.getNodeByName(nodeName)
	if node == nil {
		node = woc.initializeNode(nodeName, wfv1.NodeTypeSuspend, tmpl.Name, boundaryID, wfv1.NodeRunning)
	}
	woc.log.Infof("node %s suspended", node)
	return node
}
