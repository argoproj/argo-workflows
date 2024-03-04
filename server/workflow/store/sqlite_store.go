package store

import (
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/sqlite"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
)

const (
	workflowTableName       = "argo_workflows"
	workflowLabelsTableName = "argo_workflows_labels"
)

type workflowMetadata struct {
	UID        string             `db:"uid"`
	InstanceID string             `db:"instanceid"`
	Name       string             `db:"name"`
	Namespace  string             `db:"namespace"`
	Phase      wfv1.WorkflowPhase `db:"phase"`
	StartedAt  time.Time          `db:"startedat"`
	FinishedAt time.Time          `db:"finishedat"`
}

type workflowRecord struct {
	workflowMetadata
	Workflow string `db:"workflow"`
}

type workflowLabelRecord struct {
	UID string `db:"uid"`
	// Why is this called "name" not "key"? Key is an SQL reserved word.
	Key   string `db:"name"`
	Value string `db:"value"`
}

type workflowCount struct {
	Total uint64 `db:"total,omitempty" json:"total"`
}

func initDB() (db.Session, error) {
	sess, err := sqlite.Open(sqlite.ConnectionURL{
		Database: `:memory:`,
		Options: map[string]string{
			"mode": "memory",
		},
	})
	if err != nil {
		return nil, err
	}
	_, err = sess.SQL().Exec("pragma foreign_keys = on")
	if err != nil {
		return nil, fmt.Errorf("failed to enable foreign key support: %w", err)
	}

	_, err = sess.SQL().Exec(`create table if not exists argo_workflows (
uid varchar(128) not null,
instanceid varchar(64),
name varchar(256),
namespace varchar(256),
phase varchar(25),
startedat timestamp,
finishedat timestamp,
workflow text,
primary key (uid)
)`)
	if err != nil {
		return nil, err
	}

	// create index for instanceid
	_, err = sess.SQL().Exec(`create index if not exists idx_instanceid on argo_workflows (instanceid)`)
	if err != nil {
		return nil, err
	}

	// create table for labels
	_, err = sess.SQL().Exec(`create table if not exists argo_workflows_labels (
name varchar(317) not null,
uid varchar(128) not null,
value varchar(63) not null,
primary key (uid, name, value),
foreign key (uid) references argo_workflows (uid) on delete cascade 
)`)
	if err != nil {
		return nil, err
	}
	// create index for name, value
	_, err = sess.SQL().Exec(`create index if not exists idx_name_value on argo_workflows_labels (name, value)`)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

type WorkflowStore interface {
	cache.Store
	ListWorkflows(namespace string, name string, namePrefix string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements, limit, offset int, showRemainingItemCount bool) (*wfv1.WorkflowList, error)
	CountWorkflows(namespace string, name string, namePrefix string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements) (int64, error)
}

// sqliteStore is a sqlite-based store.
type sqliteStore struct {
	session         db.Session
	instanceService instanceid.Service
}

var _ WorkflowStore = &sqliteStore{}

func NewSqliteStore(instanceService instanceid.Service) (db.Session, WorkflowStore, error) {
	session, err := initDB()
	if err != nil {
		return nil, nil, err
	}
	return session, &sqliteStore{session: session, instanceService: instanceService}, nil
}

func (s *sqliteStore) ListWorkflows(namespace string, name string, namePrefix string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements, limit, offset int, showRemainingItemCount bool) (*wfv1.WorkflowList, error) {
	var wfs []workflowRecord

	selector := s.session.SQL().
		Select("workflow").
		From(workflowTableName).
		Where(db.Cond{"instanceid": s.instanceService.InstanceID()})

	selector, err := sqldb.BuildWorkflowSelector(selector, workflowTableName, workflowLabelsTableName, false, sqldb.SQLite, namespace, name, namePrefix, minStartAt, maxStartAt, labelRequirements, limit, offset)
	if err != nil {
		return nil, err
	}

	err = selector.All(&wfs)
	if err != nil {
		return nil, err
	}

	workflows := make(wfv1.Workflows, 0)
	for _, wf := range wfs {
		w := wfv1.Workflow{}
		err := json.Unmarshal([]byte(wf.Workflow), &w)
		if err != nil {
			log.WithFields(log.Fields{"workflowUID": wf.UID, "workflowName": wf.Name}).Errorln("unable to unmarshal workflow from database")
		} else {
			workflows = append(workflows, w)
		}
	}

	meta := metav1.ListMeta{}
	if showRemainingItemCount || limit != 0 {
		total, err := s.CountWorkflows(namespace, name, namePrefix, minStartAt, maxStartAt, labelRequirements)
		if err != nil {
			return nil, sutils.ToStatusError(err, codes.Internal)
		}
		count := total - int64(offset) - int64(len(workflows))
		if count < 0 {
			count = 0
		}
		meta.RemainingItemCount = &count
		if count > 0 {
			meta.Continue = fmt.Sprintf("%v", offset+limit)
		}
	}

	return &wfv1.WorkflowList{Items: workflows, ListMeta: meta}, nil
}

func (s *sqliteStore) CountWorkflows(namespace string, name string, namePrefix string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements) (int64, error) {
	total := &workflowCount{}

	selector := s.session.SQL().
		Select(db.Raw("count(*) as total")).
		From(workflowTableName).
		Where(db.Cond{"instanceid": s.instanceService.InstanceID()})

	selector, err := sqldb.BuildWorkflowSelector(selector, workflowTableName, workflowLabelsTableName, false, sqldb.SQLite, namespace, name, namePrefix, minStartAt, maxStartAt, labelRequirements, 0, 0)
	if err != nil {
		return 0, err
	}

	err = selector.One(total)
	if err != nil {
		return 0, err
	}
	return int64(total.Total), nil
}

func (s *sqliteStore) Add(obj interface{}) error {
	wf, ok := obj.(*wfv1.Workflow)
	if !ok {
		return fmt.Errorf("unable to convert object to Workflow. object: %v", obj)
	}
	workflow, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	return s.session.Tx(func(sess db.Session) error {
		_, err := sess.SQL().
			DeleteFrom(workflowTableName).
			Where("uid", string(wf.UID)).
			Exec()
		if err != nil {
			return err
		}
		_, err = sess.Collection(workflowTableName).
			Insert(&workflowRecord{
				workflowMetadata: workflowMetadata{
					InstanceID: s.instanceService.InstanceID(),
					UID:        string(wf.UID),
					Name:       wf.Name,
					Namespace:  wf.Namespace,
					Phase:      wf.Status.Phase,
					StartedAt:  wf.Status.StartedAt.Time,
					FinishedAt: wf.Status.FinishedAt.Time,
				},
				Workflow: string(workflow),
			})
		if err != nil {
			return err
		}

		_, err = sess.SQL().
			DeleteFrom(workflowLabelsTableName).
			Where("uid", string(wf.UID)).
			Exec()
		if err != nil {
			return err
		}

		labelBatch := sess.SQL().
			InsertInto(workflowLabelsTableName).
			Batch(len(wf.GetLabels()))
		for key, value := range wf.GetLabels() {
			labelBatch.Values(&workflowLabelRecord{
				UID:   string(wf.UID),
				Key:   key,
				Value: value,
			})
		}
		labelBatch.Done()
		err = labelBatch.Wait()
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *sqliteStore) Update(obj interface{}) error {
	wf, ok := obj.(*wfv1.Workflow)
	if !ok {
		return fmt.Errorf("unable to convert object to Workflow. object: %v", obj)
	}
	workflow, err := json.Marshal(wf)
	if err != nil {
		return err
	}

	return s.session.Tx(func(sess db.Session) error {
		_, err = sess.SQL().
			Update(workflowTableName).
			Where("uid", string(wf.UID)).
			Set("workflow", workflow).
			Set("phase", wf.Status.Phase).
			Set("startedat", wf.Status.StartedAt.Time).
			Set("finishedat", wf.Status.FinishedAt.Time).
			Exec()
		if err != nil {
			return err
		}

		_, err = sess.SQL().
			DeleteFrom(workflowLabelsTableName).
			Where("uid", string(wf.UID)).
			Exec()
		if err != nil {
			return err
		}

		labelBatch := sess.SQL().
			InsertInto(workflowLabelsTableName).
			Batch(len(wf.GetLabels()))
		for key, value := range wf.GetLabels() {
			labelBatch.Values(&workflowLabelRecord{
				UID:   string(wf.UID),
				Key:   key,
				Value: value,
			})
		}
		labelBatch.Done()
		err = labelBatch.Wait()
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *sqliteStore) Delete(obj interface{}) error {
	wf, ok := obj.(*wfv1.Workflow)
	if !ok {
		return fmt.Errorf("unable to convert object to Workflow. object: %v", obj)
	}
	return s.session.Tx(func(sess db.Session) error {
		_, err := sess.SQL().
			DeleteFrom(workflowTableName).
			Where("uid", string(wf.UID)).
			Exec()
		if err != nil {
			return err
		}
		_, err = sess.SQL().
			DeleteFrom(workflowLabelsTableName).
			Where("uid", string(wf.UID)).
			Exec()
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *sqliteStore) Replace(list []interface{}, resourceVersion string) error {
	wfLists := make([]*workflowRecord, len(list))
	wfLabels := make([]*workflowLabelRecord, 0)
	for i, obj := range list {
		wf, ok := obj.(*wfv1.Workflow)
		if !ok {
			return fmt.Errorf("unable to convert object to Workflow. object: %v", obj)
		}
		workflow, err := json.Marshal(wf)
		if err != nil {
			return err
		}
		wfLists[i] = &workflowRecord{
			workflowMetadata: workflowMetadata{
				InstanceID: s.instanceService.InstanceID(),
				UID:        string(wf.UID),
				Name:       wf.Name,
				Namespace:  wf.Namespace,
				Phase:      wf.Status.Phase,
				StartedAt:  wf.Status.StartedAt.Time,
				FinishedAt: wf.Status.FinishedAt.Time,
			},
			Workflow: string(workflow),
		}
		for key, value := range wf.GetLabels() {
			wfLabels = append(wfLabels, &workflowLabelRecord{
				UID:   string(wf.UID),
				Key:   key,
				Value: value,
			})
		}
	}
	return s.session.Tx(func(sess db.Session) error {
		if err := sess.Collection(workflowTableName).Truncate(); err != nil {
			return err
		}
		if len(wfLists) == 0 {
			return nil
		}
		batch := sess.SQL().
			InsertInto(workflowTableName).
			Batch(len(wfLists))
		for _, wf := range wfLists {
			batch.Values(wf)
		}
		batch.Done()
		err := batch.Wait()
		if err != nil {
			return err
		}

		if err := sess.Collection(workflowLabelsTableName).Truncate(); err != nil {
			return err
		}
		labelBatch := sess.SQL().
			InsertInto(workflowLabelsTableName).
			Batch(len(wfLabels))
		for _, label := range wfLabels {
			labelBatch.Values(label)
		}
		labelBatch.Done()
		err = labelBatch.Wait()
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *sqliteStore) Resync() error {
	return nil
}

func (s *sqliteStore) List() []interface{} {
	panic("not implemented")
}

func (s *sqliteStore) ListKeys() []string {
	panic("not implemented")
}

func (s *sqliteStore) Get(obj interface{}) (item interface{}, exists bool, err error) {
	panic("not implemented")
}

func (s *sqliteStore) GetByKey(key string) (item interface{}, exists bool, err error) {
	panic("not implemented")
}
