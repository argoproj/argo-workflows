package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/upper/db/v4"

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
	require.ErrorContains(t, err, "session proxy is closed")

	// An operation that failed before Close must not reopen the proxy later.
	err = proxy.reconnectIfStale(ctx, sess)
	require.ErrorContains(t, err, "session proxy is closed")
	assert.Equal(t, 1, sess.closeCalls)
}

func TestSessionProxyFailedReconnectAllowsLaterAttempt(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	sess := &lifecycleSessionStub{}
	// Missing credentials fail immediately, without a database or retry delays.
	proxy := &SessionProxy{sess: sess}

	require.ErrorContains(t, proxy.Reconnect(ctx), "insufficient authentication information")
	assert.Equal(t, 1, sess.closeCalls)

	err := proxy.With(ctx, func(db.Session) error {
		t.Error("operation ran without a successful reconnection")
		return nil
	})
	// A new operation must attempt to connect, rather than permanently rejecting
	// the proxy as closed or reusing the session discarded by the first attempt.
	require.ErrorContains(t, err, "insufficient authentication information")
	assert.Equal(t, 1, sess.closeCalls)
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
	require.ErrorContains(t, err, "session proxy is closed")
	assert.Equal(t, 1, sess.closeCalls)
}

func TestSessionProxyDisconnectedStaleReconnectDoesNotSkip(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	stale := &lifecycleSessionStub{}
	// Another caller replaced the stale session but its later reconnect failed.
	// The different session pointer must not be mistaken for a working session.
	disconnected := &lifecycleSessionStub{}
	proxy := &SessionProxy{sess: disconnected, disconnected: true}

	err := proxy.reconnectIfStale(ctx, stale)
	require.ErrorContains(t, err, "insufficient authentication information")
	assert.Zero(t, stale.closeCalls, "the discarded session must not be closed again")
	assert.Zero(t, disconnected.closeCalls, "the failed reconnect already closed this session")
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

func TestSessionProxyDisconnectedTransactionDoesNotReconnect(t *testing.T) {
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
