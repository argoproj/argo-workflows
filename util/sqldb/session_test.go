package sqldb

import (
	"context"
	"database/sql"
	"fmt"
	"net/netip"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	testmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

const (
	dbName    = "session_proxy_test"
	userName  = "username"
	password  = "password"
	fixedPort = 15432 // Fixed port for PostgreSQL testing to avoid conflicts with standard port 5432
)

func setupPostgresContainer(ctx context.Context, t *testing.T) (config.DBConfig, func(), error) {
	postgresContainer, err := testpostgres.Run(ctx,
		"postgres:17.4-alpine",
		testpostgres.WithDatabase(dbName),
		testpostgres.WithUsername(userName),
		testpostgres.WithPassword(password),
		testcontainers.WithHostConfigModifier(func(hostConfig *container.HostConfig) {
			// Set up fixed port binding: map container port 5432 to host port 15432
			hostConfig.PortBindings = network.PortMap{
				network.MustParsePort("5432/tcp"): []network.PortBinding{
					{
						HostIP:   netip.MustParseAddr("0.0.0.0"),
						HostPort: "15432",
					},
				},
			}
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(15*time.Second)),
	)
	if err != nil {
		return config.DBConfig{}, nil, err
	}

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)
	// Use the fixed port instead of querying the dynamically assigned port
	port := fixedPort

	reconnectOpts := config.DBReconnectConfig{
		MaxRetries:       20,
		BaseDelaySeconds: 2,
		MaxDelaySeconds:  20,
		RetryMultiple:    2.0,
	}

	dbConfig := config.DBConfig{
		DBReconnectConfig: &reconnectOpts,
		PostgreSQL: &config.PostgreSQLConfig{
			DatabaseConfig: config.DatabaseConfig{
				Database: dbName,
				Host:     host,
				Port:     port,
			},
		},
	}

	termContainerFn := func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}

	return dbConfig, termContainerFn, nil
}

func setupMySQLContainerWithDatabase(ctx context.Context, t *testing.T, database string) (config.DBConfig, func(), error) {
	mysqlContainer, err := testmysql.Run(ctx,
		"mysql:8.4.5",
		testmysql.WithDatabase(database),
		testmysql.WithUsername(userName),
		testmysql.WithPassword(password),
	)
	if err != nil {
		return config.DBConfig{}, nil, err
	}

	host, err := mysqlContainer.Host(ctx)
	require.NoError(t, err)
	portS, err := mysqlContainer.MappedPort(ctx, "3306/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(portS.Port())
	require.NoError(t, err)

	cfg := config.DBConfig{
		MySQL: &config.MySQLConfig{
			DatabaseConfig: config.DatabaseConfig{
				Database: database,
				Host:     host,
				Port:     port,
			},
		},
	}

	termContainerFn := func() {
		if err := testcontainers.TerminateContainer(mysqlContainer); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}

	return cfg, termContainerFn, nil
}

func TestSessionReconnect(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("This test uses the Linux container image and therefore cannot be performed on the Windows platform")
	}

	ctx := logging.TestContext(t.Context())
	cfg, cancel, err := setupPostgresContainer(ctx, t)
	require.NoError(t, err)

	sessionProxy, err := NewSessionProxy(ctx, SessionProxyConfig{
		DBConfig: cfg,
		Username: userName,
		Password: password,
	})
	require.NoError(t, err)

	err = sessionProxy.Session().Ping()
	require.NoError(t, err)
	cancel()

	err = sessionProxy.Session().Ping()
	require.Error(t, err)

	doneChan := make(chan struct{})
	go func() {
		hasSeenErr := false
		outerErr := sessionProxy.With(ctx, func(s db.Session) error {
			innerErr := s.Ping()
			if innerErr != nil {
				hasSeenErr = true
			}
			return innerErr
		})
		assert.NoError(t, outerErr)
		assert.True(t, hasSeenErr)
		doneChan <- struct{}{}
	}()

	newDBConfig, cancel, err := setupPostgresContainer(ctx, t)
	require.NoError(t, err)
	assert.Equal(t, cfg.PostgreSQL.Host, newDBConfig.PostgreSQL.Host)
	assert.Equal(t, cfg.PostgreSQL.Port, newDBConfig.PostgreSQL.Port)
	<-doneChan
	cancel()
}

func TestPostgresSessionUsesSchema(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("This test uses the Linux container image and therefore cannot be performed on the Windows platform")
	}

	ctx := logging.TestContext(t.Context())
	cfg, cleanup, err := setupPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer cleanup()

	createSQLDB, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", cfg.PostgreSQL.Host, cfg.PostgreSQL.Port, userName, password, cfg.PostgreSQL.Database))
	require.NoError(t, err)
	defer createSQLDB.Close()

	_, err = createSQLDB.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS myschema")
	require.NoError(t, err)

	cfg.PostgreSQL.Schema = "myschema"
	session, dbType, err := CreateDBSessionWithCreds(cfg, userName, password)
	require.NoError(t, err)
	require.Equal(t, Postgres, dbType)
	defer session.Close()

	var currentSchema string
	row, err := session.SQL().QueryRow("SELECT current_schema()")
	require.NoError(t, err)
	err = row.Scan(&currentSchema)
	require.NoError(t, err)
	require.Equal(t, "myschema", currentSchema)
}

func TestMySQLSessionUsesSchema(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("This test uses the Linux container image and therefore cannot be performed on the Windows platform")
	}

	ctx := logging.TestContext(t.Context())
	cfg, cleanup, err := setupMySQLContainerWithDatabase(ctx, t, dbName)
	require.NoError(t, err)
	defer cleanup()

	session, dbType, err := CreateDBSessionWithCreds(cfg, userName, password)
	require.NoError(t, err)
	require.Equal(t, MySQL, dbType)
	defer session.Close()

	var currentDatabase string
	row, err := session.SQL().QueryRow("SELECT DATABASE()")
	require.NoError(t, err)
	err = row.Scan(&currentDatabase)
	require.NoError(t, err)
	require.Equal(t, dbName, currentDatabase)
}
