package sqldb

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var NullWorkflowHistoryRepository = &nullWorkflowHistoryRepository{}

type nullWorkflowHistoryRepository struct {
}

func (r *nullWorkflowHistoryRepository) AddWorkflowHistory(wf *wfv1.Workflow) error {
	return nil
}

func (r *nullWorkflowHistoryRepository) ListWorkflowHistory() ([]wfv1.Workflow, error) {
	return []wfv1.Workflow{}, nil
}
