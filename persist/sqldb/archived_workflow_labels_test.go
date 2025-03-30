package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upper/db/v4"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

func Test_labelsClause(t *testing.T) {
	tests := []struct {
		name         string
		dbType       sqldb.DBType
		requirements labels.Requirements
		want         db.RawExpr
	}{
		{"Empty", sqldb.Postgres, requirements(""), db.RawExpr{}},
		{"DoesNotExist", sqldb.Postgres, requirements("!foo"), *db.Raw("not exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo')")},
		{"Equals", sqldb.Postgres, requirements("foo=bar"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value = 'bar')")},
		{"DoubleEquals", sqldb.Postgres, requirements("foo==bar"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value = 'bar')")},
		{"In", sqldb.Postgres, requirements("foo in (bar,baz)"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value in ('bar', 'baz'))")},
		{"NotEquals", sqldb.Postgres, requirements("foo != bar"), *db.Raw("not exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value = 'bar')")},
		{"NotIn", sqldb.Postgres, requirements("foo notin (bar,baz)"), *db.Raw("not exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value in ('bar', 'baz'))")},
		{"Exists", sqldb.Postgres, requirements("foo"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo')")},
		{"GreaterThansqldb.Postgres", sqldb.Postgres, requirements("foo>2"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and cast(value as int) > 2)")},
		{"GreaterThanMySQL", sqldb.MySQL, requirements("foo>2"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and cast(value as signed) > 2)")},
		{"LessThansqldb.Postgres", sqldb.Postgres, requirements("foo<2"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and cast(value as int) < 2)")},
		{"LessThanMySQL", sqldb.MySQL, requirements("foo<2"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and cast(value as signed) < 2)")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, req := range tt.requirements {
				got, err := requirementToCondition(tt.dbType, req, archiveTableName, archiveLabelsTableName, true)
				require.NoError(t, err)
				assert.Equal(t, tt.want, *got)
			}
		})
	}
}

func requirements(selector string) []labels.Requirement {
	requirements, err := labels.ParseToRequirements(selector)
	if err != nil {
		panic(err)
	}
	return requirements
}
