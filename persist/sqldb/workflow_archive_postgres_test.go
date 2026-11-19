//go:build !windows

package sqldb

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// setupPostgresArchiveTest starts a PostgreSQL container, runs migrations, and returns a WorkflowArchive.
func setupPostgresArchiveTest(ctx context.Context, t *testing.T) WorkflowArchive {
	t.Helper()

	c, err := testpostgres.Run(ctx,
		"postgres:17.4-alpine",
		testpostgres.WithDatabase("argo"),
		testpostgres.WithUsername("argo"),
		testpostgres.WithPassword("argo"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)

	return newContainerArchive(ctx, t, c, "5432/tcp", func(dc config.DatabaseConfig) config.DBConfig {
		return config.DBConfig{PostgreSQL: &config.PostgreSQLConfig{DatabaseConfig: dc}}
	})
}

// TestPostgresListWorkflows verifies ListWorkflows paging and field extraction against PostgreSQL.
func TestPostgresListWorkflows(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	archive := setupPostgresArchiveTest(ctx, t)
	testListWorkflowsPaging(ctx, t, archive)
}
