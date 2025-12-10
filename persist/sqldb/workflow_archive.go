package sqldb

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

const (
	archiveTableName        = "argo_archived_workflows"
	archiveLabelsTableName  = archiveTableName + "_labels"
	postgresNullReplacement = "ARGO_POSTGRES_NULL_REPLACEMENT"
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
	sessionProxy      *sqldb.SessionProxy
	clusterName       string
	managedNamespace  string
	instanceIDService instanceid.Service
	dbType            sqldb.DBType
}

func (r *workflowArchive) IsEnabled() bool {
	return true
}

// NewWorkflowArchive creates a WorkflowArchive that stores archived workflows for the given cluster and managed namespace using the provided SessionProxy and instance ID service.
func NewWorkflowArchive(sessionProxy *sqldb.SessionProxy, clusterName, managedNamespace string, instanceIDService instanceid.Service) WorkflowArchive {
	return &workflowArchive{sessionProxy: sessionProxy, clusterName: clusterName, managedNamespace: managedNamespace, instanceIDService: instanceIDService, dbType: sqldb.DBTypeFor(sessionProxy.Session())}
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
	return r.sessionProxy.TxWith(ctx, func(sp *sqldb.SessionProxy) error {
		_, err := sp.Session().SQL().
			DeleteFrom(archiveTableName).
			Where(r.clusterManagedNamespaceAndInstanceID()).
			And(db.Cond{"uid": wf.UID}).
			Exec()
		if err != nil {
			return err
		}
		_, err = sp.Session().Collection(archiveTableName).
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

		_, err = sp.Session().SQL().
			DeleteFrom(archiveLabelsTableName).
			Where(db.Cond{"clustername": r.clusterName}).
			And(db.Cond{"uid": wf.UID}).
			Exec()
		if err != nil {
			return err
		}
		// insert the labels
		for key, value := range wf.GetLabels() {
			_, err := sp.Session().Collection(archiveLabelsTableName).
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
	}, nil)
}

func (r *workflowArchive) ListWorkflows(ctx context.Context, options sutils.ListOptions) (wfv1.Workflows, error) {
	var archivedWfs []archivedWorkflowMetadata

	switch r.dbType {
	case sqldb.MySQL:
		err := r.sessionProxy.With(ctx, func(s db.Session) error {
			baseSelector := s.SQL().Select("name", "namespace", "uid", "phase", "startedat", "finishedat", "creationtimestamp")
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
				return err
			}

			err = selectQuery.All(&archivedWfs)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	case sqldb.Postgres:
		err := r.sessionProxy.With(ctx, func(s db.Session) error {
			baseSelector := s.SQL().Select("name", "namespace", "uid", "phase", "startedat", "finishedat", "creationtimestamp")
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
				return err
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

			err = s.SQL().
				Iterator("WITH workflows AS ? ?", cteSelector, selectQuery.From("workflows")).
				All(&archivedWfs)
			return err
		})
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
		return r.countWorkflowsOptimized(ctx, options)
	}

	var result int64
	err := r.sessionProxy.With(ctx, func(s db.Session) error {
		total := &archivedWorkflowCount{}

		if len(options.LabelRequirements) == 0 {
			selector := s.SQL().
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
			}

			err := selector.One(total)
			if err != nil {
				return err
			}
			result = int64(total.Total)
			return nil
		}

		selector := s.SQL().
			Select(db.Raw("count(*) as total")).
			From(archiveTableName).
			Where(r.clusterManagedNamespaceAndInstanceID())

		selector, err := BuildArchivedWorkflowSelector(selector, archiveTableName, archiveLabelsTableName, r.dbType, options, true)
		if err != nil {
			return err
		}
		err = selector.One(total)
		if err != nil {
			return err
		}

		result = int64(total.Total)
		return nil
	})
	if err != nil {
		return 0, err
	}

	return result, nil
}

func (r *workflowArchive) countWorkflowsOptimized(ctx context.Context, options sutils.ListOptions) (int64, error) {
	var result int64
	err := r.sessionProxy.With(ctx, func(s db.Session) error {
		sampleSelector := s.SQL().
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
		}

		if options.Offset < 1000 {
			total := &archivedWorkflowCount{}
			err := sampleSelector.One(total)
			if err != nil {
				return err
			}
			result = int64(total.Total)
			return nil
		}

		sampleSize := 1000
		sampleSelector = sampleSelector.Limit(sampleSize)

		sampleTotal := &archivedWorkflowCount{}
		err := sampleSelector.One(sampleTotal)
		if err != nil {
			return err
		}

		if int64(sampleTotal.Total) < int64(sampleSize) {
			result = int64(sampleTotal.Total)
			return nil
		}

		result = int64(options.Offset) + int64(sampleTotal.Total) + int64(options.Limit)
		return nil
	})
	if err != nil {
		return 0, err
	}

	return result, nil
}

func (r *workflowArchive) HasMoreWorkflows(ctx context.Context, options sutils.ListOptions) (bool, error) {
	var hasMore bool
	err := r.sessionProxy.With(ctx, func(s db.Session) error {
		selector := s.SQL().
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
		}

		if len(options.LabelRequirements) > 0 {
			var err error
			selector, err = BuildArchivedWorkflowSelector(selector, archiveTableName, archiveLabelsTableName, r.dbType, options, false)
			if err != nil {
				return err
			}
		}

		selector = selector.Limit(1).Offset(options.Offset + options.Limit)

		var result []struct{ UID string }
		err := selector.All(&result)
		if err != nil {
			return err
		}

		hasMore = len(result) > 0
		return nil
	})
	if err != nil {
		return false, err
	}

	return hasMore, nil
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

// phaseEqual returns a database condition that filters rows where the "phase"
// column equals the provided value. If the phase argument is an empty string,
// an empty condition is returned.
func phaseEqual(phase string) db.Cond {
	if phase != "" {
		return db.Cond{"phase": phase}
	}
	return db.Cond{}
}

func (r *workflowArchive) GetWorkflow(ctx context.Context, uid string, namespace string, name string) (*wfv1.Workflow, error) {
	var result *wfv1.Workflow
	logger := logging.RequireLoggerFromContext(ctx)

	err := r.sessionProxy.With(ctx, func(s db.Session) error {
		archivedWf := &archivedWorkflowRecord{}
		var err error

		if uid != "" {
			err = s.SQL().
				Select("workflow").
				From(archiveTableName).
				Where(r.clusterManagedNamespaceAndInstanceID()).
				And(db.Cond{"uid": uid}).
				One(archivedWf)
		} else {
			if name != "" && namespace != "" {
				total := &archivedWorkflowCount{}
				err = s.SQL().
					Select(db.Raw("count(*) as total")).
					From(archiveTableName).
					Where(r.clusterManagedNamespaceAndInstanceID()).
					And(namespaceEqual(namespace)).
					And(nameEqual(name)).
					One(total)
				if err != nil {
					return err
				}
				num := int64(total.Total)
				if num > 1 {
					logger.WithFields(logging.Fields{
						"namespace": namespace,
						"name":      name,
						"num":       num,
					}).Debug(ctx, "returning latest of archived workflows")
				}
				err = s.SQL().
					Select("workflow").
					From(archiveTableName).
					Where(r.clusterManagedNamespaceAndInstanceID()).
					And(namespaceEqual(namespace)).
					And(nameEqual(name)).
					OrderBy("-startedat").
					One(archivedWf)
			} else {
				return sutils.ToStatusError(fmt.Errorf("both name and namespace are required if uid is not specified"), codes.InvalidArgument)
			}
		}
		if err != nil {
			if err == db.ErrNoMoreRows {
				result = nil
				return nil
			}
			return err
		}

		var wf *wfv1.Workflow
		if r.dbType == sqldb.Postgres {
			archivedWf.Workflow = strings.ReplaceAll(archivedWf.Workflow, postgresNullReplacement, "\\u0000")
		}
		err = json.Unmarshal([]byte(archivedWf.Workflow), &wf)
		if err != nil {
			return err
		}
		// For backward compatibility, we should label workflow retrieved from DB as Persisted.
		wf.Labels[common.LabelKeyWorkflowArchivingStatus] = "Persisted"
		result = wf
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *workflowArchive) GetWorkflowForEstimator(ctx context.Context, namespace string, requirements []labels.Requirement) (*wfv1.Workflow, error) {
	var result *wfv1.Workflow
	err := r.sessionProxy.With(ctx, func(s db.Session) error {
		selector := s.SQL().
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
			return err
		}

		var awf archivedWorkflowMetadata
		err = selector.One(&awf)
		if err != nil {
			return err
		}

		result = &wfv1.Workflow{
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
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *workflowArchive) DeleteWorkflow(ctx context.Context, uid string) error {
	logger := logging.RequireLoggerFromContext(ctx)
	return r.sessionProxy.With(ctx, func(s db.Session) error {
		rs, err := s.SQL().
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
	})
}

func (r *workflowArchive) DeleteExpiredWorkflows(ctx context.Context, ttl time.Duration) error {
	logger := logging.RequireLoggerFromContext(ctx)
	return r.sessionProxy.With(ctx, func(s db.Session) error {
		rs, err := s.SQL().
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
	})
}