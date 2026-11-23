package apiserver

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// startServer serves handler on a random local port. The returned channel is
// closed once Serve has returned and checkServeErr has accepted its error
// (checkServeErr calls os.Exit for any error that isn't a graceful shutdown).
func startServer(t *testing.T, handler http.Handler) (*http.Server, string, <-chan struct{}) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	conn, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	srv := &http.Server{Handler: handler}
	served := make(chan struct{})
	go func() {
		(&argoServer{}).checkServeErr(ctx, "httpServer", srv.Serve(conn))
		close(served)
	}()
	return srv, "http://" + conn.Addr().String(), served
}

func getWithContext(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func TestShutdownHTTPServer(t *testing.T) {
	t.Run("InFlightRequestFinishes", func(t *testing.T) {
		started, release := make(chan struct{}), make(chan struct{})
		srv, url, served := startServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-release
			_, _ = io.WriteString(w, "done")
		}))

		type result struct {
			body string
			err  error
		}
		resp := make(chan result, 1)
		go func() {
			r, err := getWithContext(t.Context(), url)
			if err != nil {
				resp <- result{err: err}
				return
			}
			defer r.Body.Close()
			b, err := io.ReadAll(r.Body)
			resp <- result{string(b), err}
		}()
		<-started

		ctx := logging.TestContext(t.Context())
		shutdownDone := make(chan struct{})
		go func() {
			shutdownHTTPServer(ctx, srv, 10*time.Second)
			close(shutdownDone)
		}()
		<-served // Serve returns as soon as shutdown starts, and the process survived checkServeErr

		r0, err := getWithContext(t.Context(), url)
		if err == nil {
			r0.Body.Close()
		}
		require.Error(t, err, "new connections should be refused once shutdown starts")

		close(release)
		r := <-resp
		require.NoError(t, r.err)
		assert.Equal(t, "done", r.body)
		<-shutdownDone
	})

	t.Run("StuckRequestIsClosedAfterTimeout", func(t *testing.T) {
		started := make(chan struct{})
		srv, url, served := startServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(started)
			<-r.Context().Done() // a watch stream that never ends on its own
		}))
		go func() {
			if r, err := getWithContext(t.Context(), url); err == nil {
				r.Body.Close()
			}
		}()
		<-started

		start := time.Now()
		shutdownHTTPServer(logging.TestContext(t.Context()), srv, 100*time.Millisecond)
		assert.Less(t, time.Since(start), 5*time.Second)
		<-served
	})
}
