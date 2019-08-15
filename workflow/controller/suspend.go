package controller

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/templateresolution"
)

func (woc *wfOperationCtx) executeSuspend(nodeName string, tmplCtx *templateresolution.Context, tmpl *wfv1.Template, orgTmpl wfv1.TemplateHolder, boundaryID string) *wfv1.NodeStatus {
	node := woc.getNodeByName(nodeName)
	if node == nil {
		node = woc.initializeNode(nodeName, wfv1.NodeTypeSuspend, tmplCtx, tmpl, orgTmpl, boundaryID, wfv1.NodeRunning)
	} else if node.CanRerun() {
		node = woc.markNodePhase(nodeName, wfv1.NodeRunning)
	} else {
		return node
	}

	woc.log.Infof("node %s suspended", node)
	return node
}
