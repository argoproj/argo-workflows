package grpc

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
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

func TestSanitizeCookieHeader(t *testing.T) {
	tests := []struct {
		name     string
		cookies  []string
		expected []string
	}{
		{
			name:     "no cookie header",
			cookies:  nil,
			expected: nil,
		},
		{
			name:     "valid cookies preserved",
			cookies:  []string{"authorization=Bearer abc.def.ghi; foo=bar"},
			expected: []string{"authorization=Bearer abc.def.ghi; foo=bar"},
		},
		{
			name:     "non-ASCII pair dropped, valid pairs kept",
			cookies:  []string{"authorization=token123; theme=h\xc3\xa9llo; foo=bar"},
			expected: []string{"authorization=token123; foo=bar"},
		},
		{
			name:     "control character pair dropped",
			cookies:  []string{"foo=bar; bad=a\x01b; authorization=token123"},
			expected: []string{"foo=bar; authorization=token123"},
		},
		{
			name:     "all pairs invalid removes header",
			cookies:  []string{"bad=h\xc3\xa9llo"},
			expected: nil,
		},
		{
			name:     "multiple header values sanitized independently",
			cookies:  []string{"foo=bar", "bad=h\xc3\xa9llo; ok=1"},
			expected: []string{"foo=bar", "ok=1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(logging.TestContext(t.Context()), http.MethodGet, "/", nil)
			require.NoError(t, err)
			for _, c := range tt.cookies {
				req.Header.Add("Cookie", c)
			}
			SanitizeCookieHeader(req)
			assert.Equal(t, tt.expected, req.Header.Values("Cookie"))
			for _, v := range req.Header.Values("Cookie") {
				for i := 0; i < len(v); i++ {
					assert.GreaterOrEqual(t, v[i], uint8(0x20))
					assert.LessOrEqual(t, v[i], uint8(0x7e))
				}
			}
		})
	}

	t.Run("authorization cookie still parseable", func(t *testing.T) {
		req, err := http.NewRequestWithContext(logging.TestContext(t.Context()), http.MethodGet, "/", nil)
		require.NoError(t, err)
		req.Header.Add("Cookie", "unrelated=h\xc3\xa9llo; authorization=Bearer token123; other=1")
		SanitizeCookieHeader(req)
		cookie, err := req.Cookie("authorization")
		require.NoError(t, err)
		assert.Equal(t, "Bearer token123", cookie.Value)
	})
}

func TestNewMuxHandler(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	grpcHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	handler := NewMuxHandler(grpcHandler, httpHandler)

	t.Run("gRPC request handling", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/", nil)
		require.NoError(t, err)
		req.ProtoMajor = 2
		req.Header.Set("Content-Type", "application/grpc")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		assert.Equal(t, 201, recorder.Result().StatusCode)
	})

	t.Run("HTTP request handling", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		assert.Equal(t, 202, recorder.Result().StatusCode)
	})
}
