package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func (woc *wfOperationCtx) executePluginTemplate(nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) *wfv1.NodeStatus {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		if opts.boundaryID == "" {
			woc.log.Warnf("[DEBUG] boundaryID was nil")
		}
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePlugin, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending, opts.nodeFlag, true)
	}
	if !node.Fulfilled() {
		woc.taskSet[node.ID] = *tmpl
	}
	return node
}
