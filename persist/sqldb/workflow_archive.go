package sqldb

import (
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/upper/db/v4"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

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

	// The following fields are not stored as columns in the database, and they are stored as JSON strings in the workflow column, and will be loaded from there.
	CreationTimestamp string `db:"creationtimestamp,omitempty"`
	Labels            string `db:"labels,omitempty"`
	Annotations       string `db:"annotations,omitempty"`
	Suspend           *bool  `db:"suspend,omitempty"`
	Message           string `db:"message,omitempty"`
	Progress          string `db:"progress,omitempty"`
	EstimatedDuration int    `db:"estimatedduration,omitempty"`
	ResourcesDuration string `db:"resourcesduration,omitempty"`
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
	ListWorkflows(options sutils.ListOptions) (wfv1.Workflows, error)
	CountWorkflows(options sutils.ListOptions) (int64, error)
	GetWorkflow(uid string, namespace string, name string) (*wfv1.Workflow, error)
	GetWorkflowForEstimator(namespace string, requirements []labels.Requirement) (*wfv1.Workflow, error)
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

func (r *workflowArchive) ListWorkflows(options sutils.ListOptions) (wfv1.Workflows, error) {
	var archivedWfs []archivedWorkflowMetadata

	selectQuery, err := selectArchivedWorkflowQuery(r.dbType)
	if err != nil {
		return nil, err
	}

	subSelector := r.session.SQL().
		Select(db.Raw("uid")).
		From(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID())

	subSelector, err = BuildArchivedWorkflowSelector(subSelector, archiveTableName, archiveLabelsTableName, r.dbType, options, false)
	if err != nil {
		return nil, err
	}

	if r.dbType == MySQL {
		// workaround for mysql 42000 error (Unsupported subquery syntax):
		//
		//     Error 1235 (42000): This version of MySQL doesn't yet support 'LIMIT \u0026 IN/ALL/ANY/SOME subquery'
		//
		// more context:
		// * https://dev.mysql.com/doc/refman/8.0/en/subquery-errors.html
		// * https://dev.to/gkoniaris/limit-mysql-subquery-results-inside-a-where-in-clause-using-laravel-s-eloquent-orm-26en
		subSelector = r.session.SQL().Select(db.Raw("*")).From(subSelector).As("x")
	}

	// why a subquery? the json unmarshal triggers for every row in the filter
	// query. by filtering on uid first, we delay json parsing until a single
	// row, speeding up the query(e.g. up to 257 times faster for some
	// deployments).
	//
	// more context: https://github.com/argoproj/argo-workflows/pull/13566
	selector := r.session.SQL().Select(selectQuery).From(archiveTableName).Where(
		r.clusterManagedNamespaceAndInstanceID().And(db.Cond{"uid IN": subSelector}),
	)

	err = selector.All(&archivedWfs)
	if err != nil {
		return nil, err
	}

	wfs := make(wfv1.Workflows, len(archivedWfs))
	for i, md := range archivedWfs {
		labels := make(map[string]string)
		if err := json.Unmarshal([]byte(md.Labels), &labels); err != nil {
			return nil, err
		}
		// For backward compatibility, we should label workflow retrieved from DB as Persisted.
		labels[common.LabelKeyWorkflowArchivingStatus] = "Persisted"

		annotations := make(map[string]string)
		if err := json.Unmarshal([]byte(md.Annotations), &annotations); err != nil {
			return nil, err
		}

		t, err := time.Parse(time.RFC3339, md.CreationTimestamp)
		if err != nil {
			return nil, err
		}

		resourcesDuration := make(map[corev1.ResourceName]wfv1.ResourceDuration)
		if err := json.Unmarshal([]byte(md.ResourcesDuration), &resourcesDuration); err != nil {
			return nil, err
		}

		wfs[i] = wfv1.Workflow{
			ObjectMeta: v1.ObjectMeta{
				Name:              md.Name,
				Namespace:         md.Namespace,
				UID:               types.UID(md.UID),
				CreationTimestamp: v1.Time{Time: t},
				Labels:            labels,
				Annotations:       annotations,
			},
			Spec: wfv1.WorkflowSpec{
				Suspend: md.Suspend,
			},
			Status: wfv1.WorkflowStatus{
				Phase:             md.Phase,
				StartedAt:         v1.Time{Time: md.StartedAt},
				FinishedAt:        v1.Time{Time: md.FinishedAt},
				Progress:          wfv1.Progress(md.Progress),
				Message:           md.Message,
				EstimatedDuration: wfv1.EstimatedDuration(md.EstimatedDuration),
				ResourcesDuration: resourcesDuration,
			},
		}
	}
	return wfs, nil
}

func (r *workflowArchive) CountWorkflows(options sutils.ListOptions) (int64, error) {
	total := &archivedWorkflowCount{}

	selector := r.session.SQL().
		Select(db.Raw("count(*) as total")).
		From(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID())

	selector, err := BuildArchivedWorkflowSelector(selector, archiveTableName, archiveLabelsTableName, r.dbType, options, true)
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
		return db.Cond{"startedat >=": from}
	}
	return db.Cond{}
}

func startedAtToClause(to time.Time) db.Cond {
	if !to.IsZero() {
		return db.Cond{"startedat <=": to}
	}
	return db.Cond{}
}

func namespaceEqual(namespace string) db.Cond {
	if namespace != "" {
		return db.Cond{"namespace": namespace}
	}
	return db.Cond{}
}

func nameEqual(name string) db.Cond {
	if name != "" {
		return db.Cond{"name": name}
	}
	return db.Cond{}
}

func namePrefixClause(namePrefix string) db.Cond {
	if namePrefix != "" {
		return db.Cond{"name LIKE": namePrefix + "%"}
	}
	return db.Cond{}
}

func phaseEqual(phase string) db.Cond {
	if phase != "" {
		return db.Cond{"phase": phase}
	}
	return db.Cond{}
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

func (r *workflowArchive) GetWorkflowForEstimator(namespace string, requirements []labels.Requirement) (*wfv1.Workflow, error) {
	selector := r.session.SQL().
		Select("name", "namespace", "uid", "startedat", "finishedat").
		From(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID()).
		And(phaseEqual(string(wfv1.NodeSucceeded)))

	selector, err := BuildArchivedWorkflowSelector(selector, archiveTableName, archiveLabelsTableName, r.dbType, sutils.ListOptions{
		Namespace:         namespace,
		LabelRequirements: requirements,
		Limit:             1,
		Offset:            0,
	}, false)
	if err != nil {
		return nil, err
	}

	var awf archivedWorkflowMetadata
	err = selector.One(&awf)
	if err != nil {
		return nil, err
	}

	return &wfv1.Workflow{
		ObjectMeta: v1.ObjectMeta{
			Name:      awf.Name,
			Namespace: awf.Namespace,
			UID:       types.UID(awf.UID),
			Labels: map[string]string{
				common.LabelKeyWorkflowArchivingStatus: "Persisted",
			},
		},
		Status: wfv1.WorkflowStatus{
			StartedAt:  v1.Time{Time: awf.StartedAt},
			FinishedAt: v1.Time{Time: awf.FinishedAt},
		},
	}, nil

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

func selectArchivedWorkflowQuery(t dbType) (*db.RawExpr, error) {
	switch t {
	case MySQL:
		return db.Raw("name, namespace, uid, phase, startedat, finishedat, coalesce(JSON_EXTRACT(workflow,'$.metadata.labels'), '{}') as labels,coalesce(JSON_EXTRACT(workflow,'$.metadata.annotations'), '{}') as annotations, coalesce(JSON_UNQUOTE(JSON_EXTRACT(workflow,'$.status.progress')), '') as progress, coalesce(JSON_UNQUOTE(JSON_EXTRACT(workflow,'$.metadata.creationTimestamp')), '') as creationtimestamp, JSON_UNQUOTE(JSON_EXTRACT(workflow,'$.spec.suspend')) as suspend, coalesce(JSON_UNQUOTE(JSON_EXTRACT(workflow,'$.status.message')), '') as message, coalesce(JSON_UNQUOTE(JSON_EXTRACT(workflow,'$.status.estimatedDuration')), '0') as estimatedduration, coalesce(JSON_EXTRACT(workflow,'$.status.resourcesDuration'), '{}') as resourcesduration"), nil
	case Postgres:
		return db.Raw("name, namespace, uid, phase, startedat, finishedat, coalesce((workflow::json)->'metadata'->>'labels', '{}') as labels, coalesce((workflow::json)->'metadata'->>'annotations', '{}') as annotations, coalesce((workflow::json)->'status'->>'progress', '') as progress, coalesce((workflow::json)->'metadata'->>'creationTimestamp', '') as creationtimestamp, (workflow::json)->'spec'->>'suspend' as suspend, coalesce((workflow::json)->'status'->>'message', '') as message, coalesce((workflow::json)->'status'->>'estimatedDuration', '0') as estimatedduration, coalesce((workflow::json)->'status'->>'resourcesDuration', '{}') as resourcesduration"), nil
	}
	return nil, fmt.Errorf("unsupported db type %s", t)
}
