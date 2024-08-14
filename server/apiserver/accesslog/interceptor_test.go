package accesslog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestInterceptor(t *testing.T) {
	logOutput := bytes.NewBufferString("")
	log.SetOutput(logOutput)

	realHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler := Interceptor(realHandler)
	handler.ServeHTTP(rr, req)

	expectedLogContains := []string{
		"level=info",
		"method=GET",
		"path=/test",
		"size=0",
		"status=200",
		"duration=",
	}

	for _, key := range expectedLogContains {
		require.Contains(t, logOutput.String(), key, "Interceptor did not log the correct information")
	}

	require.Equal(t, http.StatusOK, rr.Code, "Interceptor did not call the next handler correctly")
	require.Equal(t, "/test", log.WithFields(log.Fields{}).WithField("path", "/test").Data["path"], "Interceptor did not log the correct path")
	require.Equal(t, "GET", log.WithFields(log.Fields{}).WithField("method", "GET").Data["method"], "Interceptor did not log the correct method")
}
