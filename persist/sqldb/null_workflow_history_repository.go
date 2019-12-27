package sqldb

import (
	"fmt"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var NullWorkflowHistoryRepository = &nullWorkflowHistoryRepository{}

type nullWorkflowHistoryRepository struct {
}

func (r *nullWorkflowHistoryRepository) AddWorkflowHistory(*wfv1.Workflow) error {
	return nil
}

func (r *nullWorkflowHistoryRepository) ListWorkflowHistory(int, int) ([]wfv1.Workflow, error) {
	return []wfv1.Workflow{}, nil
}

func (r *nullWorkflowHistoryRepository) GetWorkflowHistory(string, string) (*wfv1.Workflow, error) {
	return nil, fmt.Errorf("this should not be possible")
}
