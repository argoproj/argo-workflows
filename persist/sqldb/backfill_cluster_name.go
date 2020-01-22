package sqldb

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"upper.io/db.v3"
	"upper.io/db.v3/lib/sqlbuilder"
)

type backfillClusterName struct {
	clusterName string
	tableName   string
}

func (s backfillClusterName) String() string {
	return fmt.Sprintf("backfillClusterName{%s,%s}", s.clusterName, s.tableName)
}

func (s backfillClusterName) Apply(session sqlbuilder.Database) error {
	log.WithField("clustername", s.clusterName).Info("Back-filling cluster name")
	rs, err := session.
		Select("uid").
		From(s.tableName).
		Where(db.Cond{"clustername": nil}).
		Query()
	if err != nil {
		return err
	}
	for rs.Next() {
		uid := ""
		err := rs.Scan(&uid)
		if err != nil {
			return err
		}
		logCtx := log.WithFields(log.Fields{"clustername": s.clusterName, "uid": uid})
		logCtx.Info("Back-filling cluster name")
		res, err := session.
			Update(s.tableName).
			Set("clustername", s.clusterName).
			Where(db.Cond{"clustername": nil}).
			And(db.Cond{"uuid": uid}).
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
