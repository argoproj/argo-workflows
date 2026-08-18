package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

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

// TestLinkInputArtifacts_OverwritesExistingFile covers the legacy-parity case: a
// file baked into the user's image at art.Path, on the container's own ephemeral
// filesystem (no declared volume), must be replaced by the symlink — matching
// what the legacy SubPath bind mount shadowed.
func TestLinkInputArtifacts_OverwritesExistingFile(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	srcBase := filepath.Join(dir, "inputs")
	require.NoError(t, os.MkdirAll(srcBase, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcBase, "data"), []byte("new"), 0o644))

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

// TestLinkInputArtifacts_OverwritesExistingDirectory covers a directory artifact
// whose path already holds a directory on the container's ephemeral filesystem
// (e.g. a git/directory artifact at /tmp/git): linking must replace it with the
// symlink rather than failing. No declared volume → safe to clear.
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

// TestLinkInputArtifacts_RefusesOverwriteResolvingIntoVolume is the reported-bug
// guard: art.Path is /data/sub where the image ships /data as a symlink into a
// user-declared volume (/data -> volume). Overwriting would os.RemoveAll through
// the symlink and destroy live volume data, so staging must refuse and leave the
// volume untouched.
func TestLinkInputArtifacts_RefusesOverwriteResolvingIntoVolume(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	srcBase := filepath.Join(dir, "inputs")
	require.NoError(t, os.MkdirAll(srcBase, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcBase, "data"), []byte("new"), 0o644))

	// The "volume" is a real directory holding irreplaceable data, declared as a
	// volume mount on the template.
	volumeDir := filepath.Join(dir, "volume")
	require.NoError(t, os.MkdirAll(filepath.Join(volumeDir, "sub"), 0o755))
	liveFile := filepath.Join(volumeDir, "sub", "important.txt")
	require.NoError(t, os.WriteFile(liveFile, []byte("precious"), 0o644))
	realVolumeDir, err := filepath.EvalSymlinks(volumeDir)
	require.NoError(t, err)

	// The image ships /data as a symlink into the volume, so art.Path /data/sub
	// resolves to volume/sub.
	rootfs := filepath.Join(dir, "container")
	require.NoError(t, os.MkdirAll(rootfs, 0o755))
	dataLink := filepath.Join(rootfs, "data")
	require.NoError(t, os.Symlink(volumeDir, dataLink))
	artPath := filepath.Join(dataLink, "sub")

	tmpl := &wfv1.Template{
		Container: &apiv1.Container{
			VolumeMounts: []apiv1.VolumeMount{{Name: "vol", MountPath: realVolumeDir}},
		},
		Inputs: wfv1.Inputs{
			Artifacts: wfv1.Artifacts{{Name: "data", Path: artPath}},
		},
	}

	err = linkInputArtifactsAt(ctx, srcBase, tmpl)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "volume mount")

	// The volume's contents must be untouched.
	data, readErr := os.ReadFile(liveFile)
	require.NoError(t, readErr)
	assert.Equal(t, "precious", string(data), "RemoveAll must not have crossed the symlink into the volume")
}

// TestLinkInputArtifacts_CreateResolvingIntoVolumeAllowed documents the grown-up
// create behavior: when art.Path resolves into a user volume but nothing exists
// there yet, staging creates the symlink anyway (the user asked for it). Creating
// can never destroy data, so it is not gated.
func TestLinkInputArtifacts_CreateResolvingIntoVolumeAllowed(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	srcBase := filepath.Join(dir, "inputs")
	require.NoError(t, os.MkdirAll(srcBase, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcBase, "data"), []byte("payload"), 0o644))

	volumeDir := filepath.Join(dir, "volume")
	require.NoError(t, os.MkdirAll(volumeDir, 0o755))
	realVolumeDir, err := filepath.EvalSymlinks(volumeDir)
	require.NoError(t, err)

	rootfs := filepath.Join(dir, "container")
	require.NoError(t, os.MkdirAll(rootfs, 0o755))
	dataLink := filepath.Join(rootfs, "data")
	require.NoError(t, os.Symlink(volumeDir, dataLink))
	// art.Path resolves to volume/newfile, which does not exist yet.
	artPath := filepath.Join(dataLink, "newfile")

	tmpl := &wfv1.Template{
		Container: &apiv1.Container{
			VolumeMounts: []apiv1.VolumeMount{{Name: "vol", MountPath: realVolumeDir}},
		},
		Inputs: wfv1.Inputs{
			Artifacts: wfv1.Artifacts{{Name: "data", Path: artPath}},
		},
	}

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, tmpl))

	target, err := os.Readlink(artPath)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(srcBase, "data"), target)
	data, err := os.ReadFile(artPath)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(data))
}

// TestLinkInputArtifacts_ReplacesImageSymlinkInRootfs covers an image symlink
// sitting *at* art.Path within the container's own rootfs (no declared volume).
// os.RemoveAll removes the symlink itself (it does not follow the final element),
// so staging safely replaces it and the symlink's old target is untouched.
func TestLinkInputArtifacts_ReplacesImageSymlinkInRootfs(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	srcBase := filepath.Join(dir, "inputs")
	require.NoError(t, os.MkdirAll(srcBase, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcBase, "data"), []byte("new"), 0o644))

	dstDir := filepath.Join(dir, "container", "tmp")
	require.NoError(t, os.MkdirAll(dstDir, 0o755))
	// The image ships art.Path as a symlink to some other rootfs file.
	oldTarget := filepath.Join(dstDir, "image-default")
	require.NoError(t, os.WriteFile(oldTarget, []byte("image-default"), 0o644))
	dstPath := filepath.Join(dstDir, "data")
	require.NoError(t, os.Symlink(oldTarget, dstPath))

	tmpl := &wfv1.Template{
		Inputs: wfv1.Inputs{
			Artifacts: wfv1.Artifacts{{Name: "data", Path: dstPath}},
		},
	}

	require.NoError(t, linkInputArtifactsAt(ctx, srcBase, tmpl))

	target, err := os.Readlink(dstPath)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(srcBase, "data"), target, "image symlink must be replaced by the artifact symlink")
	// RemoveAll removed the symlink, not its old target.
	old, err := os.ReadFile(oldTarget)
	require.NoError(t, err)
	assert.Equal(t, "image-default", string(old), "the image symlink's old target must be untouched")
}
