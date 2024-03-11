package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	sutils "github.com/argoproj/argo-workflows/v3/server/utils"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

const (
	workflowTableName        = "argo_workflows"
	workflowLabelsTableName  = "argo_workflows_labels"
	tableInitializationQuery = `create table if not exists argo_workflows (
  uid varchar(128) not null,
  instanceid varchar(64),
  name varchar(256),
  namespace varchar(256),
  phase varchar(25),
  startedat timestamp,
  finishedat timestamp,
  workflow text,
  primary key (uid)
);
create index if not exists idx_instanceid on argo_workflows (instanceid);
create table if not exists argo_workflows_labels (
  name varchar(317) not null,
  uid varchar(128) not null,
  value varchar(63) not null,
  primary key (uid, name, value),
  foreign key (uid) references argo_workflows (uid) on delete cascade
);
create index if not exists idx_name_value on argo_workflows_labels (name, value);
`
	insertWorkflowQuery      = `insert into argo_workflows (uid, instanceid, name, namespace, phase, startedat, finishedat, workflow) values (?, ?, ?, ?, ?, ?, ?, ?)`
	insertWorkflowLabelQuery = `insert into argo_workflows_labels (uid, name, value) values (?, ?, ?)`
	deleteWorkflowQuery      = `delete from argo_workflows where uid = ?`
)

func initDB() (*sqlite.Conn, error) {
	conn, err := sqlite.OpenConn(":memory:", sqlite.OpenReadWrite)
	//conn, err := sqlite.OpenConn("./memory.db", sqlite.OpenReadWrite)
	if err != nil {
		return nil, err
	}
	err = sqlitex.ExecuteTransient(conn, "pragma foreign_keys = on", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to enable foreign key support: %w", err)
	}

	err = sqlitex.ExecuteScript(conn, tableInitializationQuery, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

type WorkflowStore interface {
	cache.Store
	ListWorkflows(namespace string, name string, namePrefix string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements, limit, offset int, showRemainingItemCount bool) (*wfv1.WorkflowList, error)
	CountWorkflows(namespace string, name string, namePrefix string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements) (int64, error)
}

// sqliteStore is a sqlite-based store.
type sqliteStore struct {
	conn            *sqlite.Conn
	instanceService instanceid.Service
}

var _ WorkflowStore = &sqliteStore{}

func NewSQLiteStore(instanceService instanceid.Service) (WorkflowStore, error) {
	conn, err := initDB()
	if err != nil {
		return nil, err
	}
	return &sqliteStore{conn: conn, instanceService: instanceService}, nil
}

func (s *sqliteStore) ListWorkflows(namespace string, name string, namePrefix string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements, limit, offset int, showRemainingItemCount bool) (*wfv1.WorkflowList, error) {
	query := `select workflow from argo_workflows
where instanceid = ?
`
	args := []any{s.instanceService.InstanceID()}

	query, args, err := sqldb.BuildWorkflowSelectorForRawQuery(query, args, workflowTableName, workflowLabelsTableName, false, sqldb.SQLite, namespace, name, namePrefix, minStartAt, maxStartAt, labelRequirements, limit, offset)
	if err != nil {
		return nil, err
	}

	var workflows = wfv1.Workflows{}
	err = sqlitex.Execute(s.conn, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			wf := stmt.ColumnText(0)
			w := wfv1.Workflow{}
			err := json.Unmarshal([]byte(wf), &w)
			if err != nil {
				log.WithFields(log.Fields{"workflow": wf}).Errorln("unable to unmarshal workflow from database")
			} else {
				workflows = append(workflows, w)
			}
			return nil
		},
	})
	if err != nil {
		return nil, err
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
	query := `select count(*) as total from argo_workflows
where instanceid = ?
`
	args := []any{s.instanceService.InstanceID()}

	query, args, err := sqldb.BuildWorkflowSelectorForRawQuery(query, args, workflowTableName, workflowLabelsTableName, false, sqldb.SQLite, namespace, name, namePrefix, minStartAt, maxStartAt, labelRequirements, 0, 0)
	if err != nil {
		return 0, err
	}

	var total int64
	err = sqlitex.Execute(s.conn, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			total = stmt.ColumnInt64(0)
			return nil
		},
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (s *sqliteStore) Add(obj interface{}) error {
	wf, ok := obj.(*wfv1.Workflow)
	if !ok {
		return fmt.Errorf("unable to convert object to Workflow. object: %v", obj)
	}
	done := sqlitex.Transaction(s.conn)
	err := s.upsertWorkflow(wf)
	defer done(&err)
	return err
}

func (s *sqliteStore) Update(obj interface{}) error {
	wf, ok := obj.(*wfv1.Workflow)
	if !ok {
		return fmt.Errorf("unable to convert object to Workflow. object: %v", obj)
	}
	done := sqlitex.Transaction(s.conn)
	err := s.upsertWorkflow(wf)
	defer done(&err)
	return err
}

func (s *sqliteStore) Delete(obj interface{}) error {
	wf, ok := obj.(*wfv1.Workflow)
	if !ok {
		return fmt.Errorf("unable to convert object to Workflow. object: %v", obj)
	}
	return sqlitex.Execute(s.conn, deleteWorkflowQuery, &sqlitex.ExecOptions{Args: []any{string(wf.UID)}})
}

func (s *sqliteStore) Replace(list []interface{}, resourceVersion string) error {
	wfs := make([]*wfv1.Workflow, 0, len(list))
	for _, obj := range list {
		wf, ok := obj.(*wfv1.Workflow)
		if !ok {
			return fmt.Errorf("unable to convert object to Workflow. object: %v", obj)
		}
		wfs = append(wfs, wf)
	}
	done := sqlitex.Transaction(s.conn)
	err := s.replaceWorkflows(wfs)
	defer done(&err)
	return err
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

func (s *sqliteStore) upsertWorkflow(wf *wfv1.Workflow) error {
	err := sqlitex.Execute(s.conn, deleteWorkflowQuery, &sqlitex.ExecOptions{Args: []any{string(wf.UID)}})
	if err != nil {
		return err
	}
	workflow, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	err = sqlitex.Execute(s.conn, insertWorkflowQuery,
		&sqlitex.ExecOptions{
			Args: []any{string(wf.UID), s.instanceService.InstanceID(), wf.Name, wf.Namespace, wf.Status.Phase, wf.Status.StartedAt.Time, wf.Status.FinishedAt.Time, string(workflow)},
		},
	)
	if err != nil {
		return err
	}
	stmt, err := s.conn.Prepare(insertWorkflowLabelQuery)
	if err != nil {
		return err
	}
	for key, value := range wf.GetLabels() {
		if err = stmt.Reset(); err != nil {
			return err
		}
		stmt.BindText(1, string(wf.UID))
		stmt.BindText(2, key)
		stmt.BindText(3, value)
		if _, err = stmt.Step(); err != nil {
			return err
		}
	}
	return nil
}

func (s *sqliteStore) replaceWorkflows(workflows []*wfv1.Workflow) error {
	err := sqlitex.Execute(s.conn, `delete from argo_workflows`, nil)
	if err != nil {
		return err
	}
	// add all workflows to argo_workflows table
	stmt, err := s.conn.Prepare(insertWorkflowQuery)
	if err != nil {
		return err
	}
	for _, wf := range workflows {
		if err = stmt.Reset(); err != nil {
			return err
		}
		stmt.BindText(1, string(wf.UID))
		stmt.BindText(2, s.instanceService.InstanceID())
		stmt.BindText(3, wf.Name)
		stmt.BindText(4, wf.Namespace)
		stmt.BindText(5, string(wf.Status.Phase))
		stmt.BindText(6, wf.Status.StartedAt.String())
		stmt.BindText(7, wf.Status.FinishedAt.String())
		workflow, err := json.Marshal(wf)
		if err != nil {
			return err
		}
		stmt.BindText(8, string(workflow))
		if _, err = stmt.Step(); err != nil {
			return err
		}
	}
	stmt, err = s.conn.Prepare(insertWorkflowLabelQuery)
	if err != nil {
		return err
	}
	for _, wf := range workflows {
		for key, val := range wf.GetLabels() {
			if err = stmt.Reset(); err != nil {
				return err
			}
			stmt.BindText(1, string(wf.UID))
			stmt.BindText(2, key)
			stmt.BindText(3, val)
			if _, err = stmt.Step(); err != nil {
				return err
			}
		}
	}
	return nil
}
