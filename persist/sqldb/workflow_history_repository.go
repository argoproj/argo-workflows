package sqldb

import (
	"upper.io/db.v3/lib/sqlbuilder"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type WorkflowHistoryRepository interface {
	ListWorkflowHistory() ([]wfv1.Workflow, error)
}

type workflowHistoryRepository struct {
	sqlbuilder.Database
}

func NewWorkflowHistoryRepository(database sqlbuilder.Database) WorkflowHistoryRepository {
	return &workflowHistoryRepository{Database: database}
}

func (r *workflowHistoryRepository) ListWorkflowHistory() ([]wfv1.Workflow, error) {
	var wfDBs []WorkflowDB
	err := r.Collection("workflow_history").Find().OrderBy("-startedat").All(&wfDBs)
	return wfDB2wf(wfDBs), err
}
