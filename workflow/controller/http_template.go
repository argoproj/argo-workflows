package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func (woc *wfOperationCtx) executeHTTPTemplate(nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) *wfv1.NodeStatus {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypeHTTP, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending, opts.nodeFlag)
	}
	if !node.Fulfilled() {
		woc.taskSet[node.ID] = *tmpl
	}
	return node
}
