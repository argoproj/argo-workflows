package sqldb

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

// TestSkipIfIndexExistsString pins what the migration docs print for the guarded index
// steps: only the statement for the change's own database type, never another type's SQL.
func TestSkipIfIndexExistsString(t *testing.T) {
	tests := []struct {
		dbType sqldb.DBType
		want   []string
	}{
		{
			dbType: sqldb.MySQL,
			want: []string{
				"create index idx_argo_offloaded_wf_templates_uid on argo_offloaded_workflow_templates (uid, namespace)",
				"no statement for mysql",
			},
		},
		{
			dbType: sqldb.Postgres,
			want: []string{
				"create index idx_argo_offloaded_wf_templates_uid on argo_offloaded_workflow_templates (uid)",
				"create index idx_argo_offloaded_wf_templates_namespace on argo_offloaded_workflow_templates (clustername, namespace)",
			},
		},
	}
	for _, tc := range tests {
		t.Run(string(tc.dbType), func(t *testing.T) {
			var got []string
			for _, change := range MigrateChanges("test", "argo_workflows", tc.dbType) {
				if s, ok := change.(skipIfIndexExists); ok {
					got = append(got, s.String())
				}
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
