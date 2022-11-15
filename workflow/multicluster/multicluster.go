package multicluster

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type MultiClusterProcessor interface {
	ProcessWorkflow(wf *wfv1.Workflow) error
	ProcessWorkflowDeletion(wf *wfv1.Workflow) error
}
