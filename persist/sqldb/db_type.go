package sqldb

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-sql-driver/mysql"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"upper.io/db.v3"
)

type dbType string

const (
	MySQL    dbType = "mysql"
	Postgres dbType = "postgres"
)

func dbTypeFor(session db.Database) dbType {
	switch session.Driver().(*sql.DB).Driver().(type) {
	case *mysql.MySQLDriver:
		return MySQL
	}
	return Postgres
}

func (t dbType) labelsClause(requirements labels.Requirements) (db.Compound, error) {
	var conds []db.Compound
	for _, r := range requirements {
		cond, err := t.requirementToCondition(r)
		if err != nil {
			return nil, err
		}
		conds = append(conds, cond)
	}
	return db.And(conds...), nil
}

func (t dbType) requirementToCondition(r labels.Requirement) (db.Compound, error) {
	// Should we "sanitize our inputs"? No.
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
	// Valid label values must be 63 characters or less and must be empty or begin and end with an alphanumeric character ([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and alphanumerics between.
	// https://kb.objectrocket.com/postgresql/casting-in-postgresql-570#string+to+integer+casting
	switch r.Operator() {
	case selection.DoesNotExist:
		return db.Raw(fmt.Sprintf("not exists (select 1 from %s where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s')", archiveLabelsTableName, r.Key())), nil
	case selection.Equals, selection.DoubleEquals:
		return db.Raw(fmt.Sprintf("exists (select 1 from %s where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s' and value = '%s')", archiveLabelsTableName, r.Key(), r.Values().List()[0])), nil
	case selection.In:
		return db.Raw(fmt.Sprintf("exists (select 1 from %s where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s' and value in ('%s'))", archiveLabelsTableName, r.Key(), strings.Join(r.Values().List(), "', '"))), nil
	case selection.NotEquals:
		return db.Raw(fmt.Sprintf("not exists (select 1 from %s where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s' and value = '%s')", archiveLabelsTableName, r.Key(), r.Values().List()[0])), nil
	case selection.NotIn:
		return db.Raw(fmt.Sprintf("not exists (select 1 from %s where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s' and value in ('%s'))", archiveLabelsTableName, r.Key(), strings.Join(r.Values().List(), "', '"))), nil
	case selection.Exists:
		return db.Raw(fmt.Sprintf("exists (select 1 from %s where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s')", archiveLabelsTableName, r.Key())), nil
	case selection.GreaterThan:
		i, err := strconv.Atoi(r.Values().List()[0])
		if err != nil {
			return nil, err
		}
		return db.Raw(fmt.Sprintf("exists (select 1 from %s where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s' and cast(value as %s) > %d)", archiveLabelsTableName, r.Key(), t.intType(), i)), nil
	case selection.LessThan:
		i, err := strconv.Atoi(r.Values().List()[0])
		if err != nil {
			return nil, err
		}
		return db.Raw(fmt.Sprintf("exists (select 1 from %s where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = '%s' and cast(value as %s) < %d)", archiveLabelsTableName, r.Key(), t.intType(), i)), nil
	}
	return nil, fmt.Errorf("operation %v is not supported", r.Operator())
}

func (t dbType) intType() string {
	if t == MySQL {
		return "signed"
	}
	return "int"
}
