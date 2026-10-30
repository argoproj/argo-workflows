package poolpod

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sandbox is a stand-in for main's filesystem: an "image" directory with
// a file in it, and a "volume" directory mounted at vol.
type sandbox struct {
	root      string
	imageFile string
	vol       string
}

func newSandbox(t *testing.T) sandbox {
	t.Helper()
	root := t.TempDir()
	s := sandbox{
		root:      root,
		imageFile: filepath.Join(root, "etc", "config"),
		vol:       filepath.Join(root, "mnt", "vol"),
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(s.imageFile), 0o755))
	require.NoError(t, os.WriteFile(s.imageFile, []byte("image"), 0o644))
	require.NoError(t, os.MkdirAll(s.vol, 0o755))
	return s
}

func (s sandbox) path(elem ...string) string {
	return filepath.Join(append([]string{s.root}, elem...)...)
}

func (s sandbox) isolation() *Isolation {
	return NewIsolation([]string{s.vol})
}

// An input or declared output path that exists in the image fails the task
// and leaves the image file intact.
func TestReserve_RefusesAPathInTheImage(t *testing.T) {
	s := newSandbox(t)
	iso := s.isolation()

	err := iso.Reserve(s.imageFile)
	require.ErrorIs(t, err, ErrPathExists)
	require.NoError(t, iso.Cleanup())
	data, err := os.ReadFile(s.imageFile)
	require.NoError(t, err)
	assert.Equal(t, "image", string(data))

	require.ErrorIs(t, iso.Symlink(s.path("input"), s.imageFile), ErrPathExists)
	require.NoError(t, iso.Cleanup())
	data, err = os.ReadFile(s.imageFile)
	require.NoError(t, err)
	assert.Equal(t, "image", string(data), "an input isn't linked over an image file")
}

func TestReserve_RefusesADirectoryInTheImage(t *testing.T) {
	s := newSandbox(t)
	require.ErrorIs(t, s.isolation().Reserve(filepath.Dir(s.imageFile)), ErrPathExists)
	assert.FileExists(t, s.imageFile)
}

func TestReserve_RefusesASymlinkInTheImage(t *testing.T) {
	s := newSandbox(t)
	link := s.path("link")
	require.NoError(t, os.Symlink(s.path("nowhere"), link))
	require.ErrorIs(t, s.isolation().Reserve(link), ErrPathExists, "even a dangling one")
}

func TestReserve_RefusesARelativePath(t *testing.T) {
	require.Error(t, newSandbox(t).isolation().Reserve("tmp/out"))
}

func TestReserve_AllowsAnExistingPathOnAVolume(t *testing.T) {
	s := newSandbox(t)
	onVol := filepath.Join(s.vol, "data")
	require.NoError(t, os.WriteFile(onVol, []byte("volume"), 0o644))
	iso := s.isolation()

	require.NoError(t, iso.Reserve(onVol))
	require.NoError(t, iso.Reserve(s.vol))
	require.NoError(t, iso.Cleanup())
	assert.FileExists(t, onVol, "volume contents are never touched")
}

func TestReserve_RefusesAPathHoldingAVolume(t *testing.T) {
	s := newSandbox(t)
	iso := s.isolation()
	err := iso.Reserve(s.path("mnt"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "volume")
	require.NoError(t, iso.Cleanup())
	assert.DirExists(t, s.vol)
}

// A path whose parent is an image symlink into a volume is on that volume.
func TestReserve_ResolvesSymlinksIntoAVolume(t *testing.T) {
	s := newSandbox(t)
	require.NoError(t, os.Symlink(s.vol, s.path("data")))
	onVol := filepath.Join(s.vol, "existing")
	require.NoError(t, os.WriteFile(onVol, nil, 0o644))
	iso := s.isolation()

	require.NoError(t, iso.Reserve(s.path("data", "existing")))
	require.NoError(t, iso.Reserve(s.path("data", "new", "out")))
	require.NoError(t, os.MkdirAll(filepath.Join(s.vol, "new"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(s.vol, "new", "out"), nil, 0o644))
	require.NoError(t, iso.Cleanup())
	assert.FileExists(t, onVol)
	assert.FileExists(t, filepath.Join(s.vol, "new", "out"), "outputs written to a volume stay there")
}

// The next task sees none of this one's declared outputs, and the
// directories they needed are gone too.
func TestCleanup_RemovesDeclaredOutputs(t *testing.T) {
	s := newSandbox(t)
	iso := s.isolation()
	file := s.path("tmp", "out", "result.txt")
	dir := s.path("tmp", "out", "dir")
	require.NoError(t, os.MkdirAll(s.path("tmp"), 0o777))
	require.NoError(t, iso.Reserve(file))
	require.NoError(t, iso.Reserve(dir))

	// The task writes its outputs.
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(file, []byte("out"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a"), []byte("a"), 0o644))

	require.NoError(t, iso.Cleanup())
	assert.NoFileExists(t, file)
	assert.NoDirExists(t, dir)
	assert.NoDirExists(t, s.path("tmp", "out"), "a parent that didn't exist before the task is removed once empty")
	assert.DirExists(t, s.path("tmp"), "a parent that existed before the task stays")

	require.NoError(t, s.isolation().Reserve(file), "the next task can declare the same output")
}

func TestCleanup_KeepsWhatTheTaskWroteBesideItsOutputs(t *testing.T) {
	s := newSandbox(t)
	iso := s.isolation()
	out := s.path("work", "out")
	require.NoError(t, iso.Reserve(out))
	require.NoError(t, os.MkdirAll(s.path("work"), 0o755))
	require.NoError(t, os.WriteFile(out, nil, 0o644))
	require.NoError(t, os.WriteFile(s.path("work", "scratch"), nil, 0o644))

	require.NoError(t, iso.Cleanup())
	assert.NoFileExists(t, out)
	// Files written outside volumes, other than declared outputs, persist
	// within a worker pod.
	assert.FileExists(t, s.path("work", "scratch"))
}

func TestCleanup_ToleratesOutputsNeverWritten(t *testing.T) {
	s := newSandbox(t)
	iso := s.isolation()
	require.NoError(t, iso.Reserve(s.path("tmp", "never", "written")))
	require.NoError(t, iso.Cleanup())
}

// The next task sees none of this one's inputs.
func TestSymlink_LinksAnInputAndCleanupRemovesIt(t *testing.T) {
	s := newSandbox(t)
	staged := s.path("staged", "input")
	require.NoError(t, os.MkdirAll(filepath.Dir(staged), 0o755))
	require.NoError(t, os.WriteFile(staged, []byte("in"), 0o644))
	iso := s.isolation()

	dst := s.path("inputs", "a", "b", "input.txt")
	require.NoError(t, iso.Symlink(staged, dst))
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, "in", string(data))

	require.NoError(t, iso.Cleanup())
	_, err = os.Lstat(dst)
	require.ErrorIs(t, err, os.ErrNotExist)
	assert.NoDirExists(t, s.path("inputs"), "the parent directories made for the link are removed")
	assert.FileExists(t, staged, "the staged artifact is the worker's, on its volume")
}

func TestSymlink_RefusesAVolumePath(t *testing.T) {
	s := newSandbox(t)
	iso := s.isolation()
	require.Error(t, iso.Symlink(s.path("staged"), filepath.Join(s.vol, "input")), "the worker stages inputs on volumes in place")
	_, err := os.Lstat(filepath.Join(s.vol, "input"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

// The task can replace a parent of a declared output with a symlink into a
// volume; cleanup must not follow it.
func TestCleanup_DoesNotFollowASymlinkIntoAVolume(t *testing.T) {
	s := newSandbox(t)
	iso := s.isolation()
	out := s.path("work", "out")
	require.NoError(t, iso.Reserve(out))

	require.NoError(t, os.WriteFile(filepath.Join(s.vol, "out"), []byte("volume"), 0o644))
	require.NoError(t, os.Symlink(s.vol, s.path("work")))

	require.NoError(t, iso.Cleanup())
	assert.FileExists(t, filepath.Join(s.vol, "out"))
}

func TestCleanup_DoesNotRemoveAPathHoldingAVolume(t *testing.T) {
	s := newSandbox(t)
	iso := NewIsolation([]string{s.path("out", "vol")})
	require.Error(t, iso.Reserve(s.path("out")), "a volume is mounted beneath it")

	// Cleanup checks again rather than recurse into a volume.
	require.NoError(t, os.MkdirAll(s.path("out", "vol"), 0o755))
	require.NoError(t, os.WriteFile(s.path("out", "vol", "data"), nil, 0o644))
	iso.paths = append(iso.paths, s.path("out"))
	require.Error(t, iso.Cleanup())
	assert.FileExists(t, s.path("out", "vol", "data"))
}

func TestCleanup_IsIdempotent(t *testing.T) {
	s := newSandbox(t)
	iso := s.isolation()
	out := s.path("out")
	require.NoError(t, iso.Reserve(out))
	require.NoError(t, os.WriteFile(out, nil, 0o644))
	require.NoError(t, iso.Cleanup())
	require.NoError(t, os.WriteFile(out, []byte("next task's"), 0o644))
	require.NoError(t, iso.Cleanup())
	assert.FileExists(t, out, "a second Cleanup removes nothing more")
}
