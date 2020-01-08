package sqldb

import (
	"context"
	"strings"

	log "github.com/sirupsen/logrus"
	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type OffloadNodeStatusRepo interface {
	Save(wf *wfv1.Workflow) error
	Get(name, namespace string) (*wfv1.Workflow, error)
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

	wfdb, err := toRecord(wf)
	if err != nil {
		return err
	}

	err = wdc.update(wfdb)
	if err != nil {
		if errors.IsCode(CodeDBUpdateRowNotFound, err) {
			return wdc.insert(wfdb)
		} else {
			log.Warn(err)
			return errors.InternalErrorf("Error in inserting workflow in persistence. %v", err)
		}
	}

	log.Info("Workflow update successfully into persistence")
	return nil
}

func (wdc *nodeOffloadRepo) insert(wfDB *WorkflowRecord) error {
	tx, err := wdc.session.NewTx(context.TODO())
	if err != nil {
		return errors.InternalErrorf("Error in creating transaction. %v", err)
	}

	defer func() {
		if tx != nil {
			err := tx.Close()
			if err != nil {
				log.Warnf("Transaction failed to close")
			}
		}
	}()

	_, err = tx.Collection(wdc.tableName).Insert(wfDB)
	if err != nil {
		return errors.InternalErrorf("Error in inserting workflow in persistence. %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return errors.InternalErrorf("Error in Committing workflow insert in persistence. %v", err)
	}

	return nil
}

func (wdc *nodeOffloadRepo) update(wfDB *WorkflowRecord) error {
	tx, err := wdc.session.NewTx(context.TODO())
	if err != nil {
		return errors.InternalErrorf("Error in creating transaction. %v", err)
	}

	defer func() {
		if tx != nil {
			err := tx.Close()
			if err != nil {
				log.Warnf("Transaction failed to close")
			}
		}
	}()

	err = tx.Collection(wdc.tableName).UpdateReturning(wfDB)
	if err != nil {
		if strings.Contains(err.Error(), "upper: no more rows in this result set") {
			return DBUpdateNoRowFoundError(err)
		}
		return errors.InternalErrorf("Error in updating workflow in persistence %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return errors.InternalErrorf("Error in Committing workflow update in persistence %v", err)
	}

	return nil
}

func (wdc *nodeOffloadRepo) Get(name, namespace string) (*wfv1.Workflow, error) {
	wf := &WorkflowOnlyRecord{}
	err := wdc.session.
		Select("workflow").
		From(wdc.tableName).
		Where(db.Cond{"name": name}).
		And(db.Cond{"namespace": namespace}).
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
	return wdc.session.Collection(wdc.tableName).Find(db.Cond{"name": name}).And(db.Cond{"namespace": namespace}).Delete()
}
