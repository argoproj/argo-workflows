package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type lifecycleSessionStub struct {
	db.Session
	closeCalls int
	txContext  func(context.Context, func(db.Session) error, *sql.TxOptions) error
}

func (s *lifecycleSessionStub) Close() error {
	s.closeCalls++
	return nil
}

// With binds the session to the caller's context before running fn. Return the
// same stub so the tests can still compare session identity.
func (s *lifecycleSessionStub) WithContext(context.Context) db.Session {
	return s
}

func (s *lifecycleSessionStub) TxContext(ctx context.Context, fn func(db.Session) error, opts *sql.TxOptions) error {
	return s.txContext(ctx, fn, opts)
}

func TestSessionProxyCloseRejectsAutomaticReconnect(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sess := &lifecycleSessionStub{}
	proxy := &SessionProxy{sess: sess}

	require.NoError(t, proxy.Close())
	require.NoError(t, proxy.Close())

	err := proxy.With(ctx, func(db.Session) error {
		t.Error("operation ran after explicit Close")
		return nil
	})
	require.ErrorIs(t, err, errSessionProxyClosed)

	// An operation that failed before Close must not reopen the proxy later.
	err = proxy.reconnectIfStale(ctx, sess)
	require.ErrorIs(t, err, errSessionProxyClosed)
	assert.Equal(t, 1, sess.closeCalls)
}

func TestSessionProxyFailedReconnectKeepsProxyOpen(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sess := &lifecycleSessionStub{}
	// Missing credentials fail immediately, without a database or retry delays.
	proxy := &SessionProxy{sess: sess}

	require.ErrorContains(t, proxy.Reconnect(ctx), "insufficient authentication information")

	// The proxy stays open with the old session, whose pool redials by itself.
	calls := 0
	require.NoError(t, proxy.With(ctx, func(current db.Session) error {
		calls++
		assert.Same(t, sess, current)
		return nil
	}))
	assert.Equal(t, 1, calls)

	// A later operation that fails with a network error tries to reconnect again.
	err := proxy.With(ctx, func(db.Session) error { return io.ErrUnexpectedEOF })
	require.ErrorContains(t, err, "insufficient authentication information")
	assert.Zero(t, sess.closeCalls, "a failed reconnect must not close the old session")
}

// droppingDatabase listens like a database that accepts and drops every
// connection, so each dial is counted and fails with a network error.
func droppingDatabase(t *testing.T) (config.DBConfig, *atomic.Int32) {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	dials := &atomic.Int32{}
	go func() {
		for {
			conn, acceptErr := ln.Accept()
			if acceptErr != nil {
				return
			}
			dials.Add(1)
			conn.Close()
		}
	}()
	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	return config.DBConfig{PostgreSQL: &config.PostgreSQLConfig{
		DatabaseConfig: config.DatabaseConfig{Database: "argo", Host: host, Port: port},
	}}, dials
}

// While the database is down, callers that fail on the same session must share
// one reconnect instead of each running a full retry budget in turn.
func TestSessionProxyFailedReconnectIsShared(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cfg, dials := droppingDatabase(t)
	proxy := &SessionProxy{
		sess:     &lifecycleSessionStub{},
		dbConfig: &cfg, username: "argo", password: "argo",
		maxRetries: 2, baseDelay: 50 * time.Millisecond, maxDelay: 50 * time.Millisecond, retryMultiple: 1,
	}

	start := time.Now()
	require.Error(t, proxy.Reconnect(ctx))
	budgetDials, budgetTime := dials.Swap(0), time.Since(start)

	const callers = 5
	errs := make(chan error, callers)
	start = time.Now()
	for range callers {
		go func() { errs <- proxy.With(ctx, func(db.Session) error { return io.ErrUnexpectedEOF }) }()
	}
	for range callers {
		require.ErrorContains(t, <-errs, "reconnection failed")
	}
	assert.Equal(t, budgetDials, dials.Load(), "queued callers should share one reconnect")
	assert.Less(t, time.Since(start), 2*budgetTime+100*time.Millisecond)
}

// A reconnect in progress must not hold up callers that do not need it, and a
// caller that joins it stops waiting when its own ctx ends.
func TestSessionProxyReconnectDoesNotBlockOtherCallers(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cfg, dials := droppingDatabase(t)
	sess := &lifecycleSessionStub{}
	proxy := &SessionProxy{
		sess:     sess,
		dbConfig: &cfg, username: "argo", password: "argo",
		maxRetries: 2, baseDelay: 500 * time.Millisecond, maxDelay: 500 * time.Millisecond, retryMultiple: 1,
	}

	reconnectDone := make(chan error, 1)
	go func() {
		reconnectDone <- proxy.With(ctx, func(db.Session) error { return io.ErrUnexpectedEOF })
	}()
	require.Eventually(t, func() bool { return dials.Load() > 0 }, 5*time.Second, time.Millisecond)

	start := time.Now()
	assert.Same(t, sess, proxy.Session(ctx), "Session() must not wait for the reconnect")
	require.NoError(t, proxy.With(ctx, func(current db.Session) error {
		assert.Same(t, sess, current)
		return nil
	}), "an operation that works must not wait for the reconnect")

	shortCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	err := proxy.With(shortCtx, func(db.Session) error { return io.ErrUnexpectedEOF })
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Less(t, time.Since(start), 400*time.Millisecond, "callers must not wait for the whole retry budget")

	require.ErrorContains(t, <-reconnectDone, "reconnection failed")
	assert.Zero(t, sess.closeCalls, "a failed reconnect must not close the old session")
}

func TestSessionProxyFailedExplicitReconnectPreservesClose(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sess := &lifecycleSessionStub{}
	proxy := &SessionProxy{sess: sess}
	require.NoError(t, proxy.Close())

	// Explicit Reconnect may reopen a closed proxy, but only after connecting.
	require.ErrorContains(t, proxy.Reconnect(ctx), "insufficient authentication information")
	err := proxy.With(ctx, func(db.Session) error {
		t.Error("operation ran after a failed explicit reopen")
		return nil
	})
	require.ErrorIs(t, err, errSessionProxyClosed)
	assert.Equal(t, 1, sess.closeCalls)
}

func TestSessionProxyStaleReconnectPreservesReplacement(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	stale := &lifecycleSessionStub{}
	replacement := &lifecycleSessionStub{}
	proxy := &SessionProxy{sess: replacement}

	require.NoError(t, proxy.reconnectIfStale(ctx, stale))
	calls := 0
	require.NoError(t, proxy.With(ctx, func(sess db.Session) error {
		calls++
		assert.Same(t, replacement, sess)
		return nil
	}))
	assert.Equal(t, 1, calls)
	assert.Zero(t, replacement.closeCalls)
	assert.Zero(t, stale.closeCalls)
}

func TestSessionProxyWithDoesNotReplayNonRetryableOperation(t *testing.T) {
	for _, tt := range []struct {
		name              string
		operationErr      error
		insideTransaction bool
	}{
		{name: "non-network error", operationErr: errors.New("constraint violation")},
		{name: "network error inside transaction", operationErr: io.EOF, insideTransaction: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			sess := &lifecycleSessionStub{}
			proxy := &SessionProxy{sess: sess, insideTransaction: tt.insideTransaction}
			calls := 0

			err := proxy.With(ctx, func(current db.Session) error {
				calls++
				assert.Same(t, sess, current)
				return tt.operationErr
			})
			require.ErrorIs(t, err, tt.operationErr)
			assert.Equal(t, 1, calls)
			assert.Zero(t, sess.closeCalls)
		})
	}
}

func TestSessionProxyTransactionWithoutSessionDoesNotReconnect(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	proxy := &SessionProxy{insideTransaction: true}

	err := proxy.With(ctx, func(db.Session) error {
		t.Error("operation ran without a transaction session")
		return nil
	})
	require.ErrorContains(t, err, "no active session")
	require.NotContains(t, err.Error(), "authentication")
}

func TestSessionProxyTransactionDoesNotInheritConcurrentClose(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	txSession := &lifecycleSessionStub{}
	transactionStarted := make(chan struct{})
	continueTransaction := make(chan struct{})
	sess := &lifecycleSessionStub{
		txContext: func(_ context.Context, fn func(db.Session) error, _ *sql.TxOptions) error {
			close(transactionStarted)
			<-continueTransaction
			return fn(txSession)
		},
	}
	proxy := &SessionProxy{sess: sess}
	done := make(chan error, 1)
	calls := 0
	go func() {
		done <- proxy.TxWith(ctx, func(tx *SessionProxy) error {
			return tx.With(ctx, func(current db.Session) error {
				calls++
				assert.Same(t, txSession, current)
				return nil
			})
		}, nil)
	}()

	<-transactionStarted
	closeErr := proxy.Close()
	close(continueTransaction)
	require.NoError(t, closeErr)
	require.NoError(t, <-done)
	assert.Equal(t, 1, calls)
	assert.Equal(t, 1, sess.closeCalls)
	assert.Zero(t, txSession.closeCalls)
}
