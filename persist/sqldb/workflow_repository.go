package sqldb

import (
	"context"
	"encoding/json"
	"fmt"

	"upper.io/db.v3"

	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"upper.io/db.v3/lib/sqlbuilder"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type WorkflowDBContext struct {
	TableName         string
	NodeStatusOffload bool
	Session           sqlbuilder.Database
}

func NewWorkflowDBContext(tableName string, nodeStatusOffload bool, session sqlbuilder.Database) *WorkflowDBContext {
	return &WorkflowDBContext{
		TableName:         tableName,
		NodeStatusOffload: nodeStatusOffload,
		Session:           session,
	}
}

type DBRepository interface {
	Save(wf *wfv1.Workflow) error
	Get(uid string) (*wfv1.Workflow, error)
	List(orderBy string) (*wfv1.WorkflowList, error)
	Query(condition db.Cond, orderBy ...interface{}) ([]wfv1.Workflow, error)
	Delete(condition db.Cond) error
	IsNodeStatusOffload() bool
	QueryWithPagination(condition db.Cond, pageSize uint, lastID string, orderBy ...interface{}) (*wfv1.WorkflowList, error)
}

type WorkflowDB struct {
	Id         string         `db:"id"`
	Name       string         `db:"name"`
	Phase      wfv1.NodePhase `db:"phase"`
	Namespace  string         `db:"namespace"`
	Workflow   string         `db:"workflow"`
	StartedAt  time.Time      `db:"startedat"`
	FinishedAt time.Time      `db:"finishedat"`
}

func convert(wf *wfv1.Workflow) (*WorkflowDB, error) {
	jsonWf, err := json.Marshal(wf)
	if err != nil {
		return nil, err
	}
	startT, err := time.Parse(time.RFC3339, wf.Status.StartedAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	endT, err := time.Parse(time.RFC3339, wf.Status.FinishedAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}

	return &WorkflowDB{
		Id:         string(wf.UID),
		Name:       wf.Name,
		Namespace:  wf.Namespace,
		Workflow:   string(jsonWf),
		Phase:      wf.Status.Phase,
		StartedAt:  startT,
		FinishedAt: endT,
	}, nil
}

func (wdc *WorkflowDBContext) IsNodeStatusOffload() bool {
	return wdc.NodeStatusOffload
}

func (wdc *WorkflowDBContext) Init(sess sqlbuilder.Database) {
	wdc.Session = sess
}

// Save will upset the workflow
func (wdc *WorkflowDBContext) Save(wf *wfv1.Workflow) error {
	if wdc != nil && wdc.Session == nil {
		return DBInvalidSession(fmt.Errorf("session nil"))
	}

	wfdb, err := convert(wf)
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

func (wdc *WorkflowDBContext) insert(wfDB *WorkflowDB) error {
	if wdc.Session == nil {
		return DBInvalidSession(fmt.Errorf("session nil"))
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
		return DBInvalidSession(fmt.Errorf("session nil"))
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

func (wdc *WorkflowDBContext) Get(uid string) (*wfv1.Workflow, error) {
	if wdc.Session == nil {
		return nil, DBInvalidSession(fmt.Errorf("session nil"))
	}

	cond := db.Cond{"id": uid}
	wfs, err := wdc.Query(cond)
	if err != nil {
		return nil, DBOperationError(err)
	}

	if len(wfs) > 0 {
		return &wfs[0], nil
	}

	return nil, DBOperationError(fmt.Errorf("not workflows"))
}

func (wdc *WorkflowDBContext) List(orderBy string) (*wfv1.WorkflowList, error) {
	if wdc.Session == nil {
		return nil, DBInvalidSession(fmt.Errorf("session nil"))
	}

	wfs, err := wdc.Query(nil, orderBy)
	if err != nil {
		return nil, err
	}

	return &wfv1.WorkflowList{
		Items: wfs,
	}, nil
}

func (wdc *WorkflowDBContext) Query(condition db.Cond, orderBy ...interface{}) ([]wfv1.Workflow, error) {
	var wfDBs []WorkflowDB
	if wdc.Session == nil {
		return nil, DBInvalidSession(fmt.Errorf("session nil"))
	}
	var err error
	//default Orderby
	var queryOrderBy []interface{}
	queryOrderBy = append(queryOrderBy, "-startedat")

	if orderBy != nil {
		queryOrderBy = orderBy
	}
	if condition != nil {
		err = wdc.Session.Collection(wdc.TableName).Find(condition).OrderBy(queryOrderBy...).All(&wfDBs)
	} else {
		err = wdc.Session.Collection(wdc.TableName).Find().OrderBy(queryOrderBy...).All(&wfDBs)
	}

	if err != nil {
		return nil, DBOperationError(err)
	}

	return wfDB2wf(wfDBs), nil
}

func wfDB2wf(wfDBs []WorkflowDB) []wfv1.Workflow {
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
	return wfs
}

func (wdc *WorkflowDBContext) QueryWithPagination(condition db.Cond, pageLimit uint, lastId string, orderBy ...interface{}) (*wfv1.WorkflowList, error) {
	var wfDBs []WorkflowDB
	if wdc.Session == nil {
		return nil, DBInvalidSession(fmt.Errorf("session nil"))
	}

	err := wdc.Session.Collection(wdc.TableName).Find(condition).OrderBy(orderBy).Paginate(pageLimit).NextPage(lastId).All(&wfDBs)
	if err != nil {
		return nil, DBOperationError(err)
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

	return &wfv1.WorkflowList{
		Items: wfs,
	}, nil
}

func (wdc *WorkflowDBContext) Delete(condition db.Cond) error {
	if wdc.Session == nil {
		return DBInvalidSession(fmt.Errorf("session nil"))
	}
	return wdc.Session.Collection(wdc.TableName).Find(condition).Delete()
}
