//go:build !windows

package sqldb

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	sutils "github.com/argoproj/argo-workflows/v4/server/utils"
	"github.com/argoproj/argo-workflows/v4/util/instanceid"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	usqldb "github.com/argoproj/argo-workflows/v4/util/sqldb"
)

// setupMySQLArchiveTest starts a MySQL or MariaDB container, runs migrations, and returns a WorkflowArchive.
func setupMySQLArchiveTest(ctx context.Context, t *testing.T, v usqldb.MySQLVariant) WorkflowArchive {
	t.Helper()

	c, err := testmysql.Run(ctx,
		v.Image,
		testmysql.WithDatabase("argo"),
		testmysql.WithUsername("argo"),
		testmysql.WithPassword("argo"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog(v.WaitMessage).WithStartupTimeout(60*time.Second),
				wait.ForListeningPort("3306/tcp"),
			)),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		if termErr := testcontainers.TerminateContainer(c); termErr != nil {
			t.Logf("failed to terminate container: %s", termErr)
		}
	})

	host, err := c.Host(ctx)
	require.NoError(t, err)
	p, err := c.MappedPort(ctx, "3306/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(p.Port())
	require.NoError(t, err)

	proxy, err := usqldb.NewSessionProxy(ctx, usqldb.SessionProxyConfig{
		DBConfig: config.DBConfig{
			MySQL: &config.MySQLConfig{
				DatabaseConfig: config.DatabaseConfig{
					Database: "argo",
					Host:     host,
					Port:     port,
				},
			},
		},
		Username: "argo",
		Password: "argo",
	})
	require.NoError(t, err)

	err = Migrate(ctx, proxy.Session(), "test", "argo_workflows", proxy.DBType())
	require.NoError(t, err)

	t.Cleanup(func() { proxy.Close() })

	return NewWorkflowArchive(proxy, "test", "", instanceid.NewService(""))
}

// TestMySQLListWorkflows verifies that JSON_EXTRACT/JSON_UNQUOTE queries in
// ListWorkflows execute correctly against both MySQL and MariaDB.
func TestMySQLListWorkflows(t *testing.T) {
	for name, variant := range usqldb.MySQLVariants {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			archive := setupMySQLArchiveTest(ctx, t, variant)

			now := metav1.Now()
			err := archive.ArchiveWorkflow(ctx, &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-wf",
					Namespace:         "default",
					UID:               types.UID("test-uid-001"),
					CreationTimestamp: now,
					Labels:            map[string]string{"env": "test"},
					Annotations:       map[string]string{"note": "integration-test"},
				},
				Spec: wfv1.WorkflowSpec{
					Suspend: new(true),
					Arguments: wfv1.Arguments{
						Parameters: []wfv1.Parameter{
							{Name: "msg", Value: wfv1.AnyStringPtr("hello")},
						},
					},
				},
				Status: wfv1.WorkflowStatus{
					Phase:             wfv1.WorkflowSucceeded,
					StartedAt:         now,
					FinishedAt:        now,
					Progress:          "1/1",
					Message:           "completed",
					EstimatedDuration: wfv1.EstimatedDuration(30),
				},
			})
			require.NoError(t, err)

			results, err := archive.ListWorkflows(ctx, sutils.ListOptions{Namespace: "default", Limit: 10})
			require.NoError(t, err)
			require.Len(t, results, 1)

			wf := results[0]
			assert.Equal(t, "test-wf", wf.Name)
			assert.Equal(t, wfv1.WorkflowSucceeded, wf.Status.Phase)
			assert.Equal(t, wfv1.Progress("1/1"), wf.Status.Progress)
			assert.Equal(t, "completed", wf.Status.Message)
			assert.Equal(t, "test", wf.GetLabels()["env"])
			assert.Equal(t, "integration-test", wf.GetAnnotations()["note"])
			assert.Equal(t, new(true), wf.Spec.Suspend)
			assert.Equal(t, "hello", wf.Spec.Arguments.Parameters[0].Value.String())
			assert.Equal(t, wfv1.EstimatedDuration(30), wf.Status.EstimatedDuration)
		})
	}
}
