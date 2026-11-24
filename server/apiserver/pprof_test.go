package apiserver

import (
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestServerProfilingListener(t *testing.T) {
	previousMux := http.DefaultServeMux
	t.Cleanup(func() { http.DefaultServeMux = previousMux })

	for _, tc := range []struct {
		name     string
		setting  string
		occupied bool
	}{
		{name: "enabled", setting: "true"},
		{name: "disabled", setting: "false"},
		{name: "profiling port occupied", setting: "true", occupied: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ARGO_PPROF", tc.setting)
			ctx := logging.TestContext(t.Context())
			profilePort := 0
			if tc.occupied {
				var lc net.ListenConfig
				occupied, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
				require.NoError(t, err)
				t.Cleanup(func() { occupied.Close() })
				profilePort = occupied.Addr().(*net.TCPAddr).Port
			}

			as := &argoServer{}
			apiListener, profiler, err := as.listen(ctx, "127.0.0.1:0", profilePort)
			require.NoError(t, err)
			if profiler != nil {
				t.Cleanup(func() { profiler.Close() })
			}
			api := &http.Server{
				Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusNoContent)
				}),
				ReadHeaderTimeout: time.Second,
			}
			done := make(chan error, 1)
			go func() { done <- api.Serve(apiListener) }()
			t.Cleanup(func() {
				require.NoError(t, api.Close())
				require.ErrorIs(t, <-done, http.ErrServerClosed)
			})
			client := &http.Client{Timeout: 5 * time.Second}
			t.Cleanup(client.CloseIdleConnections)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+apiListener.Addr().String()+"/", nil)
			require.NoError(t, err)
			apiResponse, err := client.Do(req)
			require.NoError(t, err)
			defer apiResponse.Body.Close()
			require.Equal(t, http.StatusNoContent, apiResponse.StatusCode)

			if tc.setting != "true" || tc.occupied {
				require.Nil(t, profiler)
				return
			}
			require.NotNil(t, profiler)
			host, _, err := net.SplitHostPort(profiler.Addr)
			require.NoError(t, err)
			require.Equal(t, "127.0.0.1", host)
			req, err = http.NewRequestWithContext(ctx, http.MethodGet, "http://"+profiler.Addr+"/debug/pprof/heap", nil)
			require.NoError(t, err)
			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"))
			heap, err := gzip.NewReader(resp.Body)
			require.NoError(t, err)
			defer heap.Close()
			data, err := io.ReadAll(heap)
			require.NoError(t, err)
			require.NotEmpty(t, data)
		})
	}
}
