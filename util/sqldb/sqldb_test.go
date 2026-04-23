package sqldb

import (
	"context"
	"runtime"
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

type mysqlVariant struct {
	image       string
	waitMessage string
}

var mysqlVariants = map[string]mysqlVariant{
	"MySQL":   {image: "mysql:8.4", waitMessage: "port: 3306  MySQL Community Server"},
	"MariaDB": {image: "mariadb:11.4", waitMessage: "mariadbd: ready for connections"},
}

func setupMySQLContainer(ctx context.Context, t *testing.T, v mysqlVariant) (config.DBConfig, func()) {
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

	return config.DBConfig{
		MySQL: &config.MySQLConfig{
			DatabaseConfig: config.DatabaseConfig{
				Database: "argo",
				Host:     host,
				Port:     port,
			},
		},
	}, func() { testcontainers.TerminateContainer(c) } //nolint:errcheck
}

// TestMySQLSessionConnect verifies that CreateDBSessionWithCreds can connect
// to both MySQL and MariaDB. MariaDB requires AllowNativePasswords.
func TestMySQLSessionConnect(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test requires Linux container")
	}

	for name, variant := range mysqlVariants {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			dbConfig, cleanup := setupMySQLContainer(ctx, t, variant)
			defer cleanup()

			sess, dbType, err := CreateDBSessionWithCreds(dbConfig, "argo", "argo")
			require.NoError(t, err)
			defer sess.Close()

			require.Equal(t, MySQL, dbType)
			require.NoError(t, sess.Ping())
		})
	}
}
