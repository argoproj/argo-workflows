package sqldb

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
)

var NullWorkflowArchive WorkflowArchive = &nullWorkflowArchive{}

type nullWorkflowArchive struct{}

func (r *nullWorkflowArchive) IsEnabled() bool {
	return false
}

func (r *nullWorkflowArchive) ArchiveWorkflow(ctx context.Context, wf *wfv1.Workflow) error {
	return nil
}

func (r *nullWorkflowArchive) ListWorkflows(ctx context.Context, options sutils.ListOptions) (wfv1.Workflows, error) {
	return wfv1.Workflows{}, nil
}

func (r *nullWorkflowArchive) CountWorkflows(ctx context.Context, options sutils.ListOptions) (int64, error) {
	return 0, nil
}

func (r *nullWorkflowArchive) GetWorkflow(ctx context.Context, uid string, namespace string, name string) (*wfv1.Workflow, error) {
	return nil, fmt.Errorf("getting archived workflows not supported")
}

func (r *nullWorkflowArchive) GetWorkflowForEstimator(ctx context.Context, namespace string, requirements []labels.Requirement) (*wfv1.Workflow, error) {
	return nil, fmt.Errorf("getting archived workflow for estimator not supported")
}

func (r *nullWorkflowArchive) DeleteWorkflow(ctx context.Context, uid string) error {
	return fmt.Errorf("deleting archived workflows not supported")
}

func (r *nullWorkflowArchive) DeleteExpiredWorkflows(ctx context.Context, ttl time.Duration) error {
	return nil
}

func (r *nullWorkflowArchive) ListWorkflowsLabelKeys(ctx context.Context) (*wfv1.LabelKeys, error) {
	return &wfv1.LabelKeys{}, nil
}

func (r *nullWorkflowArchive) ListWorkflowsLabelValues(ctx context.Context, key string) (*wfv1.LabelValues, error) {
	return &wfv1.LabelValues{}, nil
}
