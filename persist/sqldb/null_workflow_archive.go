package sqldb

import (
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

func (r *nullWorkflowArchive) ArchiveWorkflow(*wfv1.Workflow) error {
	return nil
}

func (r *nullWorkflowArchive) ListWorkflows(options sutils.ListOptions) (wfv1.Workflows, error) {
	return wfv1.Workflows{}, nil
}

func (r *nullWorkflowArchive) CountWorkflows(options sutils.ListOptions) (int64, error) {
	return 0, nil
}

func (r *nullWorkflowArchive) GetWorkflow(string, string, string) (*wfv1.Workflow, error) {
	return nil, fmt.Errorf("getting archived workflows not supported")
}

func (r *nullWorkflowArchive) GetWorkflowForEstimator(namespace string, requirements []labels.Requirement) (*wfv1.Workflow, error) {
	return nil, fmt.Errorf("getting archived workflow for estimator not supported")
}

func (r *nullWorkflowArchive) DeleteWorkflow(string) error {
	return fmt.Errorf("deleting archived workflows not supported")
}

func (r *nullWorkflowArchive) DeleteExpiredWorkflows(time.Duration) error {
	return nil
}

func (r *nullWorkflowArchive) ListWorkflowsLabelKeys() (*wfv1.LabelKeys, error) {
	return &wfv1.LabelKeys{}, nil
}

func (r *nullWorkflowArchive) ListWorkflowsLabelValues(string) (*wfv1.LabelValues, error) {
	return &wfv1.LabelValues{}, nil
}
