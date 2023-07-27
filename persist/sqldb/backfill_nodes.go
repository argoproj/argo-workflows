package sqldb

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/upper/db/v4"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type backfillNodes struct {
	tableName string
}

func (s backfillNodes) String() string {
	return fmt.Sprintf("backfillNodes{%s}", s.tableName)
}

func (s backfillNodes) apply(session db.Session) (err error) {
	log.Info("Backfill node status")
	rs, err := session.SQL().SelectFrom(s.tableName).
		Columns("workflow").
		Where(db.Cond{"version": nil}).
		Query()
	if err != nil {
		return err
	}

	defer func() {
		tmpErr := rs.Close()
		if err == nil {
			err = tmpErr
		}
	}()

	for rs.Next() {
		if err := rs.Err(); err != nil {
			return err
		}
		workflow := ""
		err := rs.Scan(&workflow)
		if err != nil {
			return err
		}
		var wf *wfv1.Workflow
		err = json.Unmarshal([]byte(workflow), &wf)
		if err != nil {
			return err
		}
		marshalled, version, err := nodeStatusVersion(wf.Status.Nodes)
		if err != nil {
			return err
		}
		logCtx := log.WithFields(log.Fields{"name": wf.Name, "namespace": wf.Namespace, "version": version})
		logCtx.Info("Back-filling node status")
		res, err := session.SQL().Update(archiveTableName).
			Set("version", wf.ResourceVersion).
			Set("nodes", marshalled).
			Where(db.Cond{"name": wf.Name}).
			And(db.Cond{"namespace": wf.Namespace}).
			Exec()
		if err != nil {
			return err
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected != 1 {
			logCtx.WithField("rowsAffected", rowsAffected).Warn("Expected exactly one row affected")
		}
	}
	return nil
}
