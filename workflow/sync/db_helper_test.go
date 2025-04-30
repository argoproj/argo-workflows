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

	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
)

const (
	testDBName     = `sync`
	testDBUser     = `user`
	testDBPassword = `pass`
)

// createTestDBSession creates a test database session
func createTestDBSession(t *testing.T, dbType sqldb.DBType) (dbInfo, func(), config.SyncConfig, error) {
	t.Helper()

	ctx := context.Background()
	var cfg config.SyncConfig
	var termContainerFn func()
	var err error

	switch dbType {
	case sqldb.Postgres:
		cfg, termContainerFn, err = setupPostgresContainer(t, ctx)
	case sqldb.MySQL:
		cfg, termContainerFn, err = setupMySQLContainer(t, ctx)
	}
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}

	info := dbInfo{
		config:  dbConfigFromConfig(&cfg),
		session: dbSessionFromConfigWithCreds(ctx, &cfg, testDBUser, testDBPassword),
	}
	require.NotNil(t, info.session, "failed to create database session")
	deferfn := func() {
		info.session.Close()
		termContainerFn()
	}

	info.migrate(ctx)
	require.NotNil(t, info.session, "failed to migrate database")

	// Mark this controller as alive immediately
	_, err = info.session.Collection(info.config.controllerTable).
		Insert(&controllerHealthRecord{
			Controller: info.config.controllerName,
			Time:       time.Now(),
		})
	if err != nil {
		info.session.Close()
		info.session = nil
		return info, deferfn, cfg, err
	}

	return info, deferfn, cfg, nil
}

// setupPostgresContainer sets up a Postgres test container and returns the config and cleanup function
func setupPostgresContainer(t *testing.T, ctx context.Context) (config.SyncConfig, func(), error) {
	postgresContainer, err := testpostgres.Run(ctx,
		"postgres:17.4-alpine",
		testpostgres.WithDatabase(testDBName),
		testpostgres.WithUsername(testDBUser),
		testpostgres.WithPassword(testDBPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
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
func setupMySQLContainer(t *testing.T, ctx context.Context) (config.SyncConfig, func(), error) {
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

