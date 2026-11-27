package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalPathForObject(t *testing.T) {
	dest := t.TempDir()
	tests := []struct {
		name      string
		keyPrefix string
		objKey    string
		want      string
	}{
		{"nested key", "dir/", "dir/a/b.txt", filepath.Join(dest, "a", "b.txt")},
		{"prefix without trailing slash", "dir", "dir/a", filepath.Join(dest, "a")},
		{"empty prefix", "", "a/b", filepath.Join(dest, "a", "b")},
		{"dot-dot that stays inside", "dir/", "dir/a/../b", filepath.Join(dest, "b")},
		{"key equal to prefix", "dir/", "dir/", dest},
		{"doubled slash", "dir/", "dir//etc/x", filepath.Join(dest, "etc", "x")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LocalPathForObject(dest, tt.keyPrefix, tt.objKey)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("dot-dot escape", func(t *testing.T) {
		_, err := LocalPathForObject(dest, "dir/", "dir/../../etc/passwd")
		require.ErrorContains(t, err, "resolves outside")
	})
	t.Run("dot-dot escape with unnormalized prefix", func(t *testing.T) {
		_, err := LocalPathForObject(dest, "dir", "dir/../evil")
		require.ErrorContains(t, err, "resolves outside")
	})
	t.Run("dest does not exist yet", func(t *testing.T) {
		missing := filepath.Join(dest, "not-yet")
		got, err := LocalPathForObject(missing, "dir/", "dir/a")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(missing, "a"), got)
	})
	t.Run("symlink ancestor escaping dest", func(t *testing.T) {
		outside := t.TempDir()
		require.NoError(t, os.Symlink(outside, filepath.Join(dest, "pivot")))
		_, err := LocalPathForObject(dest, "dir/", "dir/pivot/deeper/x")
		require.ErrorContains(t, err, "through a symlink")
	})
	t.Run("symlink ancestor inside dest", func(t *testing.T) {
		require.NoError(t, os.Mkdir(filepath.Join(dest, "real"), 0o700))
		require.NoError(t, os.Symlink(filepath.Join(dest, "real"), filepath.Join(dest, "inner")))
		got, err := LocalPathForObject(dest, "dir/", "dir/inner/x")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dest, "inner", "x"), got)
	})
	t.Run("symlink at final path", func(t *testing.T) {
		target := filepath.Join(t.TempDir(), "target")
		require.NoError(t, os.WriteFile(target, []byte("x"), 0o600))
		require.NoError(t, os.Symlink(target, filepath.Join(dest, "final")))
		_, err := LocalPathForObject(dest, "dir/", "dir/final")
		require.ErrorContains(t, err, "is a symlink")
	})
}
