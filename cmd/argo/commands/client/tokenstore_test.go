package client

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// useTempTokenStore points the token store at a temporary file and selects a
// test Argo Server, restoring both afterwards.
func useTempTokenStore(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "argo", "tokens.json")
	originalPath := tokenStorePath
	originalOpts := ArgoServerOpts
	tokenStorePath = func() (string, error) { return path, nil }
	ArgoServerOpts = apiclient.ArgoServerOpts{URL: "argo.example.com:443", Path: "/argo/", Secure: true}
	t.Cleanup(func() {
		tokenStorePath = originalPath
		ArgoServerOpts = originalOpts
	})
	return path
}

func TestServerKey(t *testing.T) {
	useTempTokenStore(t)
	assert.Equal(t, "https://argo.example.com:443/argo", ServerKey())
}

func TestTokenStoreRoundTrip(t *testing.T) {
	path := useTempTokenStore(t)

	stored, err := LoadToken("https://a")
	require.NoError(t, err)
	assert.Nil(t, stored)

	expiry := time.Now().Add(time.Hour).Truncate(time.Second)
	require.NoError(t, SaveToken("https://a", StoredToken{Token: "Bearer v2:a", Expiry: expiry}))
	require.NoError(t, SaveToken("https://b", StoredToken{Token: "Bearer v2:b"}))

	// Windows has no Unix permission bits; the file is still created in the
	// user's own profile directory.
	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(path)
		require.NoError(t, statErr)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		dirInfo, statErr := os.Stat(filepath.Dir(path))
		require.NoError(t, statErr)
		assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())
	}

	stored, err = LoadToken("https://a")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "Bearer v2:a", stored.Token)
	assert.True(t, expiry.Equal(stored.Expiry))

	deleted, err := DeleteToken("https://a")
	require.NoError(t, err)
	assert.True(t, deleted)
	deleted, err = DeleteToken("https://a")
	require.NoError(t, err)
	assert.False(t, deleted)

	stored, err = LoadToken("https://b")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "Bearer v2:b", stored.Token)
}

func TestLoadTokenCorruptStore(t *testing.T) {
	path := useTempTokenStore(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))
	_, err := LoadToken("https://a")
	require.ErrorContains(t, err, path)
}

func TestStoredTokenExpired(t *testing.T) {
	now := time.Now()
	assert.False(t, StoredToken{}.Expired(now), "an unknown expiry never expires")
	assert.False(t, StoredToken{Expiry: now.Add(time.Minute)}.Expired(now))
	assert.True(t, StoredToken{Expiry: now}.Expired(now))
}

func TestGetAuthStringUsesStoredToken(t *testing.T) {
	useTempTokenStore(t)
	ctx := logging.TestContext(t.Context())
	t.Setenv("ARGO_TOKEN", "")
	require.NoError(t, os.Unsetenv("ARGO_TOKEN"))
	require.NoError(t, SaveToken(ServerKey(), StoredToken{Token: "Bearer v2:stored", Expiry: time.Now().Add(time.Hour)}))

	authString, err := GetAuthString(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Bearer v2:stored", authString)

	t.Run("ARGO_TOKEN wins", func(t *testing.T) {
		t.Setenv("ARGO_TOKEN", "my-token")
		authString, err := GetAuthString(ctx)
		require.NoError(t, err)
		assert.Equal(t, "my-token", authString)
	})
}

func TestGetAuthStringSkipsExpiredStoredToken(t *testing.T) {
	useTempTokenStore(t)
	ctx := logging.TestContext(t.Context())
	t.Setenv("ARGO_TOKEN", "")
	require.NoError(t, os.Unsetenv("ARGO_TOKEN"))
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, SaveToken(ServerKey(), StoredToken{Token: "Bearer v2:expired", Expiry: time.Now().Add(-time.Minute)}))

	authString, _ := GetAuthString(ctx)
	assert.NotEqual(t, "Bearer v2:expired", authString)
}

func TestGetAuthStringIgnoresStoredTokenWithoutArgoServer(t *testing.T) {
	useTempTokenStore(t)
	ctx := logging.TestContext(t.Context())
	t.Setenv("ARGO_TOKEN", "")
	require.NoError(t, os.Unsetenv("ARGO_TOKEN"))
	t.Setenv("KUBECONFIG", filepath.Join(t.TempDir(), "missing"))
	require.NoError(t, SaveToken(ServerKey(), StoredToken{Token: "Bearer v2:stored"}))
	ArgoServerOpts.URL = ""

	authString, _ := GetAuthString(ctx)
	assert.NotEqual(t, "Bearer v2:stored", authString)
}
