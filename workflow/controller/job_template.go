package controller

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func (woc *wfOperationCtx) executeJobTemplate(ctx context.Context, jobName string, templateScope string, tmpl *wfv1.Template, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) (*wfv1.NodeStatus, error) {
	node := woc.wf.GetNodeByName(jobName)
	if node == nil {
		node = woc.initializeExecutableNode(jobName, wfv1.NodeTypePod, templateScope, tmpl, orgTmpl, opts.boundaryID, wfv1.NodePending)
	}

	job := tmpl.Job
	steps := job.Steps
	for i, step := range steps {
		nodeName := fmt.Sprintf("%s.%s", jobName, step.Name)
		stepNode := woc.wf.GetNodeByName(nodeName)
		if stepNode == nil {
			_ = woc.initializeNode(nodeName, wfv1.NodeTypeJobStep, templateScope, orgTmpl, node.ID, wfv1.NodePending)
		}
		if i == 0 {
			woc.addChildNode(jobName, nodeName)
		} else {
			previousStep := steps[i-1]
			woc.addChildNode(fmt.Sprintf("%s.%s", jobName, previousStep.Name), nodeName)
		}
	}

	_, err := woc.createWorkflowPod(ctx, jobName, job.GetContainers(), tmpl, &createWorkflowPodOpts{
		onExitPod:         opts.onExitTemplate,
		executionDeadline: opts.executionDeadline,
	})

	if err != nil {
		return woc.requeueIfTransientErr(err, node.Name)
	}
	return node, nil
}
