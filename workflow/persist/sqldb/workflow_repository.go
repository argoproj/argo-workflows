package sqldb

import (
	"context"
	"encoding/json"

	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"upper.io/db.v3/lib/sqlbuilder"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type (
	WorkflowDBContext struct {
		TableName            string
		SupportLargeWorkflow bool
		Session              sqlbuilder.Database
	}

	DBRepository interface {
		Save(wf *wfv1.Workflow) error
		Get(uid string) (*wfv1.Workflow, error)
		List() ([]wfv1.Workflow, error)
		Query(condition interface{}) ([]wfv1.Workflow, error)
		Close() error
		IsSupportLargeWorkflow() bool
	}
)

type WorkflowDB struct {
	Id         string         `db:"id"`
	Name       string         `db:"name"`
	Phase      wfv1.NodePhase `db:"phase"`
	Namespace  string         `db:"namespace"`
	Workflow   string         `db:"workflow"`
	StartedAt  time.Time      `db:"startedat"`
	FinishedAt time.Time      `db:"finishedat"`
}

func convert(wf *wfv1.Workflow) *WorkflowDB {
	jsonWf, _ := json.Marshal(wf)
	startT, _ := time.Parse(time.RFC3339, wf.Status.StartedAt.Format(time.RFC3339))
	endT, _ := time.Parse(time.RFC3339, wf.Status.FinishedAt.Format(time.RFC3339))
	return &WorkflowDB{
		Id:         string(wf.UID),
		Name:       wf.Name,
		Namespace:  wf.Namespace,
		Workflow:   string(jsonWf),
		Phase:      wf.Status.Phase,
		StartedAt:  startT,
		FinishedAt: endT,
	}

}

func (wdc *WorkflowDBContext) IsSupportLargeWorkflow() bool {
	return wdc.SupportLargeWorkflow
}

func (wdc *WorkflowDBContext) Init(sess sqlbuilder.Database) {
	wdc.Session = sess
}

// Save will upset the workflow
func (wdc *WorkflowDBContext) Save(wf *wfv1.Workflow) error {

	if wdc != nil && wdc.Session == nil {
		return DBInvalidSession(nil, "DB session is not initialized")
	}
	wfdb := convert(wf)

	err := wdc.update(wfdb)

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

func (wdc *WorkflowDBContext) insert(wfDB *WorkflowDB) error {
	if wdc.Session == nil {
		return DBInvalidSession(nil, "DB session is not initialized")
	}
	tx, err := wdc.Session.NewTx(context.TODO())
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
	_, err = tx.Collection(wdc.TableName).Insert(wfDB)
	if err != nil {
		return errors.InternalErrorf("Error in inserting workflow in persistence. %v", err)
	}
	err = tx.Commit()
	if err != nil {
		return errors.InternalErrorf("Error in Committing workflow insert in persistence. %v", err)
	}
	return nil
}

func (wdc *WorkflowDBContext) update(wfDB *WorkflowDB) error {
	if wdc.Session == nil {
		return DBInvalidSession(nil, "DB session is not initialized")
	}
	tx, err := wdc.Session.NewTx(context.TODO())

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
	err = tx.Collection(wdc.TableName).UpdateReturning(wfDB)
	if err != nil {
		if strings.Contains(err.Error(), "upper: no more rows in this result set") {
			return DBUpdateNoRowFoundError(err, "")
		}
		return errors.InternalErrorf("Error in updating workflow in persistence %v", err)
	}

	err = tx.Commit()
	if err != nil {
		return errors.InternalErrorf("Error in Committing workflow update in persistence %v", err)
	}
	return nil
}

func (wdc *WorkflowDBContext) Get(uid string) (*wfv1.Workflow, error) {
	var wfDB WorkflowDB
	var wf wfv1.Workflow
	if wdc.Session == nil {
		return nil, DBInvalidSession(nil, "DB session is not initiallized")
	}

	err := wdc.Session.Collection(wdc.TableName).Find("id", uid).One(&wfDB)
	if err != nil {
		return nil, DBOperationError(err, "DB GET operation failed")
	}
	if wfDB.Id != "" {
		err := json.Unmarshal([]byte(wfDB.Workflow), &wf)
		if err != nil {
			log.Warnf(" Workflow unmarshalling failed for row=%v", wfDB)
		}
	} else {
		return nil, DBOperationError(nil, "Row is not found")
	}
	return &wf, nil

}

func (wdc *WorkflowDBContext) List() ([]wfv1.Workflow, error) {
	var wfDBs []WorkflowDB

	if wdc.Session == nil {
		return nil, DBInvalidSession(nil, "DB session is not initialized")
	}

	if err := wdc.Session.Collection(wdc.TableName).Find().OrderBy(" startedAt DESC").All(&wfDBs); err != nil {
		return nil, DBOperationError(err, "DB List operation failed")
	}
	var wfs []wfv1.Workflow
	for _, wfDB := range wfDBs {
		var wf wfv1.Workflow
		err := json.Unmarshal([]byte(wfDB.Workflow), &wf)
		if err != nil {
			log.Warnf(" Workflow unmarshalling failed for row=%v", wfDB)
		} else {
			wfs = append(wfs, wf)
		}
	}
	return wfs, nil
}

func (wdc *WorkflowDBContext) Query(condition interface{}) ([]wfv1.Workflow, error) {
	var wfDBs []WorkflowDB
	if wdc.Session == nil {
		return nil, DBInvalidSession(nil, "DB session is not initialized")
	}

	if err := wdc.Session.Collection(wdc.TableName).Find(condition).OrderBy(" startedAt DESC").All(&wfDBs); err != nil {
		return nil, DBOperationError(err, "DB Query opeartion failed")
	}
	var wfs []wfv1.Workflow
	for _, wfDB := range wfDBs {
		var wf wfv1.Workflow
		err := json.Unmarshal([]byte(wfDB.Workflow), &wf)
		if err != nil {
			log.Warnf(" Workflow unmarshalling failed for row=%v", wfDB)
		} else {
			wfs = append(wfs, wf)
		}
	}
	return wfs, nil
}

func (wdc *WorkflowDBContext) Close() error {
	if wdc.Session == nil {
		return DBInvalidSession(nil, "DB session is not initialized")
	}
	return wdc.Session.Close()
}
