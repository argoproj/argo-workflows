package osspecific

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, WriteFileAtomic(dir, "f", strings.NewReader("data"), 0o644))

	got, err := os.ReadFile(filepath.Join(dir, "f"))
	require.NoError(t, err)
	assert.Equal(t, "data", string(got))
	info, err := os.Stat(filepath.Join(dir, "f"))
	require.NoError(t, err)
	want := os.FileMode(0o644)
	if runtime.GOOS == "windows" {
		want = 0o666 // Windows has only a read-only attribute
	}
	assert.Equal(t, want, info.Mode().Perm())
}

// A read-only file is replaced too, which Windows won't rename over unless
// it is made writable, even with shorter content, and no temporary file is
// left behind.
func TestWriteFileAtomic_ReplacesAReadOnlyFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, WriteFileAtomic(dir, "f", strings.NewReader("longer, older"), 0o444))
	require.NoError(t, WriteFileAtomic(dir, "f", strings.NewReader("newer"), 0o444))

	got, err := os.ReadFile(filepath.Join(dir, "f"))
	require.NoError(t, err)
	assert.Equal(t, "newer", string(got))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "f", entries[0].Name())
}

func TestWriteFileAtomic_NoDirectory(t *testing.T) {
	require.Error(t, WriteFileAtomic(filepath.Join(t.TempDir(), "missing"), "f", strings.NewReader(""), 0o644))
}
