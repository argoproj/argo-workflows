package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upper/db/v4"
	"k8s.io/apimachinery/pkg/labels"
)

func Test_labelsClause(t *testing.T) {
	tests := []struct {
		name         string
		dbType       dbType
		requirements labels.Requirements
		want         db.RawExpr
	}{
		{"Empty", Postgres, requirements(""), db.RawExpr{}},
		{"DoesNotExist", Postgres, requirements("!foo"), *db.Raw("not exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo')")},
		{"Equals", Postgres, requirements("foo=bar"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value = 'bar')")},
		{"DoubleEquals", Postgres, requirements("foo==bar"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value = 'bar')")},
		{"In", Postgres, requirements("foo in (bar,baz)"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value in ('bar', 'baz'))")},
		{"NotEquals", Postgres, requirements("foo != bar"), *db.Raw("not exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value = 'bar')")},
		{"NotIn", Postgres, requirements("foo notin (bar,baz)"), *db.Raw("not exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and value in ('bar', 'baz'))")},
		{"Exists", Postgres, requirements("foo"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo')")},
		{"GreaterThanPostgres", Postgres, requirements("foo>2"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and cast(value as int) > 2)")},
		{"GreaterThanMySQL", MySQL, requirements("foo>2"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and cast(value as signed) > 2)")},
		{"LessThanPostgres", Postgres, requirements("foo<2"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and cast(value as int) < 2)")},
		{"LessThanMySQL", MySQL, requirements("foo<2"), *db.Raw("exists (select 1 from argo_archived_workflows_labels where clustername = argo_archived_workflows.clustername and uid = argo_archived_workflows.uid and name = 'foo' and cast(value as signed) < 2)")},
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
