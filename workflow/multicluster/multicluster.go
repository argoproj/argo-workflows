package multicluster

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type MultiClusterProcessor interface {
	ProcessWorkflow(ctx context.Context, wf *wfv1.Workflow) error
	ProcessWorkflowDeletion(ctx context.Context, wf *wfv1.Workflow) error
}
