package sqldb

import (
	"encoding/json"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

const tableName = "argo_archived_workflows"

type WorkflowArchive interface {
	ArchiveWorkflow(wf *wfv1.Workflow) error
	ListWorkflows(namespace string, limit, offset int) (wfv1.Workflows, error)
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
	wfs := make(wfv1.Workflows, len(wfMDs))
	for i, wf := range wfMDs {
		wfs[i] = wfv1.Workflow{
			ObjectMeta: v1.ObjectMeta{
				Name:              wf.Name,
				Namespace:         wf.Namespace,
				UID:               types.UID(wf.Id),
				CreationTimestamp: v1.Time{Time: wf.StartedAt},
			},
			Status: wfv1.WorkflowStatus{
				Phase:      wf.Phase,
				StartedAt:  v1.Time{Time: wf.StartedAt},
				FinishedAt: v1.Time{Time: wf.FinishedAt},
			},
		}
	}
	return wfs, nil
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
