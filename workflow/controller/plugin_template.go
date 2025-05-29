package controller

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func (woc *wfOperationCtx) executePluginTemplate(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) *wfv1.NodeStatus {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		if opts.boundaryID == "" {
			woc.log.Warnf(ctx, "[DEBUG] boundaryID was nil")
		}
		node = woc.initializeExecutableNode(ctx, nodeName, wfv1.NodeTypePlugin, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending, opts.nodeFlag)
	}
	if !node.Fulfilled() {
		woc.taskSet[node.ID] = *tmpl
	}
	return node
}
