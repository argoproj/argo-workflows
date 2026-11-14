package externalsecrets

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultVersionAuthorizationAndResponseValidation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(file, []byte("local-synthetic"), 0600))
	mode := "ok"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "local-synthetic", r.Header.Get("X-Vault-Token"))
		assert.Equal(t, http.MethodGet, r.Method)
		switch mode {
		case "denied":
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "DO-NOT-LEAK")
			return
		case "oversized":
			fmt.Fprint(w, strings.Repeat("x", 131073))
			return
		case "redirect":
			http.Redirect(w, r, "http://169.254.169.254/", http.StatusFound)
			return
		}
		if strings.Contains(r.URL.Path, "/metadata/") {
			fmt.Fprint(w, `{"data":{"current_version":2,"versions":{"1":{"destroyed":true},"2":{"destroyed":false,"deletion_time":""}}}}`)
			return
		}
		assert.Equal(t, "2", r.URL.Query().Get("version"))
		fmt.Fprint(w, `{"data":{"metadata":{"version":2},"data":{"token":"synthetic-value"}}}`)
	}))
	defer server.Close()
	config := VaultConfig{Address: server.URL, Mount: "secret", Entry: "namespace/example", Field: "token", TokenFile: file}
	_, err := NewVault(config)
	require.ErrorContains(t, err, "HTTPS")
	config.AllowHTTPDevelopment = true
	v, err := NewVault(config)
	require.NoError(t, err)
	meta, err := v.Metadata(t.Context())
	require.NoError(t, err)
	require.Equal(t, 2, meta.Current)
	require.False(t, meta.Versions["1"].Active(time.Now()))
	value, err := v.Read(t.Context(), 2)
	require.NoError(t, err)
	require.Equal(t, "synthetic-value", string(value))
	for _, failure := range []string{"denied", "oversized", "redirect"} {
		mode = failure
		_, err = v.Read(t.Context(), 2)
		require.Error(t, err)
		require.NotContains(t, err.Error(), "DO-NOT-LEAK")
	}
	mode = "ok"
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = v.Read(ctx, 2)
	require.Error(t, err)
	for _, entry := range []string{"../escape", "/leading", "entry?version=1", "entry%2fother"} {
		config.Entry = entry
		_, err = NewVault(config)
		require.Error(t, err)
	}
}
