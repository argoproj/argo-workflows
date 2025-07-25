package accesslog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestInterceptor(t *testing.T) {
	logOutput := bytes.NewBufferString("")
	logger := logging.NewSlogLoggerCustom(logging.Info, logging.Text, logOutput)

	realHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler := NewLoggingInterceptor(logger).Interceptor(realHandler)
	handler.ServeHTTP(rr, req)

	expectedLogContains := []string{
		"level=INFO",
		"method=GET",
		"path=/test",
		"size=0",
		"status=200",
		"duration=",
	}

	for _, key := range expectedLogContains {
		assert.Contains(t, logOutput.String(), key, "Interceptor did not log the correct information")
	}

	assert.Equal(t, http.StatusOK, rr.Code, "Interceptor did not call the next handler correctly")

	// Test that the logger can create fields correctly
	testFields := logger.WithField("path", "/test").WithField("method", "GET")
	assert.NotNil(t, testFields, "Logger should be able to create fields")
}
