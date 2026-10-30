package emissary

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestEmissary_ReadsUnderItsOwnRoot(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	dir := t.TempDir()
	param := filepath.Join(dir, "outputs", "parameters", "tmp", "p")
	require.NoError(t, os.MkdirAll(filepath.Dir(param), 0o755))
	require.NoError(t, os.WriteFile(param, []byte("v"), 0o644))
	stdout := filepath.Join(dir, "ctr", "main", "stdout")
	require.NoError(t, os.MkdirAll(filepath.Dir(stdout), 0o755))
	require.NoError(t, os.WriteFile(stdout, []byte("out"), 0o644))

	e := New(dir)

	got, err := e.GetFileContents("main", "/tmp/p")
	require.NoError(t, err)
	assert.Equal(t, "v", got)

	r, err := e.GetOutputStream(ctx, "main", false)
	require.NoError(t, err)
	defer func() { _ = r.Close() }()
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "out", string(data))
}
