package sqldb

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type OffloadNodeStatusRepo interface {
	Save(wf *wfv1.Workflow) error
	Get(name, namespace, resourceVersion string) (*wfv1.Workflow, error)
	List(namespace string) (wfv1.Workflows, error)
	Delete(name, namespace string) error
	IsEnabled() bool
}

func NewOffloadNodeStatusRepo(tableName string, session sqlbuilder.Database) OffloadNodeStatusRepo {
	return &nodeOffloadRepo{tableName, session}
}

type nodeOffloadRepo struct {
	tableName string
	session   sqlbuilder.Database
}

func (wdc *nodeOffloadRepo) IsEnabled() bool {
	return true
}

// Save will upsert the workflow
func (wdc *nodeOffloadRepo) Save(wf *wfv1.Workflow) error {
	logCtx := log.WithFields(log.Fields{"name": wf.Name, "namespace": wf.Namespace})
	logCtx.Debug("Saving offloaded workflow")
	record, err := toRecord(wf)
	if err != nil {
		return err
	}

	_, err = wdc.session.Collection(wdc.tableName).Insert(record)
	if err != nil {
		// if we have a duplicate, then it must have the same name+namespace+resourceVersion, which MUST mean that we
		// have already written this record
		if !strings.Contains(err.Error(), "duplicate key") {
			return err
		}
	}

	logCtx.Info("Workflow offloaded into persistence")

	// this might fail, which kind of fine (maybe a bug),
	/// it might not delete all records, which is also fine, as we always key on resource version
	err = wdc.session.Collection(wdc.tableName).
		Find(db.Cond{"name": wf.Name}).
		And(db.Cond{"namespace": wf.Namespace}).
		And(db.Cond{"resourceversion <>": wf.ResourceVersion}).
		And(db.Cond{"updatedat <": "now()"}).
		Delete()

	return err
}

func (wdc *nodeOffloadRepo) Get(name, namespace, resourceVersion string) (*wfv1.Workflow, error) {
	log.WithFields(log.Fields{"name": name, "namespace": namespace, "resourceVersion": resourceVersion}).Debug("Getting offloaded workflow")
	wf := &WorkflowOnlyRecord{}
	err := wdc.session.
		Select("workflow").
		From(wdc.tableName).
		Where(db.Cond{"name": name}).
		And(db.Cond{"namespace": namespace}).
		And(db.Cond{"resourceversion": resourceVersion}).
		One(wf)
	if err != nil {
		return nil, err
	}

	return toWorkflow(wf)
}

func (wdc *nodeOffloadRepo) List(namespace string) (wfv1.Workflows, error) {
	var wfDBs []WorkflowOnlyRecord
	err := wdc.session.
		Select("workflow").
		From(tableName).
		Where(namespaceEqual(namespace)).
		OrderBy("-startedat").
		All(&wfDBs)
	if err != nil {
		return nil, err
	}

	return toWorkflows(wfDBs)
}

func (wdc *nodeOffloadRepo) Delete(name, namespace string) error {
	log.WithFields(log.Fields{"name": name, "namespace": namespace}).Debug("Deleting offloaded workflow")
	return wdc.session.Collection(wdc.tableName).Find(db.Cond{"name": name}).And(db.Cond{"namespace": namespace}).Delete()
}
