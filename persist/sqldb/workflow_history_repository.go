package sqldb

import (
	"upper.io/db.v3/lib/sqlbuilder"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

const WorkflowHistoryTableName = "argo_workflow_history"

type WorkflowHistoryRepository interface {
	AddWorkflowHistory(wf *wfv1.Workflow) error
	ListWorkflowHistory() ([]wfv1.Workflow, error)
}

type workflowHistoryRepository struct {
	sqlbuilder.Database
}

func NewWorkflowHistoryRepository(database sqlbuilder.Database) WorkflowHistoryRepository {
	return &workflowHistoryRepository{Database: database}
}

func (r *workflowHistoryRepository) AddWorkflowHistory(wf *wfv1.Workflow) error {
	// TODO upsert
	wfDB, err := convert(wf)
	if err != nil {
		return err
	}
	_, err = r.Collection(WorkflowHistoryTableName).Insert(wfDB)
	return err
}

func (r *workflowHistoryRepository) ListWorkflowHistory() ([]wfv1.Workflow, error) {
	var wfDBs []WorkflowDB
	err := r.Collection(WorkflowHistoryTableName).Find().OrderBy("-startedat").All(&wfDBs)
	return wfDB2wf(wfDBs), err
}
