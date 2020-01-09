package sqldb

import (
	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

const tableName = "argo_archived_workflows"

type WorkflowArchive interface {
	ArchiveWorkflow(wf *wfv1.Workflow) error
	ListWorkflows(namespace string, limit, offset int) (wfv1.Workflows, error)
	GetWorkflow(uid string) (*wfv1.Workflow, error)
	DeleteWorkflow(uid string) error
}

type workflowArchive struct {
	sqlbuilder.Database
}

func NewWorkflowArchive(database sqlbuilder.Database) WorkflowArchive {
	return &workflowArchive{Database: database}
}

func (r *workflowArchive) ArchiveWorkflow(wf *wfv1.Workflow) error {
	err := r.DeleteWorkflow(string(wf.UID))
	if err != nil {
		return err
	}
	wfDB, err := toRecord(wf)
	if err != nil {
		return err
	}
	_, err = r.Collection(tableName).Insert(wfDB)
	return err
}

func (r *workflowArchive) ListWorkflows(namespace string, limit int, offset int) (wfv1.Workflows, error) {
	var wfMDs []WorkflowMetadata
	err := r.
		Select("name", "namespace", "id", "phase", "startedat", "finishedat").
		From(tableName).
		Where(namespaceEqual(namespace)).
		OrderBy("-startedat").
		Limit(limit).
		Offset(offset).
		All(&wfMDs)
	if err != nil {
		return nil, err
	}
	wfs := toSlimWorkflows(wfMDs)
	return wfs, nil
}

func namespaceEqual(namespace string) db.Cond {
	if namespace == "" {
		return db.Cond{}
	} else {
		return db.Cond{"namespace": namespace}
	}
}

func (r *workflowArchive) GetWorkflow(uid string) (*wfv1.Workflow, error) {
	rs := r.Collection(tableName).
		Find().
		Where(db.Cond{"id": uid})
	exists, err := rs.Exists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	workflow := &WorkflowOnlyRecord{}
	err = rs.One(workflow)
	if err != nil {
		return nil, err
	}
	return toWorkflow(workflow)
}

func (r *workflowArchive) DeleteWorkflow(uid string) error {
	return r.Collection(tableName).
		Find().
		Where(db.Cond{"id": uid}).
		Delete()
}
