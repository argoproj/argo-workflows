package controller

import (
	"context"
	"fmt"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func (woc *wfOperationCtx) executeJobTemplate(ctx context.Context, nodeName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node := woc.wf.GetNodeByName(nodeName)
	if node == nil {
		node = woc.initializeExecutableNode(nodeName, wfv1.NodeTypePod, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	}

	job := tmpl.Job
	steps := job.Steps
	for i, step := range steps {
		stepNodeName := fmt.Sprintf("%s.%s", nodeName, step.Name)
		stepNode := woc.wf.GetNodeByName(stepNodeName)
		if stepNode == nil {
			_ = woc.initializeNode(stepNodeName, wfv1.NodeTypeJobStep, "", &wfv1.NodeStatus{}, node.ID, wfv1.NodePending)
		}
		if i == 0 {
			woc.addChildNode(nodeName, stepNodeName)
		} else {
			previousStep := steps[i-1]
			woc.addChildNode(fmt.Sprintf("%s.%s", nodeName, previousStep.Name), stepNodeName)
		}
	}

	_, err := woc.createWorkflowPod(ctx, nodeName, job.GetContainers(), tmpl, &createWorkflowPodOpts{
		onExitPod:         opts.onExitTemplate,
		executionDeadline: opts.executionDeadline,
	})

	if err != nil {
		return woc.requeueIfTransientErr(err, node.Name)
	}
	return node, nil
}
