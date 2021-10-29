package controller

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func (woc *wfOperationCtx) executeHTTPTemplate(nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) *wfv1.NodeStatus {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypeHTTP, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
		woc.taskSet[node.ID] = *tmpl
	}
	woc.runTemplateExecutorPlugins(tmpl, node)
	return node
}

func (woc *wfOperationCtx) httpReconciliation(ctx context.Context) {
	err := woc.reconcileTaskSet(ctx)
	if err != nil {
		woc.log.WithError(err).Error("error in workflowtaskset reconciliation")
		return
	}

	err = woc.reconcileAgentPod(ctx)
	if err != nil {
		woc.log.WithError(err).Error("error in agent pod reconciliation")
		woc.markWorkflowError(ctx, err)
		return
	}
}

func (woc *wfOperationCtx) nodeRequiresHttpReconciliation(nodeName string) bool {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		return false
	}
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
