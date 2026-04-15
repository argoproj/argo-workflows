package sqldb

import (
	"context"
	"runtime"
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

type mysqlVariant struct {
	image       string
	waitMessage string
}

var mysqlVariants = map[string]mysqlVariant{
	"MySQL":   {image: "mysql:8.4", waitMessage: "port: 3306  MySQL Community Server"},
	"MariaDB": {image: "mariadb:11.4", waitMessage: "mariadbd: ready for connections"},
}

func setupMySQLArchiveTest(ctx context.Context, t *testing.T, v mysqlVariant) (WorkflowArchive, func()) {
	t.Helper()

	c, err := testmysql.Run(ctx,
		v.image,
		testmysql.WithDatabase("argo"),
		testmysql.WithUsername("argo"),
		testmysql.WithPassword("argo"),
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog(v.waitMessage).WithStartupTimeout(60*time.Second),
				wait.ForListeningPort("3306/tcp"),
			)),
	)
	require.NoError(t, err)

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

	return NewWorkflowArchive(proxy, "test", "", instanceid.NewService("")), func() {
		proxy.Close()
		testcontainers.TerminateContainer(c) //nolint:errcheck
	}
}

// TestMySQLListWorkflows verifies that JSON_EXTRACT/JSON_UNQUOTE queries in
// ListWorkflows execute correctly against both MySQL and MariaDB.
func TestMySQLListWorkflows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test requires Linux container")
	}

	for name, variant := range mysqlVariants {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			archive, cleanup := setupMySQLArchiveTest(ctx, t, variant)
			defer cleanup()

			now := metav1.Now()
			err := archive.ArchiveWorkflow(ctx, &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-wf",
					Namespace:         "default",
					UID:               types.UID("test-uid-001"),
					CreationTimestamp: now,
					Labels: map[string]string{
						"workflows.argoproj.io/archive-strategy": "Persisted",
						"env":                                    "test",
					},
				},
				Spec: wfv1.WorkflowSpec{
					Arguments: wfv1.Arguments{
						Parameters: []wfv1.Parameter{
							{Name: "msg", Value: wfv1.AnyStringPtr("hello")},
						},
					},
				},
				Status: wfv1.WorkflowStatus{
					Phase:      wfv1.WorkflowSucceeded,
					StartedAt:  now,
					FinishedAt: now,
					Progress:   "1/1",
					Message:    "completed",
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
		})
	}
}
