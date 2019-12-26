package workflow

import (
	dblib "upper.io/db.v3"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/persist/sqldb"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type DBService struct {
	wfDBctx sqldb.DBRepository
}

func NewDBService(wfDBctx sqldb.DBRepository) *DBService {
	return &DBService{wfDBctx}
}

func (db *DBService) Get(wfName string, namespace string) (*v1alpha1.Workflow, error) {
	if db.wfDBctx == nil {
		return nil, errors.New(errors.CodeInternal, "DB Context is not initialized")
	}

	cond := dblib.Cond{"name": wfName, "namespace": namespace}

	wfs, err := db.wfDBctx.Query(cond)
	if err != nil {
		return nil, err
	}
	if len(wfs) > 0 {
		return &wfs[0], nil
	}
	return nil, nil
}

func (db *DBService) List(namespace string, pageSize uint, lastId string) (*v1alpha1.WorkflowList, error) {
	if db.wfDBctx == nil {
		return nil, errors.New(errors.CodeInternal, "DB Context is not initialized")
	}

	var cond dblib.Cond
	if namespace != "" {
		cond = dblib.Cond{"namespace": namespace}
	}

	if pageSize == 0 {
		items, err := db.wfDBctx.Query(cond)
		if err != nil {
			return nil, err
		}
		return &v1alpha1.WorkflowList{
			Items: items,
		}, nil
	}

	wfList, err := db.wfDBctx.QueryWithPagination(cond, pageSize, lastId)
	if err != nil {
		return nil, err
	}
	return wfList, nil
}

func (db *DBService) Delete(wfName string, namespace string) error {
	if db.wfDBctx == nil {
		return errors.New(errors.CodeInternal, "DB Context is not initialized")
	}
	cond := dblib.Cond{"name": wfName, "namespace": namespace}

	return db.wfDBctx.Delete(cond)
}
