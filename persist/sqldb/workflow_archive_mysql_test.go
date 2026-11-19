//go:build !windows

package sqldb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/argoproj/argo-workflows/v4/config"
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

	return newContainerArchive(ctx, t, c, "3306/tcp", func(dc config.DatabaseConfig) config.DBConfig {
		return config.DBConfig{MySQL: &config.MySQLConfig{DatabaseConfig: dc}}
	})
}

// TestMySQLListWorkflows verifies that JSON_EXTRACT/JSON_UNQUOTE queries and paging in
// ListWorkflows execute correctly against both MySQL and MariaDB.
func TestMySQLListWorkflows(t *testing.T) {
	for name, variant := range usqldb.MySQLVariants {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			archive := setupMySQLArchiveTest(ctx, t, variant)
			testListWorkflowsPaging(ctx, t, archive)
		})
	}
}
