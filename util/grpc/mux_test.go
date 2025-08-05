package grpc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncomingHeaderMatcher(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{
			name:  "Content-Length header is filtered",
			key:   "Content-Length",
			valid: false,
		},
		{
			name:  "Connection header is filtered",
			key:   "Connection",
			valid: false,
		},
		{
			name:  "X-Custom-Header is allowed",
			key:   "X-Custom-Header",
			valid: true,
		},
		{
			name:  "mixed case filtered header",
			key:   "content-Length",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, valid := IncomingHeaderMatcher(tt.key)
			assert.Equal(t, tt.key, key)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

func TestNewMuxHandler(t *testing.T) {
	grpcHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	})
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(202)
	})

	handler := NewMuxHandler(grpcHandler, httpHandler)

	t.Run("gRPC request handling", func(t *testing.T) {
		req, err := http.NewRequest("POST", "/", nil)
		require.NoError(t, err)
		req.ProtoMajor = 2
		req.Header.Set("Content-Type", "application/grpc")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		assert.Equal(t, 201, recorder.Result().StatusCode)
	})

	t.Run("HTTP request handling", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		assert.Equal(t, 202, recorder.Result().StatusCode)
	})
}
