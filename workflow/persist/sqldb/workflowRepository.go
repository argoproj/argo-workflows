package sqldb

import (
	"encoding/json"
	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/prometheus/common/log"
	"strings"
	"time"
	"upper.io/db.v3/lib/sqlbuilder"
)

type (
	WorkflowDBContext struct {
		TableName            string
		SupportLargeWorkflow bool
		Session              sqlbuilder.Database
	}

	DBRepository interface {
		Save(wf *wfv1.Workflow) error
		Get(uid string) *wfv1.Workflow
		ListAll() []wfv1.Workflow
		Query(condition interface{}) []wfv1.Workflow
		Close() error
		IsSupportLargeWorkflow() bool
		IsInterfaceNil() bool
	}
)

type WorkflowDB struct {
	Id         string         `db:"id"`
	Name       string         `db:"name"`
	Phase      wfv1.NodePhase `db:"phase"`
	Namespace  string         `db:"namespace"`
	Workflow   string         `db:"workflow"`
	StartedAt  time.Time      `db:"startedAt"`
	FinishedAt time.Time      `db:"finishedAt"`
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

func (wdc *WorkflowDBContext) IsInterfaceNil() bool{
	return wdc == nil
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
		return errors.InternalError("DB is not initiallized")
	}
	wfdb := convert(wf)

	err := wdc.update(wfdb)

	if err != nil {
		if errors.IsCode(errors.CodeDBUpdateRowNotFound, err) {
			return wdc.insert(wfdb)
		} else {
			log.Warn(err)
			return errors.InternalError("Error in inserting workflow in persistance")
		}
	}

	log.Info("Workflow update successfully into persistence")

	return nil
}


func (wdc *WorkflowDBContext) insert(wfDB *WorkflowDB) error {
	tx, err := wdc.Session.NewTx(nil)
	defer tx.Commit()
	err = tx.Collection(wdc.TableName).InsertReturning(wfDB)
	if err != nil {
		return errors.InternalErrorf("Error in inserting workflow in persistance %v", err)
	}
	return nil
}

func (wdc *WorkflowDBContext) update(wfDB *WorkflowDB) error {
	tx, err := wdc.Session.NewTx(nil)
	defer tx.Commit()
	err = tx.Collection(wdc.TableName).UpdateReturning(wfDB)
	if err != nil {
		if strings.Contains(err.Error(), "upper: no more rows in this result set") {
			return errors.DBUpdateNoRowFoundError(err, "")
		}
		return errors.InternalErrorf("Error in updating workflow in persistance %v", err)
	}
	return nil
}

func (wdc *WorkflowDBContext) Get(uid string) *wfv1.Workflow {
	var wfDB WorkflowDB
	var wf wfv1.Workflow
	if wdc.Session == nil {
		return nil
	}

	wdc.Session.Collection(wdc.TableName).Find("id", uid).One(&wfDB)

	if wfDB.Id != "" {
		err := json.Unmarshal([]byte(wfDB.Workflow), &wf)
		if err != nil {
			log.Warn(" Workflow unmarshalling is failed for ")
		}
	}
	return &wf

}

func (wdc *WorkflowDBContext) ListAll() []wfv1.Workflow {
	var wfDBs []WorkflowDB

	if err := wdc.Session.Collection(wdc.TableName).Find().OrderBy(" startedAt DESC").All(&wfDBs); err != nil {
		log.Fatal(err)
		return nil
	}
	var wfs [] wfv1.Workflow
	for _, wfDB := range wfDBs {
		var wf wfv1.Workflow
		err := json.Unmarshal([]byte(wfDB.Workflow), wf)
		if err != nil {
			log.Warn(" Workflow unmarshalling is failed for ")
		}else{
			wfs = append(wfs, wf)
		}
	}
	return wfs
}

func (wdc *WorkflowDBContext) Query(condition interface{}) []wfv1.Workflow {
	var wfDBs []WorkflowDB

	if err := wdc.Session.Collection(wdc.TableName).Find(condition).OrderBy(" startedAt DESC").All(&wfDBs); err != nil {
		log.Fatal(err)
		return nil
	}
	var wfs [] wfv1.Workflow
	for _, wfDB := range wfDBs {
		var wf wfv1.Workflow
		err := json.Unmarshal([]byte(wfDB.Workflow), wf)
		if err != nil {
			log.Warn(" Workflow unmarshalling is failed for ")
		}else{
			wfs = append(wfs, wf)
		}
	}
	return wfs
}

func (wdc *WorkflowDBContext) Close() error {
	return wdc.Session.Close()
}

