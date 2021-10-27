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

func (woc *wfOperationCtx) nodeRequiresHttpReconciliation(nodeName string) bool {
	node := woc.wf.GetNodeByName(nodeName)
	// If this node is of type HTTP, it will need an HTTP reconciliation
	if node.Type == wfv1.NodeTypeHTTP {
		return true
	}
	for _, child := range node.Children {
		// If any of the node's children need an HTTP reconciliation, the parent node will also need one
		if woc.nodeRequiresHttpReconciliation(child) {
			return true
		}
	}
	// If neither of the children need one -- or if there are no children -- no HTTP reconciliation is needed.
	return false
}