package sqldb

import (
	"context"
	"errors"
	"fmt"
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

	err = sessionProxy.Session(ctx).Ping()
	require.NoError(t, err)
	cancel()

	err = sessionProxy.Session(ctx).Ping()
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

func TestIsNetworkErrorIgnoresContextErrors(t *testing.T) {
	sp := &SessionProxy{}
	assert.False(t, sp.isNetworkError(context.Canceled))
	assert.False(t, sp.isNetworkError(context.DeadlineExceeded))
	assert.False(t, sp.isNetworkError(fmt.Errorf("query: %w", context.DeadlineExceeded)))
	assert.True(t, sp.isNetworkError(errors.New("read tcp: i/o timeout")))
}

func TestSessionWithCancelsStatement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("This test uses the Linux container image and therefore cannot be performed on the Windows platform")
	}

	ctx := logging.TestContext(t.Context())
	cfg, cancel, err := setupPostgresContainer(ctx, t)
	require.NoError(t, err)
	defer cancel()

	sessionProxy, err := NewSessionProxy(ctx, SessionProxyConfig{
		DBConfig: cfg,
		Username: userName,
		Password: password,
	})
	require.NoError(t, err)
	origSess := sessionProxy.sess

	queryCtx, queryCancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer queryCancel()
	start := time.Now()
	err = sessionProxy.With(queryCtx, func(s db.Session) error {
		_, execErr := s.SQL().Exec("SELECT pg_sleep(30)")
		return execErr
	})
	require.Error(t, err)
	assert.Less(t, time.Since(start), 10*time.Second, "statement should be cancelled with its context")
	assert.Same(t, origSess, sessionProxy.sess, "a cancelled statement must not trigger a reconnect")

	// The server-side statement is cancelled too, not just abandoned by the client.
	assert.Eventually(t, func() bool {
		var count int
		row, err := sessionProxy.Session(ctx).SQL().QueryRow(
			"SELECT count(*) FROM pg_stat_activity WHERE state = 'active' AND query = 'SELECT pg_sleep(30)'")
		if err != nil || row.Scan(&count) != nil {
			return false
		}
		return count == 0
	}, 10*time.Second, 200*time.Millisecond)
}

// A reconnect must not break a caller that is still using the session it
// took before the reconnect: With hands fn a session and releases the lock,
// so fn can still be running when another caller reconnects.
func TestSessionProxyReconnectWhileSessionInUse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("This test uses the Linux container image and therefore cannot be performed on the Windows platform")
	}
	ctx := logging.TestContext(t.Context())
	cfg, terminate, err := setupPostgresContainer(ctx, t)
	require.NoError(t, err)
	t.Cleanup(terminate)
	proxy, err := NewSessionProxy(ctx, SessionProxyConfig{DBConfig: cfg, Username: userName, Password: password})
	require.NoError(t, err)
	t.Cleanup(func() { _ = proxy.Close() })

	query := func(s db.Session) error {
		row, err := s.SQL().QueryRow("SELECT 1")
		if err != nil {
			return err
		}
		var one int
		return row.Scan(&one)
	}

	inFn := make(chan struct{})
	release := make(chan struct{})
	type result struct {
		err      error
		panicked string
	}
	results := make(chan result, 1)
	go func() {
		var r result
		defer func() {
			if p := recover(); p != nil {
				r.panicked = fmt.Sprint(p)
			}
			results <- r
		}()
		r.err = proxy.With(ctx, func(s db.Session) error {
			if err := query(s); err != nil {
				return err
			}
			close(inFn)
			<-release
			// The next statement on the session taken before the reconnect.
			return query(s)
		})
	}()

	<-inFn
	require.NoError(t, proxy.Reconnect(ctx))
	close(release)
	r := <-results

	assert.Empty(t, r.panicked, "a statement on the session in use panicked after a reconnect")
	assert.NoError(t, r.err, "a statement on the session in use failed after a reconnect")
}

// releaseProbeSession satisfies only the calls SessionProxy.With makes around
// the callback, so the reference-counting tests below need no database.
type releaseProbeSession struct {
	db.Session
	closed   bool
	closeErr error
}

func (s *releaseProbeSession) WithContext(ctx context.Context) db.Session { return s }
func (s *releaseProbeSession) Close() error                               { s.closed = true; return s.closeErr }

func TestSessionProxyWithReleasesReferenceOnPanic(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	probe := &releaseProbeSession{}
	live := &liveSession{sess: probe}
	sp := &SessionProxy{sess: live}

	// The happy path still balances the reference.
	require.NoError(t, sp.With(ctx, func(db.Session) error { return nil }))
	require.Equal(t, int32(0), live.refs.Load())

	// A panicking callback must also release the reference, so a later
	// reconnect or Close is not blocked forever on a phantom caller.
	assert.Panics(t, func() {
		_ = sp.With(ctx, func(db.Session) error { panic("callback panic") })
	})
	require.Equal(t, int32(0), live.refs.Load(),
		"With must release its session reference even when fn panics")

	// With the reference released, retiring the session must actually close
	// it instead of leaking the session for the life of the process.
	require.NoError(t, live.retire())
	assert.True(t, probe.closed, "retired session with no remaining references must be closed")
}

func TestSessionProxyCloseReturnsSessionError(t *testing.T) {
	probe := &releaseProbeSession{closeErr: errors.New("close failed")}
	sp := &SessionProxy{sess: &liveSession{sess: probe}}

	// An unused session closes on Close, which must report the session's own
	// close error instead of dropping it.
	require.ErrorIs(t, sp.Close(), probe.closeErr)
	assert.True(t, probe.closed)
	require.NoError(t, sp.Close(), "a second Close is a no-op")

	// While a With caller still holds the session, Close only retires it and
	// returns nil; the deferred close runs on the last release.
	probe = &releaseProbeSession{closeErr: errors.New("close failed")}
	live := &liveSession{sess: probe}
	live.refs.Add(1)
	sp = &SessionProxy{sess: live}
	require.NoError(t, sp.Close())
	assert.False(t, probe.closed, "a session still in use must not be closed")
	live.release()
	assert.True(t, probe.closed)
}
