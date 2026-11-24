package pprof

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestInit(t *testing.T) {
	previous := http.DefaultServeMux
	t.Cleanup(func() { http.DefaultServeMux = previous })
	ctx := logging.TestContext(t.Context())
	for _, setting := range []string{"", "false", "true"} {
		t.Run("ARGO_PPROF="+setting, func(t *testing.T) {
			t.Setenv("ARGO_PPROF", setting)
			handler := Init(ctx)

			get := func(path string) *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				http.DefaultServeMux.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil))
				return rec
			}
			index := get("/debug/pprof/")
			heap := get("/debug/pprof/heap")
			if setting != "true" {
				require.Nil(t, handler)
				require.Equal(t, http.StatusNotFound, index.Code)
				require.Equal(t, http.StatusNotFound, heap.Code)
				return
			}
			require.Same(t, http.DefaultServeMux, handler)
			require.Equal(t, http.StatusOK, index.Code)
			require.Contains(t, index.Body.String(), "goroutine")
			require.Equal(t, http.StatusOK, heap.Code)
			require.Equal(t, "application/octet-stream", heap.Header().Get("Content-Type"))
			reader, err := gzip.NewReader(heap.Body)
			require.NoError(t, err)
			defer reader.Close()
			data, err := io.ReadAll(reader)
			require.NoError(t, err)
			require.NotEmpty(t, data)
		})
	}
}
