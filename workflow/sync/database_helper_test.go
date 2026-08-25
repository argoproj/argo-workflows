package sync

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	testmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	testpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
	syncdb "github.com/argoproj/argo-workflows/v4/util/sync/db"
)

const (
	testDBName     = `sync`
	testDBUser     = `user`
	testDBPassword = `pass`
)

// createTestDBSession creates a test database session
func createTestDBSession(ctx context.Context, t *testing.T, dbType sqldb.DBType) (syncdb.Info, func(), config.SyncConfig, error) {
	t.Helper()

	var cfg config.SyncConfig
	var termContainerFn func()
	var err error

	switch dbType {
	case sqldb.Postgres:
		cfg, termContainerFn, err = setupPostgresContainer(ctx, t)
	case sqldb.MySQL:
		cfg, termContainerFn, err = setupMySQLContainer(ctx, t)
	}
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}

	// Create SessionProxy instead of raw Session
	sessionProxy, err := sqldb.NewSessionProxy(ctx, sqldb.SessionProxyConfig{
		DBConfig:   cfg.DBConfig,
		Username:   testDBUser,
		Password:   testDBPassword,
		MaxRetries: 5,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   30 * time.Second,
	})
	if err != nil {
		termContainerFn()
		return syncdb.Info{}, func() {}, cfg, err
	}

	info := syncdb.Info{
		Config:       syncdb.ConfigFromConfig(&cfg),
		SessionProxy: sessionProxy,
	}
	require.NotNil(t, info.SessionProxy, "failed to create database session proxy")
	deferfn := func() {
		info.SessionProxy.Close()
		termContainerFn()
	}

	info.Migrate(ctx)
	require.NotNil(t, info.SessionProxy, "failed to migrate database")

	// Mark this controller as alive immediately
	_, err = info.SessionProxy.Session().Collection(info.Config.ControllerTable).
		Insert(&syncdb.ControllerHealthRecord{
			Controller: info.Config.ControllerName,
			Time:       time.Now(),
		})
	if err != nil {
		info.SessionProxy.Close()
		return info, deferfn, cfg, err
	}

	return info, deferfn, cfg, nil
}

// setupPostgresContainer sets up a Postgres test container and returns the config and cleanup function
func setupPostgresContainer(ctx context.Context, t *testing.T) (config.SyncConfig, func(), error) {
	postgresContainer, err := testpostgres.Run(ctx,
		"postgres:17.4-alpine",
		testpostgres.WithDatabase(testDBName),
		testpostgres.WithUsername(testDBUser),
		testpostgres.WithPassword(testDBPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(15*time.Second)),
	)
	if err != nil {
		return config.SyncConfig{}, nil, err
	}

	host, err := postgresContainer.Host(ctx)
	require.NoError(t, err)
	portS, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(portS.Port())
	require.NoError(t, err)

	cfg := config.SyncConfig{
		ControllerName: "test1",
		DBConfig: config.DBConfig{
			PostgreSQL: &config.PostgreSQLConfig{
				DatabaseConfig: config.DatabaseConfig{
					Database: testDBName,
					Host:     host,
					Port:     port,
				},
			},
		},
	}

	termContainerFn := func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}

	return cfg, termContainerFn, nil
}

// setupMySQLContainer sets up a MySQL test container and returns the config and cleanup function
func setupMySQLContainer(ctx context.Context, t *testing.T) (config.SyncConfig, func(), error) {
	mysqlContainer, err := testmysql.Run(ctx,
		"mysql:8.4.5",
		testmysql.WithDatabase(testDBName),
		testmysql.WithUsername(testDBUser),
		testmysql.WithPassword(testDBPassword),
	)
	if err != nil {
		return config.SyncConfig{}, nil, err
	}

	host, err := mysqlContainer.Host(ctx)
	require.NoError(t, err)
	portS, err := mysqlContainer.MappedPort(ctx, "3306/tcp")
	require.NoError(t, err)
	port, err := strconv.Atoi(portS.Port())
	require.NoError(t, err)

	cfg := config.SyncConfig{
		ControllerName: "test1",
		DBConfig: config.DBConfig{
			MySQL: &config.MySQLConfig{
				DatabaseConfig: config.DatabaseConfig{
					Database: testDBName,
					Host:     host,
					Port:     port,
				},
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
