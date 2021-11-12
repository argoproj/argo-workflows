package controller

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
)

func (woc *wfOperationCtx) executePluginTemplate(nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) *wfv1.NodeStatus {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePlugin, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	}
	if err := woc.runNodePreExecutePlugins(tmpl, node); err != nil {
		if errorsutil.IsTransientErr(err) {
			return node
		}
		return woc.markNodeError(nodeName, err)
	}
	if !node.Fulfilled() {
		woc.taskSet[node.ID] = *tmpl
	}
	return node
}
