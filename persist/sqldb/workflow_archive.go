package sqldb

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/upper/db/v4"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/labels"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
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

type archivedWorkflowCount struct {
	Total uint64 `db:"total,omitempty" json:"total"`
}

//go:generate mockery --name=WorkflowArchive

type WorkflowArchive interface {
	ArchiveWorkflow(wf *wfv1.Workflow) error
	// list workflows, with the most recently started workflows at the beginning (i.e. index 0 is the most recent)
	ListWorkflows(namespace string, name string, namePrefix string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements, limit, offset int) (wfv1.Workflows, error)
	CountWorkflows(namespace string, name string, namePrefix string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements) (int64, error)
	GetWorkflow(uid string, namespace string, name string) (*wfv1.Workflow, error)
	DeleteWorkflow(uid string) error
	DeleteExpiredWorkflows(ttl time.Duration) error
	IsEnabled() bool
	ListWorkflowsLabelKeys() (*wfv1.LabelKeys, error)
	ListWorkflowsLabelValues(key string) (*wfv1.LabelValues, error)
}

type workflowArchive struct {
	session           db.Session
	clusterName       string
	managedNamespace  string
	instanceIDService instanceid.Service
	dbType            dbType
}

func (r *workflowArchive) IsEnabled() bool {
	return true
}

// NewWorkflowArchive returns a new workflowArchive
func NewWorkflowArchive(session db.Session, clusterName, managedNamespace string, instanceIDService instanceid.Service) WorkflowArchive {
	return &workflowArchive{session: session, clusterName: clusterName, managedNamespace: managedNamespace, instanceIDService: instanceIDService, dbType: dbTypeFor(session)}
}

func (r *workflowArchive) ArchiveWorkflow(wf *wfv1.Workflow) error {
	logCtx := log.WithFields(log.Fields{"uid": wf.UID, "labels": wf.GetLabels()})
	logCtx.Debug("Archiving workflow")
	wf.ObjectMeta.Labels[common.LabelKeyWorkflowArchivingStatus] = "Persisted"
	workflow, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	return r.session.Tx(func(sess db.Session) error {
		_, err := sess.SQL().
			DeleteFrom(archiveTableName).
			Where(r.clusterManagedNamespaceAndInstanceID()).
			And(db.Cond{"uid": wf.UID}).
			Exec()
		if err != nil {
			return err
		}
		_, err = sess.Collection(archiveTableName).
			Insert(&archivedWorkflowRecord{
				archivedWorkflowMetadata: archivedWorkflowMetadata{
					ClusterName: r.clusterName,
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

		_, err = sess.SQL().
			DeleteFrom(archiveLabelsTableName).
			Where(db.Cond{"clustername": r.clusterName}).
			And(db.Cond{"uid": wf.UID}).
			Exec()
		if err != nil {
			return err
		}
		// insert the labels
		for key, value := range wf.GetLabels() {
			_, err := sess.Collection(archiveLabelsTableName).
				Insert(&archivedWorkflowLabelRecord{
					ClusterName: r.clusterName,
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

func (r *workflowArchive) ListWorkflows(namespace string, name string, namePrefix string, minStartedAt, maxStartedAt time.Time, labelRequirements labels.Requirements, limit int, offset int) (wfv1.Workflows, error) {
	var archivedWfs []archivedWorkflowRecord

	// If we were passed 0 as the limit, then we should load all available archived workflows
	// to match the behavior of the `List` operations in the Kubernetes API
	if limit == 0 {
		limit = -1
		offset = -1
	}

	selector := r.session.SQL().
		Select("workflow").
		From(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID()).
		And(namespaceEqual(namespace)).
		And(nameEqual(name)).
		And(namePrefixClause(namePrefix)).
		And(startedAtFromClause(minStartedAt)).
		And(startedAtToClause(maxStartedAt))

	selector, err := labelsClause(selector, r.dbType, labelRequirements)
	if err != nil {
		return nil, err
	}
	err = selector.
		OrderBy("-startedat").
		Limit(limit).
		Offset(offset).
		All(&archivedWfs)
	if err != nil {
		return nil, err
	}

	wfs := make(wfv1.Workflows, 0)
	for _, archivedWf := range archivedWfs {
		wf := wfv1.Workflow{}
		err = json.Unmarshal([]byte(archivedWf.Workflow), &wf)
		if err != nil {
			log.WithFields(log.Fields{"workflowUID": archivedWf.UID, "workflowName": archivedWf.Name}).Errorln("unable to unmarshal workflow from database")
		} else {
			// For backward compatibility, we should label workflow retrieved from DB as Persisted.
			wf.ObjectMeta.Labels[common.LabelKeyWorkflowArchivingStatus] = "Persisted"
			wfs = append(wfs, wf)
		}
	}
	return wfs, nil
}

func (r *workflowArchive) CountWorkflows(namespace string, name string, namePrefix string, minStartedAt, maxStartedAt time.Time, labelRequirements labels.Requirements) (int64, error) {
	total := &archivedWorkflowCount{}

	selector := r.session.SQL().
		Select(db.Raw("count(*) as total")).
		From(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID()).
		And(namespaceEqual(namespace)).
		And(nameEqual(name)).
		And(namePrefixClause(namePrefix)).
		And(startedAtFromClause(minStartedAt)).
		And(startedAtToClause(maxStartedAt))

	selector, err := labelsClause(selector, r.dbType, labelRequirements)
	if err != nil {
		return 0, err
	}
	err = selector.One(total)
	if err != nil {
		return 0, err
	}

	return int64(total.Total), nil
}

func (r *workflowArchive) clusterManagedNamespaceAndInstanceID() *db.AndExpr {
	return db.And(
		db.Cond{"clustername": r.clusterName},
		namespaceEqual(r.managedNamespace),
		db.Cond{"instanceid": r.instanceIDService.InstanceID()},
	)
}

func startedAtFromClause(from time.Time) db.Cond {
	if !from.IsZero() {
		return db.Cond{"startedat > ": from}
	}
	return db.Cond{}
}

func startedAtToClause(to time.Time) db.Cond {
	if !to.IsZero() {
		return db.Cond{"startedat < ": to}
	}
	return db.Cond{}
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

func (r *workflowArchive) GetWorkflow(uid string, namespace string, name string) (*wfv1.Workflow, error) {
	var err error
	archivedWf := &archivedWorkflowRecord{}
	if uid != "" {
		err = r.session.SQL().
			Select("workflow").
			From(archiveTableName).
			Where(r.clusterManagedNamespaceAndInstanceID()).
			And(db.Cond{"uid": uid}).
			One(archivedWf)
	} else {
		if name != "" && namespace != "" {
			total := &archivedWorkflowCount{}
			err = r.session.SQL().
				Select(db.Raw("count(*) as total")).
				From(archiveTableName).
				Where(r.clusterManagedNamespaceAndInstanceID()).
				And(namespaceEqual(namespace)).
				And(nameEqual(name)).
				One(total)
			if err != nil {
				return nil, err
			}
			num := int64(total.Total)
			if num > 1 {
				return nil, fmt.Errorf("found %d archived workflows with namespace/name: %s/%s", num, namespace, name)
			}
			err = r.session.SQL().
				Select("workflow").
				From(archiveTableName).
				Where(r.clusterManagedNamespaceAndInstanceID()).
				And(namespaceEqual(namespace)).
				And(nameEqual(name)).
				One(archivedWf)
		} else {
			return nil, sutils.ToStatusError(fmt.Errorf("both name and namespace are required if uid is not specified"), codes.InvalidArgument)
		}
	}
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
	// For backward compatibility, we should label workflow retrieved from DB as Persisted.
	wf.ObjectMeta.Labels[common.LabelKeyWorkflowArchivingStatus] = "Persisted"
	return wf, nil
}

func (r *workflowArchive) DeleteWorkflow(uid string) error {
	rs, err := r.session.SQL().
		DeleteFrom(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID()).
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

func (r *workflowArchive) DeleteExpiredWorkflows(ttl time.Duration) error {
	rs, err := r.session.SQL().
		DeleteFrom(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID()).
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
