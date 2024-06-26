package store

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/instanceid"
)

func TestInitDB(t *testing.T) {
	conn, err := initDB()
	require.NoError(t, err)
	defer conn.Close()
	t.Run("TestTablesCreated", func(t *testing.T) {
		err = sqlitex.Execute(conn, `select name from sqlite_master where type='table'`, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				name := stmt.ColumnText(0)
				assert.Contains(t, []string{workflowTableName, workflowLabelsTableName}, name)
				return nil
			},
		})
		require.NoError(t, err)
	})
	t.Run("TestForeignKeysEnabled", func(t *testing.T) {
		err = sqlitex.Execute(conn, `pragma foreign_keys`, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				assert.Equal(t, "1", stmt.ColumnText(0))
				return nil
			},
		})
		require.NoError(t, err)
	})
	t.Run("TestIndexesCreated", func(t *testing.T) {
		var indexes []string
		err = sqlitex.Execute(conn, `select name from sqlite_master where type='index'`, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				name := stmt.ColumnText(0)
				indexes = append(indexes, name)
				return nil
			},
		})
		require.NoError(t, err)
		assert.Contains(t, indexes, "idx_instanceid")
		assert.Contains(t, indexes, "idx_name_value")
	})
	t.Run("TestForeignKeysAdded", func(t *testing.T) {
		err = sqlitex.Execute(conn, `pragma foreign_key_list('argo_workflows_labels')`, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				assert.Equal(t, "argo_workflows", stmt.ColumnText(2))
				assert.Equal(t, "uid", stmt.ColumnText(3))
				assert.Equal(t, "uid", stmt.ColumnText(4))
				assert.Equal(t, "CASCADE", stmt.ColumnText(6))
				return nil
			},
		})
		require.NoError(t, err)
	})
}

func TestStoreOperation(t *testing.T) {
	instanceIdSvc := instanceid.NewService("my-instanceid")
	conn, err := initDB()
	require.NoError(t, err)
	store := SQLiteStore{
		conn:            conn,
		instanceService: instanceIdSvc,
	}
	t.Run("TestAddWorkflow", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			require.NoError(t, store.Add(generateWorkflow(i)))
		}
		num, err := store.CountWorkflows(context.Background(), "argo", "", metav1.ListOptions{})
		require.NoError(t, err)
		assert.Equal(t, int64(10), num)
		// Labels are also added
		require.NoError(t, sqlitex.Execute(conn, `select count(*) from argo_workflows_labels`, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				assert.Equal(t, 10*4, stmt.ColumnInt(0))
				return nil
			},
		}))
	})
	t.Run("TestUpdateWorkflow", func(t *testing.T) {
		wf := generateWorkflow(0)
		wf.Labels["test-label-2"] = "value-2"
		require.NoError(t, store.Update(wf))
		// workflow is updated
		require.NoError(t, sqlitex.Execute(conn, `select workflow from argo_workflows where uid = 'uid-0'`, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				w := stmt.ColumnText(0)
				require.NoError(t, json.Unmarshal([]byte(w), &wf))
				assert.Len(t, wf.Labels, 5)
				return nil
			},
		}))
		require.NoError(t, sqlitex.Execute(conn, `select count(*) from argo_workflows_labels where name = 'test-label-2' and value = 'value-2'`, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				assert.Equal(t, 1, stmt.ColumnInt(0))
				return nil
			},
		}))
	})
	t.Run("TestDeleteWorkflow", func(t *testing.T) {
		wf := generateWorkflow(0)
		require.NoError(t, store.Delete(wf))
		// workflow is deleted
		require.NoError(t, sqlitex.Execute(conn, `select count(*) from argo_workflows where uid = 'uid-0'`, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				assert.Equal(t, 0, stmt.ColumnInt(0))
				return nil
			},
		}))
		// labels are also deleted
		require.NoError(t, sqlitex.Execute(conn, `select count(*) from argo_workflows_labels where uid = 'uid-0'`, &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				assert.Equal(t, 0, stmt.ColumnInt(0))
				return nil
			},
		}))
	})
	t.Run("TestListWorkflows", func(t *testing.T) {
		wfList, err := store.ListWorkflows(context.Background(), "argo", "", metav1.ListOptions{Limit: 5})
		require.NoError(t, err)
		assert.Len(t, wfList.Items, 5)
	})
	t.Run("TestListWorkflows name", func(t *testing.T) {
		wfList, err := store.ListWorkflows(context.Background(), "argo", "Exact", metav1.ListOptions{Limit: 5, FieldSelector: "metadata.name=flow"})
		require.NoError(t, err)
		assert.Len(t, wfList.Items, 0)

		wfList, err = store.ListWorkflows(context.Background(), "argo", "Exact", metav1.ListOptions{Limit: 5, FieldSelector: "metadata.name=workflow-1"})
		require.NoError(t, err)
		assert.Len(t, wfList.Items, 1)

		wfList, err = store.ListWorkflows(context.Background(), "argo", "", metav1.ListOptions{Limit: 5, FieldSelector: "metadata.name=workflow-1"})
		require.NoError(t, err)
		assert.Len(t, wfList.Items, 1)
	})
	t.Run("TestListWorkflows namePrefix", func(t *testing.T) {
		wfList, err := store.ListWorkflows(context.Background(), "argo", "Prefix", metav1.ListOptions{Limit: 5, FieldSelector: "metadata.name=flow"})
		require.NoError(t, err)
		assert.Len(t, wfList.Items, 0)

		wfList, err = store.ListWorkflows(context.Background(), "argo", "Prefix", metav1.ListOptions{Limit: 5, FieldSelector: "metadata.name=workflow-"})
		require.NoError(t, err)
		assert.Len(t, wfList.Items, 5)

		wfList, err = store.ListWorkflows(context.Background(), "argo", "Prefix", metav1.ListOptions{Limit: 5, FieldSelector: "metadata.name=workflow-1"})
		require.NoError(t, err)
		assert.Len(t, wfList.Items, 1)
	})
	t.Run("TestListWorkflows namePattern", func(t *testing.T) {
		wfList, err := store.ListWorkflows(context.Background(), "argo", "Contains", metav1.ListOptions{Limit: 5, FieldSelector: "metadata.name=non-existing-pattern"})
		require.NoError(t, err)
		assert.Len(t, wfList.Items, 0)

		wfList, err = store.ListWorkflows(context.Background(), "argo", "Contains", metav1.ListOptions{Limit: 5, FieldSelector: "metadata.name=flow"})
		require.NoError(t, err)
		assert.Len(t, wfList.Items, 5)

		wfList, err = store.ListWorkflows(context.Background(), "argo", "Contains", metav1.ListOptions{Limit: 5, FieldSelector: "metadata.name=workflow-1"})
		require.NoError(t, err)
		assert.Len(t, wfList.Items, 1)
	})
	t.Run("TestCountWorkflows", func(t *testing.T) {
		num, err := store.CountWorkflows(context.Background(), "argo", "", metav1.ListOptions{})
		require.NoError(t, err)
		assert.Equal(t, int64(9), num)
	})
}

func generateWorkflow(uid int) *wfv1.Workflow {
	return &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{
		UID:       types.UID(fmt.Sprintf("uid-%d", uid)),
		Name:      fmt.Sprintf("workflow-%d", uid),
		Namespace: "argo",
		Labels: map[string]string{
			"workflows.argoproj.io/completed":             "true",
			"workflows.argoproj.io/phase":                 "Succeeded",
			"workflows.argoproj.io/controller-instanceid": "my-instanceid",
			"test-label": fmt.Sprintf("label-%d", uid),
		},
	}}
}
