package sqldb

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type backfillResourceVersion struct {
	tableName string
}

func (s backfillResourceVersion) String() string {
	return fmt.Sprintf("backfillResourceVersion{%s}", s.tableName)
}

func (s backfillResourceVersion) Apply(session sqlbuilder.Database) error {
	log.Info("Back-filling resource versions")
	for _, tableName := range []string{s.tableName, "argo_archived_workflows"} {
		err := backfillTable(session, tableName)
		if err != nil {
			return err
		}
	}
	return nil
}

func backfillTable(session sqlbuilder.Database, tableName string) error {
	rs, err := session.SelectFrom(tableName).
		Columns("workflow").
		Where(db.Cond{"resourceversion": nil}).
		Query()
	if err != nil {
		return err
	}
	for rs.Next() {
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
		logCtx := log.WithFields(log.Fields{"name": wf.Name, "namespace": wf.Namespace, "resourceVersion": wf.ResourceVersion})
		logCtx.Info("Back-filling resource version")
		res, err := session.Update(tableName).
			Set("resourceversion", wf.ResourceVersion).
			Where(db.Cond{"name": wf.Name}).
			And(db.Cond{"namespace": wf.Namespace}).
			And(db.Cond{"resourceversion": nil}).
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
