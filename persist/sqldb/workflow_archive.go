package sqldb

import (
	"encoding/json"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

const archiveTableName = "argo_archived_workflows"

type archivedWorkflowMetadata struct {
	ClusterName string         `db:"clustername"`
	UID         string         `db:"uid"`
	Name        string         `db:"name"`
	Namespace   string         `db:"namespace"`
	Phase       wfv1.NodePhase `db:"phase"`
	StartedAt   time.Time      `db:"startedat"`
	FinishedAt  time.Time      `db:"finishedat"`
}

type archivedWorkflowRecord struct {
	archivedWorkflowMetadata
	Workflow string `db:"workflow"`
}
type WorkflowArchive interface {
	ArchiveWorkflow(wf *wfv1.Workflow) error
	ListWorkflows(namespace string, limit, offset int) (wfv1.Workflows, error)
	GetWorkflow(uid string) (*wfv1.Workflow, error)
	DeleteWorkflow(uid string) error
}

type workflowArchive struct {
	session     sqlbuilder.Database
	clusterName string
}

func NewWorkflowArchive(session sqlbuilder.Database, clusterName string) WorkflowArchive {
	return &workflowArchive{session: session, clusterName: clusterName}
}

func (r *workflowArchive) ArchiveWorkflow(wf *wfv1.Workflow) error {
	log.WithField("uid", wf.UID).Debug("Archiving workflow")
	err := r.DeleteWorkflow(string(wf.UID))
	if err != nil {
		return err
	}
	workflow, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	_, err = r.session.Collection(archiveTableName).
		Insert(&archivedWorkflowRecord{
			archivedWorkflowMetadata: archivedWorkflowMetadata{
				ClusterName: r.clusterName,
				UID:         string(wf.UID),
				Name:        wf.Name,
				Namespace:   wf.Namespace,
				Phase:       wf.Status.Phase,
				StartedAt:   wf.Status.StartedAt.Time,
				FinishedAt:  wf.Status.FinishedAt.Time,
			},
			Workflow: string(workflow),
		})
	return err
}

func (r *workflowArchive) ListWorkflows(namespace string, limit int, offset int) (wfv1.Workflows, error) {
	var archivedWfs []archivedWorkflowMetadata
	err := r.session.
		Select("name", "namespace", "uid", "phase", "startedat", "finishedat").
		From(archiveTableName).
		Where(db.Cond{"clustername": r.clusterName}).
		And(namespaceEqual(namespace)).
		OrderBy("-startedat").
		Limit(limit).
		Offset(offset).
		All(&archivedWfs)
	if err != nil {
		return nil, err
	}
	wfs := make(wfv1.Workflows, len(archivedWfs))
	for i, md := range archivedWfs {
		wfs[i] = wfv1.Workflow{
			ObjectMeta: v1.ObjectMeta{
				Name:              md.Name,
				Namespace:         md.Namespace,
				UID:               types.UID(md.UID),
				CreationTimestamp: v1.Time{Time: md.StartedAt},
			},
			Status: wfv1.WorkflowStatus{
				Phase:      md.Phase,
				StartedAt:  v1.Time{Time: md.StartedAt},
				FinishedAt: v1.Time{Time: md.FinishedAt},
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

func (r *workflowArchive) GetWorkflow(uid string) (*wfv1.Workflow, error) {
	archivedWf := &archivedWorkflowRecord{}
	err := r.session.
		Select("workflow").
		From(archiveTableName).
		Where(db.Cond{"clustername": r.clusterName}).
		And(db.Cond{"uid": uid}).
		One(archivedWf)
	if err != nil {
		if strings.Contains(err.Error(), "no more rows") {
			return nil, nil
		}
		return nil, err
	}
	var wf *wfv1.Workflow
	err = json.Unmarshal([]byte(archivedWf.Workflow), &wf)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (r *workflowArchive) DeleteWorkflow(uid string) error {
	rs, err := r.session.
		DeleteFrom(archiveTableName).
		Where(db.Cond{"clustername": r.clusterName}).
		And(db.Cond{"uid": uid}).
		Exec()
	if err != nil {
		return err
	}
	rowsAffected, err := rs.RowsAffected()
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{"uid": uid, "rowsAffected": rowsAffected}).Debug("Deleted archived workflow")
	return nil
}
