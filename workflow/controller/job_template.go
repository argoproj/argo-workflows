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
	nodeName := func(stepName string) string { return fmt.Sprintf("%s.%s", jobName, stepName) }

	for i, s := range steps {
		stepName := nodeName(s.Name)
		stepNode := woc.wf.GetNodeByName(stepName)
		if stepNode == nil {
			_ = woc.initializeNode(stepName, wfv1.NodeTypeJobStep, templateScope, orgTmpl, node.ID, wfv1.NodePending)
		}
		if i == 0 {
			woc.addChildNode(jobName, stepName)
		} else {
			woc.addChildNode(nodeName(steps[i-1].Name), stepName)
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
