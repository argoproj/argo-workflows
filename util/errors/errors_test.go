package errors

import (
	"errors"
	"net"
	"net/url"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type netError string

func (n netError) Error() string   { return string(n) }
func (n netError) Timeout() bool   { return false }
func (n netError) Temporary() bool { return false }

var (
	tlsHandshakeTimeoutErr net.Error = netError("net/http: TLS handshake timeout")
	ioTimeoutErr           net.Error = netError("i/o timeout")
	connectionTimedout     net.Error = netError("connection timed out")
	transientErr           net.Error = netError("this error is transient")
	transientExitErr                 = exec.ExitError{
		ProcessState: &os.ProcessState{},
		Stderr:       []byte("this error is transient"),
	}
)

const transientEnvVarKey = "TRANSIENT_ERROR_PATTERN"

func TestIsTransientErr(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		assert.False(t, IsTransientErr(nil))
	})
	t.Run("ResourceQuotaConflictErr", func(t *testing.T) {
		assert.False(t, IsTransientErr(apierr.NewConflict(schema.GroupResource{}, "", nil)))
		assert.True(t, IsTransientErr(apierr.NewConflict(schema.GroupResource{Group: "v1", Resource: "resourcequotas"}, "", nil)))
	})
	t.Run("ExceededQuotaErr", func(t *testing.T) {
		assert.False(t, IsTransientErr(apierr.NewForbidden(schema.GroupResource{}, "", nil)))
		assert.True(t, IsTransientErr(apierr.NewForbidden(schema.GroupResource{Group: "v1", Resource: "pods"}, "", errors.New("exceeded quota"))))
	})
	t.Run("TooManyRequestsDNS", func(t *testing.T) {
		assert.True(t, IsTransientErr(apierr.NewTooManyRequests("", 0)))
	})
	t.Run("DNSError", func(t *testing.T) {
		assert.True(t, IsTransientErr(&net.DNSError{}))
	})
	t.Run("OpError", func(t *testing.T) {
		assert.True(t, IsTransientErr(&net.OpError{}))
	})
	t.Run("UnknownNetworkError", func(t *testing.T) {
		assert.True(t, IsTransientErr(net.UnknownNetworkError("")))
	})
	t.Run("ConnectionClosedErr", func(t *testing.T) {
		assert.False(t, IsTransientErr(&url.Error{Err: errors.New("")}))
		assert.True(t, IsTransientErr(&url.Error{Err: errors.New("Connection closed by foreign host")}))
	})
	t.Run("TLSHandshakeTimeout", func(t *testing.T) {
		assert.True(t, IsTransientErr(tlsHandshakeTimeoutErr))
	})
	t.Run("IOHandshakeTimeout", func(t *testing.T) {
		assert.True(t, IsTransientErr(ioTimeoutErr))
	})
	t.Run("ConnectionTimeout", func(t *testing.T) {
		assert.True(t, IsTransientErr(connectionTimedout))
	})
	t.Run("TransientErrorPattern", func(t *testing.T) {
		_ = os.Setenv(transientEnvVarKey, "this error is transient")
		assert.True(t, IsTransientErr(transientErr))
		assert.True(t, IsTransientErr(&transientExitErr))

		_ = os.Setenv(transientEnvVarKey, "this error is not transient")
		assert.False(t, IsTransientErr(transientErr))
		assert.False(t, IsTransientErr(&transientExitErr))

		_ = os.Setenv(transientEnvVarKey, "")
		assert.False(t, IsTransientErr(transientErr))

		_ = os.Unsetenv(transientEnvVarKey)
	})
	t.Run("ExplicitTransientErr", func(t *testing.T) {
		assert.True(t, IsTransientErr(NewErrTransient("")))
	})
}
