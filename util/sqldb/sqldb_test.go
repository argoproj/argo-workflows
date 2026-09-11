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

// Regression test for https://github.com/argoproj/argo-workflows/issues/16707:
// driver-level entries in persistence.mysql.options (e.g. tls) must be applied
// as DSN options, not sent to the server verbatim as SET statements.
func TestBuildMySQLConfigDriverOptions(t *testing.T) {
	cfg := &config.MySQLConfig{
		DatabaseConfig: config.DatabaseConfig{
			Host:     "localhost",
			Port:     3306,
			Database: "argo",
		},
		Options: map[string]string{
			"tls":         "skip-verify",
			"readTimeout": "5s",
			// a genuine server system variable, which must remain a param
			"transaction_isolation": "'READ-COMMITTED'",
		},
	}
	mysqlCfg, err := buildMySQLConfig(cfg, "user", "pass", 7*time.Second)
	require.NoError(t, err)
	require.Equal(t, "skip-verify", mysqlCfg.TLSConfig)
	require.Equal(t, 5*time.Second, mysqlCfg.ReadTimeout)
	require.NotContains(t, mysqlCfg.Params, "tls")
	require.NotContains(t, mysqlCfg.Params, "readTimeout")
	require.Equal(t, "'READ-COMMITTED'", mysqlCfg.Params["transaction_isolation"])
	// fields set directly must survive
	require.Equal(t, "user", mysqlCfg.User)
	require.Equal(t, "pass", mysqlCfg.Passwd)
	require.Equal(t, "localhost:3306", mysqlCfg.Addr)
	require.Equal(t, "argo", mysqlCfg.DBName)
	require.True(t, mysqlCfg.ParseTime)
	require.True(t, mysqlCfg.AllowNativePasswords)
	require.Equal(t, 7*time.Second, mysqlCfg.Timeout)
}
