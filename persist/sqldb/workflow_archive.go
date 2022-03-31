package sqldb

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
)

const (
	archiveTableName       = "argo_archived_workflows"
	archiveLabelsTableName = archiveTableName + "_labels"
)

type archivedWorkflowMetadata struct {
	ClusterName string             `db:"clustername"`
	InstanceID  string             `db:"instanceid"`
	UID         string             `db:"uid"`
	Name        string             `db:"name"`
	Namespace   string             `db:"namespace"`
	Phase       wfv1.WorkflowPhase `db:"phase"`
	StartedAt   time.Time          `db:"startedat"`
	FinishedAt  time.Time          `db:"finishedat"`
}

type archivedWorkflowRecord struct {
	archivedWorkflowMetadata
	Workflow string `db:"workflow"`
}

type archivedWorkflowLabelRecord struct {
	ClusterName string `db:"clustername"`
	UID         string `db:"uid"`
	// Why is this called "name" not "key"? Key is an SQL reserved word.
	Key   string `db:"name"`
	Value string `db:"value"`
}

//go:generate mockery --name=WorkflowArchive

type WorkflowArchive interface {
	ArchiveWorkflow(clusterName string, wf *wfv1.Workflow) error
	// list workflows, with the most recently started workflows at the beginning (i.e. index 0 is the most recent)
	ListWorkflows(clusterName, namespace, name, namePrefix string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements, limit, offset int) (wfv1.Workflows, error)
	GetWorkflow(clusterName, uid string) (*wfv1.Workflow, error)
	DeleteWorkflow(clusterName, uid string) error
	DeleteExpiredWorkflows(clusterName string, ttl time.Duration) error
	IsEnabled() bool
	ListWorkflowsLabelKeys() (*wfv1.LabelKeys, error)
	ListWorkflowsLabelValues(key string) (*wfv1.LabelValues, error)
}

type workflowArchive struct {
	session           sqlbuilder.Database
	managedNamespace  string
	instanceIDService instanceid.Service
	dbType            dbType
}

func (r *workflowArchive) IsEnabled() bool {
	return true
}

// NewWorkflowArchive returns a new workflowArchive
func NewWorkflowArchive(session sqlbuilder.Database, managedNamespace string, instanceIDService instanceid.Service) WorkflowArchive {
	return &workflowArchive{session: session, managedNamespace: managedNamespace, instanceIDService: instanceIDService, dbType: dbTypeFor(session)}
}

func (r *workflowArchive) ArchiveWorkflow(clusterName string, wf *wfv1.Workflow) error {
	logCtx := log.WithFields(log.Fields{"uid": wf.UID, "labels": wf.GetLabels()})
	logCtx.Debug("Archiving workflow")
	workflow, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	return r.session.Tx(context.Background(), func(sess sqlbuilder.Tx) error {
		_, err := sess.
			DeleteFrom(archiveTableName).
			Where(r.clusterManagedNamespaceAndInstanceID(clusterName)).
			And(db.Cond{"uid": wf.UID}).
			Exec()
		if err != nil {
			return err
		}
		_, err = sess.Collection(archiveTableName).
			Insert(&archivedWorkflowRecord{
				archivedWorkflowMetadata: archivedWorkflowMetadata{
					ClusterName: clusterName,
					InstanceID:  r.instanceIDService.InstanceID(),
					UID:         string(wf.UID),
					Name:        wf.Name,
					Namespace:   wf.Namespace,
					Phase:       wf.Status.Phase,
					StartedAt:   wf.Status.StartedAt.Time,
					FinishedAt:  wf.Status.FinishedAt.Time,
				},
				Workflow: string(workflow),
			})
		if err != nil {
			return err
		}

		_, err = sess.
			DeleteFrom(archiveLabelsTableName).
			Where(db.Cond{"clustername": clusterName}).
			And(db.Cond{"uid": wf.UID}).
			Exec()
		if err != nil {
			return err
		}
		// insert the labels
		for key, value := range wf.GetLabels() {
			_, err := sess.Collection(archiveLabelsTableName).
				Insert(&archivedWorkflowLabelRecord{
					ClusterName: clusterName,
					UID:         string(wf.UID),
					Key:         key,
					Value:       value,
				})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *workflowArchive) ListWorkflows(clusterName, namespace, name, namePrefix string, minStartedAt, maxStartedAt time.Time, labelRequirements labels.Requirements, limit int, offset int) (wfv1.Workflows, error) {
	var archivedWfs []archivedWorkflowMetadata
	clause, err := labelsClause(r.dbType, labelRequirements)
	if err != nil {
		return nil, err
	}

	// If we were passed 0 as the limit, then we should load all available archived workflows
	// to match the behavior of the `List` operations in the Kubernetes API
	if limit == 0 {
		limit = -1
		offset = -1
	}

	err = r.session.
		Select("name", "namespace", "uid", "phase", "startedat", "finishedat").
		From(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID(clusterName)).
		And(namespaceEqual(namespace)).
		And(nameEqual(name)).
		And(namePrefixClause(namePrefix)).
		And(startedAtClause(minStartedAt, maxStartedAt)).
		And(clause).
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

func (r *workflowArchive) clusterManagedNamespaceAndInstanceID(clusterName string) db.Compound {
	return db.And(
		clusterNameEqual(clusterName),
		namespaceEqual(r.managedNamespace),
		db.Cond{"instanceid": r.instanceIDService.InstanceID()},
	)
}

func startedAtClause(from, to time.Time) db.Compound {
	var conds []db.Compound
	if !from.IsZero() {
		conds = append(conds, db.Cond{"startedat > ": from})
	}
	if !to.IsZero() {
		conds = append(conds, db.Cond{"startedat < ": to})
	}
	return db.And(conds...)
}

func clusterNameEqual(v string) db.Cond {
	if v == "" {
		return db.Cond{}
	} else {
		return db.Cond{"clustername": v}
	}
}

func namespaceEqual(namespace string) db.Cond {
	if namespace == "" {
		return db.Cond{}
	} else {
		return db.Cond{"namespace": namespace}
	}
}

func nameEqual(name string) db.Cond {
	if name == "" {
		return db.Cond{}
	} else {
		return db.Cond{"name": name}
	}
}

func namePrefixClause(namePrefix string) db.Cond {
	if namePrefix == "" {
		return db.Cond{}
	} else {
		return db.Cond{"name LIKE ": namePrefix + "%"}
	}
}

func (r *workflowArchive) GetWorkflow(clusterName, uid string) (*wfv1.Workflow, error) {
	archivedWf := &archivedWorkflowRecord{}
	err := r.session.
		Select("workflow").
		From(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID(clusterName)).
		And(db.Cond{"uid": uid}).
		One(archivedWf)
	if err != nil {
		if err == db.ErrNoMoreRows {
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

func (r *workflowArchive) DeleteWorkflow(clusterName, uid string) error {
	rs, err := r.session.
		DeleteFrom(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID(clusterName)).
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

func (r *workflowArchive) DeleteExpiredWorkflows(clusterName string, ttl time.Duration) error {
	rs, err := r.session.
		DeleteFrom(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID(clusterName)).
		And(fmt.Sprintf("finishedat < current_timestamp - interval '%d' second", int(ttl.Seconds()))).
		Exec()
	if err != nil {
		return err
	}
	rowsAffected, err := rs.RowsAffected()
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{"rowsAffected": rowsAffected}).Info("Deleted archived workflows")
	return nil
}
