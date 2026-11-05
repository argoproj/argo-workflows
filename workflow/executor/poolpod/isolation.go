package poolpod

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// ErrPathExists is returned for an input or declared output path that
// already exists outside a volume: main runs every task in the same
// container, so it can neither replace an image's file nor tell a stale
// output from a fresh one. The task fails and the file is left intact.
var ErrPathExists = errors.New("already exists in the container image or was left by an earlier task; a pooled task's input and output paths must not exist outside a volume")

// Isolation is main's record of what one task created outside volumes, so
// that Cleanup can remove exactly that once the task is over: input links
// and the parent directories made for them, and declared output paths and
// the parent directories they needed. Anything else the task writes outside
// a volume stays (see removeIfEmpty). Volume contents are never touched:
// they are PVC, hostPath or emptyDir contents, or the worker's, main's own
// per-task files among them.
//
// Use one Isolation per task. Reserve each declared output path before the
// task starts, link inputs with Symlink, and call Cleanup after the task's
// processes are gone (Sweep).
type Isolation struct {
	volumes []string
	// paths are removed whole by Cleanup.
	paths []string
	// dirs didn't exist when a path beneath them was reserved; Cleanup
	// removes those left empty.
	dirs []string
}

// NewIsolation returns an Isolation for one task, given the mount paths of
// the volumes whose contents it must never touch: the template's volume
// mounts and the volumes main shares with the worker.
func NewIsolation(volumes []string) *Isolation {
	cleaned := make([]string, len(volumes))
	for i, v := range volumes {
		cleaned[i] = filepath.Clean(v)
	}
	return &Isolation{volumes: cleaned}
}

// Reserve claims path for the task. Outside a volume, it fails with
// ErrPathExists if anything is already at path, and otherwise records path
// for Cleanup to remove. On a volume, it allows anything and records
// nothing. path must be absolute, and must not hold a volume mount.
func (iso *Isolation) Reserve(path string) error {
	_, err := iso.reserve(path)
	return err
}

// Symlink links an input artifact into place: it reserves path, creates its
// parent directories and makes path a symlink to target, the artifact the
// worker staged. A path on a volume is refused: the worker stages such
// inputs on the volume itself.
func (iso *Isolation) Symlink(target, path string) error {
	onVolume, err := iso.reserve(path)
	if err != nil {
		return err
	}
	if onVolume {
		return fmt.Errorf("refusing to link input %s: it is on a volume, where the worker stages inputs in place", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create parent directory of input %s: %w", path, err)
	}
	if err := os.Symlink(target, path); err != nil {
		return fmt.Errorf("link input %s to %s: %w", path, target, err)
	}
	return nil
}

func (iso *Isolation) reserve(path string) (bool, error) {
	if !filepath.IsAbs(path) {
		return false, fmt.Errorf("path %q is not absolute", path)
	}
	path = filepath.Clean(path)
	resolved, missing, err := resolve(path)
	if err != nil {
		return false, err
	}
	if iso.onVolume(path) || iso.onVolume(resolved) {
		return true, nil
	}
	if err := iso.holdsVolume(path, resolved); err != nil {
		return false, err
	}
	if _, err := os.Lstat(path); err == nil {
		return false, fmt.Errorf("%s %w", path, ErrPathExists)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	iso.paths = append(iso.paths, path)
	iso.dirs = append(iso.dirs, missing...)
	return false, nil
}

// Cleanup removes what was reserved, then the directories made for it that
// are left empty, and forgets them. A reserved path the task moved onto a
// volume, by replacing a parent with a symlink, is left alone.
func (iso *Isolation) Cleanup() error {
	var errs []error
	for _, path := range slices.Backward(iso.paths) {
		errs = append(errs, iso.remove(path))
	}
	// Deepest first, so a directory is emptied before its parent is tried.
	dirs := slices.Compact(slices.SortedFunc(slices.Values(iso.dirs), func(a, b string) int {
		return strings.Compare(b, a)
	}))
	for _, dir := range dirs {
		errs = append(errs, removeIfEmpty(dir))
	}
	iso.paths, iso.dirs = nil, nil
	return errors.Join(errs...)
}

func (iso *Isolation) remove(path string) error {
	resolved, _, err := resolve(path)
	if err != nil {
		return err
	}
	if iso.onVolume(resolved) {
		return nil
	}
	if err := iso.holdsVolume(path, resolved); err != nil {
		return err
	}
	return os.RemoveAll(path)
}

// removeIfEmpty removes dir if it is an empty directory. Anything the task
// left in it stays, as files written outside volumes persist within a
// worker pod, and so does dir.
func removeIfEmpty(dir string) error {
	err := os.Remove(dir)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if entries, readErr := os.ReadDir(dir); readErr == nil && len(entries) > 0 {
		return nil
	}
	return err
}

// onVolume reports whether path is a volume's mount point or inside one.
func (iso *Isolation) onVolume(path string) bool {
	return slices.ContainsFunc(iso.volumes, func(v string) bool {
		_, ok := common.RelWithin(v, path)
		return ok
	})
}

// holdsVolume refuses a path with a volume mounted beneath it: removing it
// would recurse into the volume.
func (iso *Isolation) holdsVolume(paths ...string) error {
	for _, path := range paths {
		for _, v := range iso.volumes {
			if _, ok := common.RelWithin(path, v); ok {
				return fmt.Errorf("refusing %s: volume %s is mounted beneath it", path, v)
			}
		}
	}
	return nil
}

// resolve returns path with the symlinks in its existing ancestors
// resolved, and the ancestors that don't exist yet, shallowest first. path
// itself is not resolved, so a symlink at path is not followed.
func resolve(path string) (string, []string, error) {
	var missing []string
	dir := filepath.Dir(path)
	for {
		if _, err := os.Lstat(dir); err == nil {
			break
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", nil, err
		}
		missing = append(missing, dir)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	slices.Reverse(missing)
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", nil, err
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return "", nil, err
	}
	return filepath.Join(resolvedDir, rel), missing, nil
}
