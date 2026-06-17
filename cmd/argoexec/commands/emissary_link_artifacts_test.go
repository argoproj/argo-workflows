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

// emptyMountinfo writes a mountinfo file with no entries and returns its path,
// so linkInputArtifactsAt's mount-boundary guard finds no mounts and proceeds
// as it would on a path with nothing mounted beneath it.
func emptyMountinfo(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "mountinfo")
	require.NoError(t, os.WriteFile(p, nil, 0o644))
	return p
}

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

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, emptyMountinfo(t, dir), tmpl))

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

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, emptyMountinfo(t, dir), tmpl))

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

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, emptyMountinfo(t, dir), tmpl))

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

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, emptyMountinfo(t, dir), tmpl))

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

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, emptyMountinfo(t, dir), tmpl))
}

// TestLinkInputArtifacts_RefusesToClearAcrossMount is the runtime guard against
// the data-loss case where art.Path is an ancestor of a mounted volume (the
// controller rejects this up front, but the emissary must not destroy a live
// volume even if it ever reaches this code). With a mount nested under art.Path,
// staging must fail rather than os.RemoveAll into the mount.
func TestLinkInputArtifacts_RefusesToClearAcrossMount(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	srcBase := filepath.Join(dir, "inputs")
	require.NoError(t, os.MkdirAll(srcBase, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcBase, "data"), []byte("new"), 0o644))

	// art.Path is a directory with a "mounted volume" nested under it.
	dataDir := filepath.Join(dir, "container", "data")
	sharedDir := filepath.Join(dataDir, "shared")
	require.NoError(t, os.MkdirAll(sharedDir, 0o755))
	liveFile := filepath.Join(sharedDir, "important.txt")
	require.NoError(t, os.WriteFile(liveFile, []byte("precious"), 0o644))

	// mountinfo declaring sharedDir as a mount point nested under dataDir.
	miPath := filepath.Join(dir, "mountinfo")
	require.NoError(t, os.WriteFile(miPath, []byte("36 35 0:42 / "+sharedDir+" rw,relatime - tmpfs tmpfs rw\n"), 0o644))

	tmpl := &wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: wfv1.Artifacts{{Name: "data", Path: dataDir}},
		},
	}

	err := linkInputArtifactsAt(ctx, srcBase, miPath, tmpl)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mount point")

	// The live volume's contents must be untouched.
	data, readErr := os.ReadFile(liveFile)
	require.NoError(t, readErr)
	assert.Equal(t, "precious", string(data), "RemoveAll must not have crossed the mount boundary")
	// And no symlink should have replaced art.Path.
	info, statErr := os.Lstat(dataDir)
	require.NoError(t, statErr)
	assert.True(t, info.IsDir(), "art.Path must remain the original directory, not a symlink")
}

// TestMountPointAtOrUnder unit-tests the mount-table scan directly.
func TestMountPointAtOrUnder(t *testing.T) {
	dir := t.TempDir()
	miPath := filepath.Join(dir, "mountinfo")
	content := "" +
		"1 0 8:1 / / rw - ext4 /dev/sda1 rw\n" +
		"42 1 0:50 / /data/shared rw - nfs server:/x rw\n" +
		"43 1 0:51 / /weird\\040dir/vol rw - tmpfs tmpfs rw\n"
	require.NoError(t, os.WriteFile(miPath, []byte(content), 0o644))

	// Mount nested under /data is found.
	mp, err := mountPointAtOrUnder(miPath, "/data")
	require.NoError(t, err)
	assert.Equal(t, "/data/shared", mp)

	// Exact mount point matches.
	mp, err = mountPointAtOrUnder(miPath, "/data/shared")
	require.NoError(t, err)
	assert.Equal(t, "/data/shared", mp)

	// Unrelated path finds nothing.
	mp, err = mountPointAtOrUnder(miPath, "/tmp/work")
	require.NoError(t, err)
	assert.Empty(t, mp)

	// A path that is an ancestor of an escaped-space mount is detected.
	mp, err = mountPointAtOrUnder(miPath, "/weird dir")
	require.NoError(t, err)
	assert.Equal(t, "/weird dir/vol", mp)

	// Unreadable mountinfo surfaces an error (caller fails closed).
	_, err = mountPointAtOrUnder(filepath.Join(dir, "nope"), "/data")
	require.Error(t, err)
}
