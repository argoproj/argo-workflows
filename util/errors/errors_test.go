package errors

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type netError string

func (n netError) Error() string   { return string(n) }
func (n netError) Timeout() bool   { return false }
func (n netError) Temporary() bool { return false }

func urlError(errString string) *url.Error {
	return &url.Error{
		Op:  "Get",
		URL: "https://argoproj.github.io",
		Err: errors.New(errString),
	}
}

var (
	tlsHandshakeTimeoutErr net.Error = netError("net/http: TLS handshake timeout")
	ioTimeoutErr           net.Error = netError("i/o timeout")
	connectionTimedoutErr  net.Error = netError("connection timed out")
	connectionResetErr     net.Error = netError("connection reset by peer")
	transientErr           net.Error = netError("this error is transient")
	transientExitErr                 = exec.ExitError{
		ProcessState: &os.ProcessState{},
		Stderr:       []byte("this error is transient"),
	}

	connectionClosedUErr    *url.Error = urlError("Connection closed by foreign host")
	tlsHandshakeTimeoutUErr *url.Error = urlError("net/http: TLS handshake timeout")
	ioTimeoutUErr           *url.Error = urlError("i/o timeout")
	connectionTimedoutUErr  *url.Error = urlError("connection timed out")
	connectionResetUErr     *url.Error = urlError("connection reset by peer")
	EOFUErr                 *url.Error = urlError("EOF")
	connectionRefusedErr    *url.Error = urlError("connect: connection refused")
)

const transientEnvVarKey = "TRANSIENT_ERROR_PATTERN"

func TestIsTransientErr(t *testing.T) {
	hook := logging.NewTestHook()
	logger := logging.NewTestLogger(logging.Info, logging.Text, hook)
	ctx := logging.WithLogger(context.Background(), logger)
	defer hook.Reset()

	t.Run("Nil", func(t *testing.T) {
		assert.False(t, IsTransientErr(ctx, nil))
		assert.Nil(t, hook.LastEntry())
	})
	t.Run("ResourceQuotaConflictErr", func(t *testing.T) {
		assert.False(t, IsTransientErr(ctx, apierr.NewConflict(schema.GroupResource{}, "", nil)))
		assert.Contains(t, hook.LastEntry().Msg, "Non-transient error")
		assert.True(t, IsTransientErr(ctx, apierr.NewConflict(schema.GroupResource{Group: "v1", Resource: "resourcequotas"}, "", nil)))
		assert.Contains(t, hook.LastEntry().Msg, "Transient error")
	})
	t.Run("ResourceQuotaTimeoutErr", func(t *testing.T) {
		assert.False(t, IsTransientErr(ctx, apierr.NewInternalError(errors.New(""))))
		assert.True(t, IsTransientErr(ctx, apierr.NewInternalError(errors.New("resource quota evaluation timed out"))))
	})
	t.Run("ExceededQuotaErr", func(t *testing.T) {
		assert.False(t, IsTransientErr(ctx, apierr.NewForbidden(schema.GroupResource{}, "", nil)))
		assert.True(t, IsTransientErr(ctx, apierr.NewForbidden(schema.GroupResource{Group: "v1", Resource: "pods"}, "", errors.New("exceeded quota"))))
	})
	t.Run("TooManyRequestsDNS", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, apierr.NewTooManyRequests("", 0)))
	})
	t.Run("DNSError", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, &net.DNSError{}))
	})
	t.Run("OpError", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, &net.OpError{}))
	})
	t.Run("UnknownNetworkError", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, net.UnknownNetworkError("")))
	})
	t.Run("TLSHandshakeTimeout", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, tlsHandshakeTimeoutErr))
	})
	t.Run("IOHandshakeTimeout", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, ioTimeoutErr))
	})
	t.Run("ConnectionTimeout", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, connectionTimedoutErr))
	})
	t.Run("ConnectionReset", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, connectionResetErr))
	})
	t.Run("TransientErrorPattern", func(t *testing.T) {
		t.Setenv(transientEnvVarKey, "this error is transient")
		assert.True(t, IsTransientErr(ctx, transientErr))
		assert.True(t, IsTransientErr(ctx, &transientExitErr))

		t.Setenv(transientEnvVarKey, "this error is not transient")
		assert.False(t, IsTransientErr(ctx, transientErr))
		assert.False(t, IsTransientErr(ctx, &transientExitErr))

		t.Setenv(transientEnvVarKey, "")
		assert.False(t, IsTransientErr(ctx, transientErr))
	})
	t.Run("ExplicitTransientErr", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, NewErrTransient("")))
	})
	t.Run("ConnectionRefusedTransientErr", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, connectionRefusedErr))
	})
}

func TestIsTransientUErr(t *testing.T) {
	hook := logging.NewTestHook()
	logger := logging.NewTestLogger(logging.Info, logging.Text, hook)
	ctx := logging.WithLogger(logging.TestContext(t.Context()), logger)
	defer hook.Reset()

	t.Run("NonExceptionalUErr", func(t *testing.T) {
		assert.False(t, IsTransientErr(ctx, &url.Error{Err: errors.New("")}))
	})
	t.Run("ConnectionClosedUErr", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, connectionClosedUErr))
	})
	t.Run("TLSHandshakeTimeoutUErr", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, tlsHandshakeTimeoutUErr))
	})
	t.Run("IOHandshakeTimeoutUErr", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, ioTimeoutUErr))
	})
	t.Run("ConnectionTimeoutUErr", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, connectionTimedoutUErr))
	})
	t.Run("ConnectionResetUErr", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, connectionResetUErr))
	})
	t.Run("EOFUErr", func(t *testing.T) {
		assert.True(t, IsTransientErr(ctx, EOFUErr))
	})
}
