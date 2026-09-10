package sqldb

import (
	"context"
	"net"
	"net/netip"
	"runtime"
	"testing"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
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

// TestNewSessionProxyInitialConnection verifies that the initial connection is
// retried with backoff when the database is not yet reachable, so components
// don't crash-loop while waiting for the database to start.
// https://github.com/argoproj/argo-workflows/issues/8797
func TestNewSessionProxyInitialConnection(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	t.Run("retries transient connection errors with backoff", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows reports refused connections with a different error (\"connectex: ...actively refused it\") that isNetworkError does not classify as transient")
		}

		// Reserve a port with no listener so connection attempts are refused.
		listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
		require.NoError(t, err)
		port := listener.Addr().(*net.TCPAddr).Port
		require.NoError(t, listener.Close())

		start := time.Now()
		_, err = NewSessionProxy(ctx, SessionProxyConfig{
			DBConfig: config.DBConfig{
				DBReconnectConfig: &config.DBReconnectConfig{
					MaxRetries:    2,
					RetryMultiple: 2.0,
				},
				PostgreSQL: &config.PostgreSQLConfig{
					DatabaseConfig: config.DatabaseConfig{
						Database: dbName,
						Host:     "127.0.0.1",
						Port:     port,
					},
				},
			},
			Username: userName,
			Password: password,
		})
		elapsed := time.Since(start)

		require.Error(t, err)
		require.ErrorContains(t, err, "connection refused")
		// With the default 100ms base delay, maxRetries=2 and retryMultiple=2.0,
		// the proxy waits 200ms and then 400ms between the three attempts.
		assert.GreaterOrEqual(t, elapsed, 500*time.Millisecond)
	})

	t.Run("does not retry non-transient errors", func(t *testing.T) {
		start := time.Now()
		_, err := NewSessionProxy(ctx, SessionProxyConfig{
			// No database is configured, which is not a network error.
			DBConfig: config.DBConfig{
				DBReconnectConfig: &config.DBReconnectConfig{
					MaxRetries:       3,
					BaseDelaySeconds: 5,
				},
			},
			Username: userName,
			Password: password,
		})
		elapsed := time.Since(start)

		require.Error(t, err)
		require.ErrorContains(t, err, "no databases are configured")
		// The first retry would only happen after a multi-second delay, so a
		// fast failure proves the error was not retried.
		assert.Less(t, elapsed, 5*time.Second)
	})
}
