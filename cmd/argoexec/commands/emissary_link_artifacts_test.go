package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestLinkInputArtifacts_Normal(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	srcBase := filepath.Join(dir, "inputs")
	require.NoError(t, os.MkdirAll(srcBase, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcBase, "my-artifact"), []byte("payload"), 0o644))
	dstDir := filepath.Join(dir, "container", "tmp")
	dstPath := filepath.Join(dstDir, "my-artifact")

	tmpl := &wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: wfv1.Artifacts{{Name: "my-artifact", Path: dstPath}},
		},
	}

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, tmpl))

	info, err := os.Lstat(dstPath)
	require.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "dst must be a symlink")
	target, err := os.Readlink(dstPath)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(srcBase, "my-artifact"), target)
	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(data))
}

func TestLinkInputArtifacts_MissingSourceIsSkipped(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	srcBase := filepath.Join(dir, "inputs")
	require.NoError(t, os.MkdirAll(srcBase, 0o755))
	// Do NOT write the source file — simulates optional artifact not
	// supplied, or overlap case where supervisor wrote via /mainctrfs.
	dstPath := filepath.Join(dir, "container", "tmp", "maybe")

	tmpl := &wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: wfv1.Artifacts{{Name: "maybe", Path: dstPath}},
		},
	}

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, tmpl))

	_, err := os.Lstat(dstPath)
	assert.True(t, os.IsNotExist(err), "no symlink must be created when source missing")
}

func TestLinkInputArtifacts_OverwritesExistingFile(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	srcBase := filepath.Join(dir, "inputs")
	require.NoError(t, os.MkdirAll(srcBase, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcBase, "data"), []byte("new"), 0o644))

	// Simulate a file already at art.Path — e.g. baked into the user's
	// container image. Legacy SubPath bind mount shadowed it; the symlink
	// approach replaces it.
	dstDir := filepath.Join(dir, "container", "tmp")
	require.NoError(t, os.MkdirAll(dstDir, 0o755))
	dstPath := filepath.Join(dstDir, "data")
	require.NoError(t, os.WriteFile(dstPath, []byte("stale-from-image"), 0o644))

	tmpl := &wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: wfv1.Artifacts{{Name: "data", Path: dstPath}},
		},
	}

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, tmpl))

	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "new", string(data), "pre-existing file at dst must be replaced by the symlink target's content")
}

// TestLinkInputArtifacts_OverwritesExistingDirectory exercises the case
// where a stale directory sits at art.Path; RemoveAll handles directories
// where os.Remove would fail.
// TestLinkInputArtifacts_OverwritesExistingDirectory covers a directory artifact
// whose path already holds a directory on the container's ephemeral filesystem
// (e.g. a git/directory artifact at /tmp/git): linking must replace it with the
// symlink rather than failing. Overlapping user volumes never reach this code
// (they're skipped earlier), so this only ever clears ephemeral container paths.
func TestLinkInputArtifacts_OverwritesExistingDirectory(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	srcBase := filepath.Join(dir, "inputs")
	require.NoError(t, os.MkdirAll(srcBase, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcBase, "data"), []byte("new"), 0o644))

	dstDir := filepath.Join(dir, "container", "tmp")
	require.NoError(t, os.MkdirAll(dstDir, 0o755))
	dstPath := filepath.Join(dstDir, "data")
	require.NoError(t, os.MkdirAll(dstPath, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dstPath, "leftover"), []byte("stale"), 0o644))

	tmpl := &wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: wfv1.Artifacts{{Name: "data", Path: dstPath}},
		},
	}

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, tmpl))

	info, err := os.Lstat(dstPath)
	require.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "directory must be replaced by symlink")
	data, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
}

// TestLinkInputArtifacts_EmptyPathSkipped covers art.Path == "".
func TestLinkInputArtifacts_EmptyPathSkipped(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	srcBase := filepath.Join(dir, "inputs")
	require.NoError(t, os.MkdirAll(srcBase, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcBase, "no-path"), []byte("payload"), 0o644))

	tmpl := &wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: wfv1.Artifacts{{Name: "no-path", Path: ""}},
		},
	}

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, tmpl))
}
