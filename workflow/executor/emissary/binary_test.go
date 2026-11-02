package emissary

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// putArgoexecOnPath makes an executable named argoexec with content the only
// thing on PATH.
func putArgoexecOnPath(t *testing.T, content string) {
	t.Helper()
	bin := t.TempDir()
	name := "argoexec"
	if runtime.GOOS == "windows" {
		name += ".exe" // what LookPath looks for there
	}
	require.NoError(t, os.WriteFile(filepath.Join(bin, name), []byte(content), 0o755))
	t.Setenv("PATH", bin)
}

func TestCopyBinary(t *testing.T) {
	dir := t.TempDir()
	putArgoexecOnPath(t, "argoexec")

	require.NoError(t, copyBinary(dir))

	target := filepath.Join(dir, "argoexec")
	got, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Equal(t, "argoexec", string(got))
	info, err := os.Stat(target)
	require.NoError(t, err)
	want := os.FileMode(0o555)
	if runtime.GOOS == "windows" {
		want = 0o444 // Windows has only a read-only attribute
	}
	assert.Equal(t, want, info.Mode().Perm())
}

// A re-run, as when an init container runs again in a recreated pod sandbox,
// replaces the read-only copy from before (which Windows won't rename over
// unless it is made writable), even with a shorter binary, and
// leaves no temporary file behind.
func TestCopyBinary_ReplacesAnEarlierCopy(t *testing.T) {
	dir := t.TempDir()
	putArgoexecOnPath(t, "a longer, older argoexec")
	require.NoError(t, copyBinary(dir))

	putArgoexecOnPath(t, "newer")
	require.NoError(t, copyBinary(dir))

	got, err := os.ReadFile(filepath.Join(dir, "argoexec"))
	require.NoError(t, err)
	assert.Equal(t, "newer", string(got))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "argoexec", entries[0].Name())
}

func TestCopyBinary_NoArgoexecOnPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	require.Error(t, copyBinary(t.TempDir()))
}
