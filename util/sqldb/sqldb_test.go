//go:build !windows

package sqldb

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// setupMySQLContainer starts a MySQL or MariaDB container and returns the corresponding DBConfig.
func setupMySQLContainer(ctx context.Context, t *testing.T, v MySQLVariant) config.DBConfig {
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

	return config.DBConfig{
		MySQL: &config.MySQLConfig{
			DatabaseConfig: config.DatabaseConfig{
				Database: "argo",
				Host:     host,
				Port:     port,
			},
		},
	}
}

// TestMySQLSessionConnect verifies that CreateDBSessionWithCreds can connect
// to both MySQL and MariaDB. MariaDB requires AllowNativePasswords.
func TestMySQLSessionConnect(t *testing.T) {
	for name, variant := range MySQLVariants {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			dbConfig := setupMySQLContainer(ctx, t, variant)

			sess, _, err := CreateDBSessionWithCreds(dbConfig, "argo", "argo")
			require.NoError(t, err)
			defer sess.Close()

			require.NoError(t, sess.Ping())
		})
	}
}
