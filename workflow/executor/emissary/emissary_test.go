package emissary

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
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

// A re-run, as when an init container runs again in a recreated pod sandbox,
// replaces the read-only template from before, even with a shorter one, and
// leaves no temporary file behind.
func TestWriteTemplate_ReplacesAnEarlierCopy(t *testing.T) {
	dir := t.TempDir()
	e := emissary{varRunArgo: dir}
	require.NoError(t, e.WriteTemplate(wfv1.Template{Name: "a-longer-older-template"}))
	require.NoError(t, e.WriteTemplate(wfv1.Template{Name: "newer"}))

	data, err := os.ReadFile(filepath.Join(dir, "template"))
	require.NoError(t, err)
	var got wfv1.Template
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "newer", got.Name)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "template", entries[0].Name())
}
