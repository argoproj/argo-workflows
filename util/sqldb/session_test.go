package sqldb

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
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
			hostConfig.PortBindings = nat.PortMap{
				"5432/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
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

func TestSessionReconnect(t *testing.T) {
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
