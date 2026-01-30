package sqldb

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/upper/db/v4"
	"google.golang.org/grpc/codes"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

const (
	archiveTableName        = "argo_archived_workflows"
	archiveLabelsTableName  = archiveTableName + "_labels"
	postgresNullReplacement = "ARGO_POSTGRES_NULL_REPLACEMENT"
	// Default timeout in seconds for database queries in GetWorkflowForEstimator to prevent blocking workflow execution
	// Can be overridden by WORKFLOW_ESTIMATION_DB_QUERY_TIMEOUT_SECONDS environment variable
	defaultEstimationDBQueryTimeoutSeconds = 5
)

type archivedWorkflowMetadata struct {
	ClusterName       string             `db:"clustername"`
	InstanceID        string             `db:"instanceid"`
	UID               string             `db:"uid"`
	Name              string             `db:"name"`
	Namespace         string             `db:"namespace"`
	Phase             wfv1.WorkflowPhase `db:"phase"`
	StartedAt         time.Time          `db:"startedat"`
	FinishedAt        time.Time          `db:"finishedat"`
	CreationTimestamp time.Time          `db:"creationtimestamp,omitempty"`

	// The following fields are not stored as columns in the database, and they are stored as JSON strings in the workflow column, and will be loaded from there.
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

type WorkflowArchive interface {
	ArchiveWorkflow(ctx context.Context, wf *wfv1.Workflow) error
	// list workflows, with the most recently started workflows at the beginning (i.e. index 0 is the most recent)
	ListWorkflows(ctx context.Context, options sutils.ListOptions) (wfv1.Workflows, error)
	CountWorkflows(ctx context.Context, options sutils.ListOptions) (int64, error)
	// HasMoreWorkflows efficiently checks if there are more workflows beyond the current offset+limit
	// This is much faster than counting all workflows for pagination purposes
	HasMoreWorkflows(ctx context.Context, options sutils.ListOptions) (bool, error)
	GetWorkflow(ctx context.Context, uid string, namespace string, name string) (*wfv1.Workflow, error)
	GetWorkflowForEstimator(ctx context.Context, namespace string, requirements []labels.Requirement) (*wfv1.Workflow, error)
	DeleteWorkflow(ctx context.Context, uid string) error
	DeleteExpiredWorkflows(ctx context.Context, ttl time.Duration) error
	IsEnabled() bool
	ListWorkflowsLabelKeys(ctx context.Context) (*wfv1.LabelKeys, error)
	ListWorkflowsLabelValues(ctx context.Context, key string) (*wfv1.LabelValues, error)
}

type workflowArchive struct {
	session           db.Session
	clusterName       string
	managedNamespace  string
	instanceIDService instanceid.Service
	dbType            sqldb.DBType
}

func (r *workflowArchive) IsEnabled() bool {
	return true
}

// NewWorkflowArchive returns a new workflowArchive
func NewWorkflowArchive(session db.Session, clusterName, managedNamespace string, instanceIDService instanceid.Service) WorkflowArchive {
	return &workflowArchive{session: session, clusterName: clusterName, managedNamespace: managedNamespace, instanceIDService: instanceIDService, dbType: sqldb.DBTypeFor(session)}
}

func (r *workflowArchive) ArchiveWorkflow(ctx context.Context, wf *wfv1.Workflow) error {
	ctx, logger := logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"uid": wf.UID, "labels": wf.GetLabels()}).InContext(ctx)
	logger.Debug(ctx, "Archiving workflow")
	wf.Labels[common.LabelKeyWorkflowArchivingStatus] = "Persisted"
	workflow, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	if r.dbType == sqldb.Postgres {
		workflow = bytes.ReplaceAll(workflow, []byte("\\u0000"), []byte(postgresNullReplacement))
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
					ClusterName:       r.clusterName,
					InstanceID:        r.instanceIDService.InstanceID(),
					UID:               string(wf.UID),
					Name:              wf.Name,
					Namespace:         wf.Namespace,
					Phase:             wf.Status.Phase,
					StartedAt:         wf.Status.StartedAt.Time,
					FinishedAt:        wf.Status.FinishedAt.Time,
					CreationTimestamp: wf.CreationTimestamp.Time,
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

func (r *workflowArchive) ListWorkflows(ctx context.Context, options sutils.ListOptions) (wfv1.Workflows, error) {
	var archivedWfs []archivedWorkflowMetadata
	var baseSelector = r.session.SQL().Select("name", "namespace", "uid", "phase", "startedat", "finishedat", "creationtimestamp")

	switch r.dbType {
	case sqldb.MySQL:
		selectQuery := baseSelector.
			Columns(
				db.Raw("coalesce(workflow->'$.metadata.labels', '{}') as labels"),
				db.Raw("coalesce(workflow->'$.metadata.annotations', '{}') as annotations"),
				db.Raw("coalesce(workflow->>'$.status.progress', '') as progress"),
				db.Raw("workflow->>'$.spec.suspend'"),
				db.Raw("coalesce(workflow->>'$.status.message', '') as message"),
				db.Raw("coalesce(workflow->>'$.status.estimatedDuration', '0') as estimatedduration"),
				db.Raw("coalesce(workflow->'$.status.resourcesDuration', '{}') as resourcesduration"),
			).
			From(archiveTableName).
			Where(r.clusterManagedNamespaceAndInstanceID())

		selectQuery, err := BuildArchivedWorkflowSelector(selectQuery, archiveTableName, archiveLabelsTableName, r.dbType, options, false)
		if err != nil {
			return nil, err
		}

		err = selectQuery.All(&archivedWfs)
		if err != nil {
			return nil, err
		}
	case sqldb.Postgres:
		// Use a common table expression to reduce detoast overhead for the "workflow" column:
		// https://github.com/argoproj/argo-workflows/issues/13601#issuecomment-2420499551
		cteSelector := baseSelector.
			Columns(
				db.Raw("coalesce(workflow->'metadata', '{}') as metadata"),
				db.Raw("coalesce(workflow->'status', '{}') as status"),
				db.Raw("workflow->'spec'->>'suspend' as suspend"),
			).
			From(archiveTableName).
			Where(r.clusterManagedNamespaceAndInstanceID())

		cteSelector, err := BuildArchivedWorkflowSelector(cteSelector, archiveTableName, archiveLabelsTableName, r.dbType, options, false)
		if err != nil {
			return nil, err
		}

		selectQuery := baseSelector.Columns(
			db.Raw("coalesce(metadata->>'labels', '{}') as labels"),
			db.Raw("coalesce(metadata->>'annotations', '{}') as annotations"),
			db.Raw("coalesce(status->>'progress', '') as progress"),
			"suspend",
			db.Raw("coalesce(status->>'message', '') as message"),
			db.Raw("coalesce(status->>'estimatedDuration', '0') as estimatedduration"),
			db.Raw("coalesce(status->>'resourcesDuration', '{}') as resourcesduration"),
		)

		err = r.session.SQL().
			Iterator("WITH workflows AS ? ?", cteSelector, selectQuery.From("workflows")).
			All(&archivedWfs)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported db type %s", r.dbType)
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

		t := md.CreationTimestamp

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

func (r *workflowArchive) CountWorkflows(ctx context.Context, options sutils.ListOptions) (int64, error) {
	if options.Limit > 0 && options.Offset > 0 {
		return r.countWorkflowsOptimized(options)
	}

	total := &archivedWorkflowCount{}

	if len(options.LabelRequirements) == 0 {
		selector := r.session.SQL().
			Select(db.Raw("count(*) as total")).
			From(archiveTableName).
			Where(r.clusterManagedNamespaceAndInstanceID()).
			And(namespaceEqual(options.Namespace)).
			And(namePrefixClause(options.NamePrefix)).
			And(startedAtFromClause(options.MinStartedAt)).
			And(startedAtToClause(options.MaxStartedAt)).
			And(createdAfterClause(options.CreatedAfter)).
			And(finishedBeforeClause(options.FinishedBefore))

		if options.Name != "" {
			nameFilter := options.NameFilter
			if nameFilter == "" {
				nameFilter = "Exact"
			}
			if nameFilter == "Exact" {
				selector = selector.And(nameEqual(options.Name))
			}
			if nameFilter == "Contains" {
				selector = selector.And(nameContainsClause(options.Name))
			}
			if nameFilter == "Prefix" {
				selector = selector.And(namePrefixClause(options.Name))
			}
			if nameFilter == "NotEquals" {
				selector = selector.And(nameNotEqual(options.Name))
			}
		}

		err := selector.One(total)
		if err != nil {
			return 0, err
		}
		return int64(total.Total), nil
	}

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

func (r *workflowArchive) countWorkflowsOptimized(options sutils.ListOptions) (int64, error) {
	sampleSelector := r.session.SQL().
		Select(db.Raw("count(*) as total")).
		From(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID()).
		And(namespaceEqual(options.Namespace)).
		And(namePrefixClause(options.NamePrefix)).
		And(startedAtFromClause(options.MinStartedAt)).
		And(startedAtToClause(options.MaxStartedAt)).
		And(createdAfterClause(options.CreatedAfter)).
		And(finishedBeforeClause(options.FinishedBefore))

	if options.Name != "" {
		nameFilter := options.NameFilter
		if nameFilter == "" {
			nameFilter = "Exact"
		}
		if nameFilter == "Exact" {
			sampleSelector = sampleSelector.And(nameEqual(options.Name))
		}
		if nameFilter == "Contains" {
			sampleSelector = sampleSelector.And(nameContainsClause(options.Name))
		}
		if nameFilter == "Prefix" {
			sampleSelector = sampleSelector.And(namePrefixClause(options.Name))
		}
		if nameFilter == "NotEquals" {
			sampleSelector = sampleSelector.And(nameNotEqual(options.Name))
		}
	}

	if options.Offset < 1000 {
		total := &archivedWorkflowCount{}
		err := sampleSelector.One(total)
		if err != nil {
			return 0, err
		}
		return int64(total.Total), nil
	}

	sampleSize := 1000
	sampleSelector = sampleSelector.Limit(sampleSize)

	sampleTotal := &archivedWorkflowCount{}
	err := sampleSelector.One(sampleTotal)
	if err != nil {
		return 0, err
	}

	if int64(sampleTotal.Total) < int64(sampleSize) {
		return int64(sampleTotal.Total), nil
	}

	estimatedTotal := int64(options.Offset) + int64(sampleTotal.Total) + int64(options.Limit)
	return estimatedTotal, nil
}

func (r *workflowArchive) HasMoreWorkflows(ctx context.Context, options sutils.ListOptions) (bool, error) {
	selector := r.session.SQL().
		Select("uid").
		From(archiveTableName).
		Where(r.clusterManagedNamespaceAndInstanceID()).
		And(namespaceEqual(options.Namespace)).
		And(namePrefixClause(options.NamePrefix)).
		And(startedAtFromClause(options.MinStartedAt)).
		And(startedAtToClause(options.MaxStartedAt)).
		And(createdAfterClause(options.CreatedAfter)).
		And(finishedBeforeClause(options.FinishedBefore))

	if options.Name != "" {
		nameFilter := options.NameFilter
		if nameFilter == "" {
			nameFilter = "Exact"
		}
		if nameFilter == "Exact" {
			selector = selector.And(nameEqual(options.Name))
		}
		if nameFilter == "Contains" {
			selector = selector.And(nameContainsClause(options.Name))
		}
		if nameFilter == "Prefix" {
			selector = selector.And(namePrefixClause(options.Name))
		}
		if nameFilter == "NotEquals" {
			selector = selector.And(nameNotEqual(options.Name))
		}
	}

	if len(options.LabelRequirements) > 0 {
		var err error
		selector, err = BuildArchivedWorkflowSelector(selector, archiveTableName, archiveLabelsTableName, r.dbType, options, false)
		if err != nil {
			return false, err
		}
	}

	selector = selector.Limit(1).Offset(options.Offset + options.Limit)

	var result []struct{ UID string }
	err := selector.All(&result)
	if err != nil {
		return false, err
	}

	return len(result) > 0, nil
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

func createdAfterClause(createdAfter time.Time) db.Cond {
	if !createdAfter.IsZero() {
		return db.Cond{"creationtimestamp >=": createdAfter}
	}
	return db.Cond{}
}

func finishedBeforeClause(finishedBefore time.Time) db.Cond {
	if !finishedBefore.IsZero() {
		return db.Cond{"finishedat <=": finishedBefore}
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

func nameNotEqual(name string) db.Cond {
	if name != "" {
		return db.Cond{"name !=": name}
	}
	return db.Cond{}
}

func namePrefixClause(namePrefix string) db.Cond {
	if namePrefix != "" {
		return db.Cond{"name LIKE": namePrefix + "%"}
	}
	return db.Cond{}
}

func nameContainsClause(nameSubstring string) db.Cond {
	if nameSubstring != "" {
		return db.Cond{"name LIKE": "%" + nameSubstring + "%"}
	}
	return db.Cond{}
}

func phaseEqual(phase string) db.Cond {
	if phase != "" {
		return db.Cond{"phase": phase}
	}
	return db.Cond{}
}

func (r *workflowArchive) GetWorkflow(ctx context.Context, uid string, namespace string, name string) (*wfv1.Workflow, error) {
	logger := logging.RequireLoggerFromContext(ctx)
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
		if name == "" || namespace == "" {
			return nil, sutils.ToStatusError(fmt.Errorf("both name and namespace are required if uid is not specified"), codes.InvalidArgument)
		}
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
			logger.WithFields(logging.Fields{
				"namespace": namespace,
				"name":      name,
				"num":       num,
			}).Debug(ctx, "returning latest of archived workflows")
		}
		err = r.session.SQL().
			Select("workflow").
			From(archiveTableName).
			Where(r.clusterManagedNamespaceAndInstanceID()).
			And(namespaceEqual(namespace)).
			And(nameEqual(name)).
			OrderBy("-startedat").
			One(archivedWf)
	}
	if err != nil {
		if errors.Is(err, db.ErrNoMoreRows) {
			return nil, nil
		}
		return nil, err
	}
	var wf *wfv1.Workflow
	if r.dbType == sqldb.Postgres {
		archivedWf.Workflow = strings.ReplaceAll(archivedWf.Workflow, postgresNullReplacement, "\\u0000")
	}
	err = json.Unmarshal([]byte(archivedWf.Workflow), &wf)
	if err != nil {
		return nil, err
	}
	// For backward compatibility, we should label workflow retrieved from DB as Persisted.
	wf.Labels[common.LabelKeyWorkflowArchivingStatus] = "Persisted"
	return wf, nil
}

func (r *workflowArchive) GetWorkflowForEstimator(ctx context.Context, namespace string, requirements []labels.Requirement) (*wfv1.Workflow, error) {
	// Add timeout to database query to prevent blocking workflow execution
	// if database is slow or locked.
	queryTimeoutSeconds := env.LookupEnvIntOr(ctx, "WORKFLOW_ESTIMATION_DB_QUERY_TIMEOUT_SECONDS", defaultEstimationDBQueryTimeoutSeconds)
	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(queryTimeoutSeconds)*time.Second)
	defer cancel()

	selector := r.session.WithContext(queryCtx).SQL().
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

func (r *workflowArchive) DeleteWorkflow(ctx context.Context, uid string) error {
	logger := logging.RequireLoggerFromContext(ctx)
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
	logger.WithFields(logging.Fields{"uid": uid, "rowsAffected": rowsAffected}).Debug(ctx, "Deleted archived workflow")
	return nil
}

func (r *workflowArchive) DeleteExpiredWorkflows(ctx context.Context, ttl time.Duration) error {
	logger := logging.RequireLoggerFromContext(ctx)
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
	logger.WithFields(logging.Fields{"rowsAffected": rowsAffected}).Info(ctx, "Deleted archived workflows")
	return nil
}
