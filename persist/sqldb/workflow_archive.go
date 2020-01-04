package sqldb

import (
	"encoding/json"

	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

const tableName = "argo_archived_workflows"

type WorkflowArchive interface {
	ArchiveWorkflow(wf *wfv1.Workflow) error
	ListWorkflows(namespace string, limit, offset int) ([]wfv1.Workflow, error)
	GetWorkflow(namespace string, uid string) (*wfv1.Workflow, error)
	DeleteWorkflow(namespace string, uid string) error
}

type workflowArchive struct {
	sqlbuilder.Database
}

func NewWorkflowArchive(database sqlbuilder.Database) WorkflowArchive {
	return &workflowArchive{Database: database}
}

func (r *workflowArchive) ArchiveWorkflow(wf *wfv1.Workflow) error {
	err := r.DeleteWorkflow(wf.Namespace, string(wf.UID))
	if err != nil {
		return err
	}
	wfDB, err := convert(wf)
	if err != nil {
		return err
	}
	_, err = r.Collection(tableName).Insert(wfDB)
	return err
}

func (r *workflowArchive) ListWorkflows(namespace string, limit int, offset int) ([]wfv1.Workflow, error) {
	var wfDBs []WorkflowDB
	err := r.Collection(tableName).
		Find().
		Where(namespaceEqual(namespace)).
		OrderBy("-startedat").
		Limit(limit).
		Offset(offset).
		All(&wfDBs)
	return wfDB2wf(wfDBs), err
}

func namespaceEqual(namespace string) db.Cond {
	if namespace == "" {
		return db.Cond{}
	} else {
		return db.Cond{"namespace": namespace}
	}
}

func (r *workflowArchive) GetWorkflow(namespace string, uid string) (*wfv1.Workflow, error) {
	rs := r.Collection(tableName).
		Find().
		Where(db.Cond{"id": uid}).
		And(db.Cond{"namespace": namespace})
	exists, err := rs.Exists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	wfDB := &WorkflowDB{}
	err = rs.One(wfDB)
	if err != nil {
		return nil, err
	}
	wf := &wfv1.Workflow{}
	err = json.Unmarshal([]byte(wfDB.Workflow), wf)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (r *workflowArchive) DeleteWorkflow(namespace string, uid string) error {
	return r.Collection(tableName).
		Find().
		Where(db.Cond{"id": uid}).
		And(db.Cond{"namespace": namespace}).
		Delete()
}
