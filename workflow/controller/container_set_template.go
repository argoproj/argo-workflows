package controller

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func (woc *wfOperationCtx) executeContainerSet(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node, err := woc.wf.GetNodeByName(nodeName)
	if err != nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePod, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending, opts.nodeFlag)
	}
	includeScriptOutput, err := woc.includeScriptOutput(nodeName, opts.boundaryID)
	if err != nil {
		return node, err
	}

	// while the node is present, the node.PodName might not be.
	// DAGs are a good example of this.
	// a DAG node will only ever have a PodName when that Pod has been launched.
	podName := ""
	if node.PodName == nil {
		podName = util.GeneratePodName(woc.wf.Name, nodeName, tmpl.Name, node.ID, util.GetWorkflowPodNameVersion(woc.wf))
	} else {
		podName = *node.PodName
	}

	_, err = woc.createWorkflowPod(ctx, nodeName, podName, tmpl.ContainerSet.GetContainers(), tmpl, &createWorkflowPodOpts{
		includeScriptOutput: includeScriptOutput,
		onExitPod:           opts.onExitTemplate,
		executionDeadline:   opts.executionDeadline,
	})
	if err != nil {
		return woc.requeueIfTransientErr(err, node.Name)
	}

	// successfully created a pod, so attach podName to Node
	node.PodName = &podName

	// we only complete the graph if we actually managed to create the pod,
	// which prevents creating many pending nodes that could never be scheduled
	for _, c := range tmpl.ContainerSet.GetContainers() {
		ctxNodeName := fmt.Sprintf("%s.%s", nodeName, c.Name)
		_, err := woc.wf.GetNodeByName(ctxNodeName)
		if err != nil {
			// the podName will be the same for all containers
			_ = woc.initializeNode(ctxNodeName, node.PodName, wfv1.NodeTypeContainer, templateScope, tmpl, orgTmpl, node.ID, wfv1.NodePending, opts.nodeFlag)
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
