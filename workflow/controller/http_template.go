package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func (woc *wfOperationCtx) executeHTTPTemplate(nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) *wfv1.NodeStatus {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypeHTTP, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
		woc.taskSet[node.ID] = *tmpl
	}
	return node
}

func (woc *wfOperationCtx) getCompletedHTTPNodes() []wfv1.NodeStatus {
	var nodeStatus []wfv1.NodeStatus
	for _, node := range woc.wf.Status.Nodes {
		if node.Type == wfv1.NodeTypeHTTP && node.Fulfilled() {
			nodeStatus = append(nodeStatus, node)
		}
	}
	return nodeStatus
}