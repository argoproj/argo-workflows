package pprof

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// freePort returns a port nothing is listening on, so the tests below do not
// depend on 6060 being free on the machine running them.
func freePort(t *testing.T) string {
	t.Helper()
	var lc net.ListenConfig
	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	require.NoError(t, ln.Close())
	return port
}

// dial reports whether something accepts a connection at addr.
func dial(ctx context.Context, addr string) error {
	d := net.Dialer{Timeout: time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	return conn.Close()
}

func get(t *testing.T, url string) (int, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(body)
}

func TestEnabledReadsTheEnvironment(t *testing.T) {
	require.False(t, Enabled(), "must be off when ARGO_PPROF is unset")
	t.Setenv("ARGO_PPROF", "false")
	require.False(t, Enabled())
	// Only the exact string "true" turns it on, so a typo leaves the endpoints
	// off rather than exposing them.
	t.Setenv("ARGO_PPROF", "1")
	require.False(t, Enabled())
	t.Setenv("ARGO_PPROF", "TRUE")
	require.False(t, Enabled())
	t.Setenv("ARGO_PPROF", "true")
	require.True(t, Enabled())
}

// The endpoints are a debugging aid in someone else's workflow pod: nothing
// may listen unless they asked for it.
func TestListenLoopbackIsSilentWhenDisabled(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	port := freePort(t)

	Init(ctx)
	addr := ListenLoopback(ctx, port)

	require.Empty(t, addr, "nothing should be bound with ARGO_PPROF unset")
	require.Error(t, dial(ctx, net.JoinHostPort("127.0.0.1", port)),
		"nothing should be listening with ARGO_PPROF unset")
}

func TestListenLoopbackServesProfilesWhenEnabled(t *testing.T) {
	t.Setenv("ARGO_PPROF", "true")
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	defer cancel()
	port := freePort(t)

	Init(ctx)
	ListenLoopback(ctx, port)

	base := "http://" + net.JoinHostPort("127.0.0.1", port)
	// The heap profile is the one #17283 needs; assert on it specifically
	// rather than only on the index.
	code, body := get(t, base+"/debug/pprof/heap?debug=1")
	require.Equal(t, http.StatusOK, code)
	require.Contains(t, body, "heap profile")

	code, _ = get(t, base+"/debug/pprof/")
	require.Equal(t, http.StatusOK, code)
}

// Binding the pod IP would publish an unauthenticated memory dump to the whole
// cluster network, so the listener must be loopback-only.
//
// This asserts on the address the listener actually bound rather than trying
// to bind 0.0.0.0 on the same port and expecting that to succeed: whether a
// wildcard bind conflicts with an existing loopback bind is platform-specific,
// and that version of the test passed on macOS and failed on Linux with
// "address already in use" while the code under test was correct.
func TestListenLoopbackDoesNotBindOutsideLoopback(t *testing.T) {
	t.Setenv("ARGO_PPROF", "true")
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	defer cancel()

	Init(ctx)
	addr := ListenLoopback(ctx, freePort(t))
	require.NotEmpty(t, addr, "the listener should have started")

	host, _, err := net.SplitHostPort(addr)
	require.NoError(t, err)
	ip := net.ParseIP(host)
	require.NotNil(t, ip, "bound host %q is not an IP", host)
	require.True(t, ip.IsLoopback(), "pprof bound %s, which is not a loopback address", addr)
	require.False(t, ip.IsUnspecified(), "pprof bound the wildcard address %s", addr)
}

func TestListenLoopbackStopsWithTheContext(t *testing.T) {
	t.Setenv("ARGO_PPROF", "true")
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	port := freePort(t)

	Init(ctx)
	ListenLoopback(ctx, port)
	code, _ := get(t, "http://"+net.JoinHostPort("127.0.0.1", port)+"/debug/pprof/")
	require.Equal(t, http.StatusOK, code)

	cancel()
	require.Eventually(t, func() bool {
		return dial(context.WithoutCancel(ctx), net.JoinHostPort("127.0.0.1", port)) != nil
	}, 5*time.Second, 20*time.Millisecond, "listener must close when ctx is done")
}

// A port already in use must not take the executor down with it.
func TestListenLoopbackSurvivesABusyPort(t *testing.T) {
	t.Setenv("ARGO_PPROF", "true")
	ctx := logging.TestContext(t.Context())

	var lc net.ListenConfig
	busy, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer busy.Close()
	_, port, err := net.SplitHostPort(busy.Addr().String())
	require.NoError(t, err)

	Init(ctx)
	var addr string
	require.NotPanics(t, func() { addr = ListenLoopback(ctx, port) })
	require.Empty(t, addr, "a port that cannot be bound reports no address")
}
