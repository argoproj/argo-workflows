package controller

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func (woc *wfOperationCtx) executeContainerSet(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePod, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	}
	includeScriptOutput, err := woc.includeScriptOutput(nodeName, opts.boundaryID)
	if err != nil {
		return node, err
	}

	_, err = woc.createWorkflowPod(ctx, nodeName, tmpl.ContainerSet.GetContainers(), tmpl, &createWorkflowPodOpts{
		includeScriptOutput: includeScriptOutput,
		onExitPod:           opts.onExitTemplate,
		executionDeadline:   opts.executionDeadline,
	})
	if err != nil {
		return woc.requeueIfTransientErr(err, node.Name)
	}

	// we only complete the graph if we actually managed to create the pod,
	// which prevents creating many pending nodes that could never be scheduled
	for _, c := range tmpl.ContainerSet.GetContainers() {
		ctxNodeName := fmt.Sprintf("%s.%s", nodeName, c.Name)
		ctrNode := woc.wf.GetNodeByName(ctxNodeName)
		if ctrNode == nil {
			_ = woc.initializeNode(ctxNodeName, wfv1.NodeTypeContainer, templateScope, orgTmpl, node.ID, wfv1.NodePending)
		}
	}
	for _, c := range tmpl.ContainerSet.GetGraph() {
		ctrNodeName := fmt.Sprintf("%s.%s", nodeName, c.Name)
		if len(c.Dependencies) == 0 {
			woc.addChildNode(nodeName, ctrNodeName)
		}
		for _, v := range c.Dependencies {
			woc.addChildNode(fmt.Sprintf("%s.%s", nodeName, v), ctrNodeName)
		}
	}

	return node, nil
}
