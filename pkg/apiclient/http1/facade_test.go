package http1

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestFacade_do(t *testing.T) {
	f := Facade{baseURL: "http://my-url"}
	u, err := f.url("GET", "/{namespace}/{name}", &metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{"foo": "1"}})
	require.NoError(t, err)
	assert.Equal(t, "http://my-url/my-ns/?labels.foo=1", u.String())

	u, err = f.url("DELETE", "/{namespace}/{name}", &metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{"foo": "1"}})
	require.NoError(t, err)
	assert.Equal(t, "http://my-url/my-ns/?labels.foo=1", u.String())
}

func TestFacade_do_RFC9110HeaderValues(t *testing.T) {
	tests := []struct {
		name        string
		headerName  string
		headerValue string
	}{
		{
			name:        "simple token value",
			headerName:  "X-Request-ID",
			headerValue: "abc-123",
		},
		{
			name:        "single URI value containing colon",
			headerName:  "Example-URI",
			headerValue: "http://example.com/a.html",
		},
		// Example values from: https://datatracker.ietf.org/doc/html/rfc9110#section-5.5-10
		{
			name:        "URI list value with comma and multiple colons",
			headerName:  "Example-URIs",
			headerValue: "\"http://example.com/a.html,foo\",\"http://without-a-comma.example.com/\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get(tt.headerName); got != tt.headerValue {
					http.Error(w, fmt.Sprintf("unexpected header %s: got %q want %q", tt.headerName, got, tt.headerValue), http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("{}"))
			}))
			defer srv.Close()

			f := Facade{
				baseURL:    srv.URL,
				headers:    []string{fmt.Sprintf("%s:%s", tt.headerName, tt.headerValue)},
				httpClient: srv.Client(),
			}

			ctx := logging.TestContext(t.Context())
			var out map[string]any
			err := f.Get(ctx, struct{}{}, &out, "/")
			require.NoError(t, err, "RFC 9110-compliant header value should be accepted and preserved")
		})
	}
}

func TestFacade_do_InvalidHeaderFormat(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "missing separator colon",
			header: "Authorization",
		},
		{
			name:   "empty header name",
			header: ":Bearer abc",
		},
		{
			name:   "empty header entry",
			header: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Facade{
				baseURL: "http://my-url",
				headers: []string{tt.header},
			}

			ctx := logging.TestContext(t.Context())
			var out map[string]any
			err := f.Get(ctx, struct{}{}, &out, "/")
			require.Error(t, err)
			require.ErrorContains(t, err, "additional headers must be colon(:)-separated")
		})
	}
}
