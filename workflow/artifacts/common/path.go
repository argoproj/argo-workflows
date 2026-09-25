package common

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// LocalPathForObject returns the path under dest at which an object stored under keyPrefix
// should be written, or an error if the key would place the file outside dest.
//
// Object keys come from the storage backend and are untrusted: a key containing ".." segments,
// or one that lands under a symlink already present inside dest, must not be able to write
// outside dest. This mirrors the containment checks applied to archive entries by untar.
// Object keys always use "/" as their separator.
func LocalPathForObject(dest, keyPrefix, objKey string) (string, error) {
	rel := strings.TrimPrefix(strings.TrimPrefix(objKey, keyPrefix), "/")
	cleanDest := filepath.Clean(dest)
	localPath := filepath.Join(cleanDest, filepath.FromSlash(rel))
	if localPath != cleanDest && !strings.HasPrefix(localPath, cleanDest+string(os.PathSeparator)) {
		return "", fmt.Errorf("illegal object key %q: resolves outside %q", objKey, dest)
	}
	if err := checkNoSymlinkEscape(cleanDest, localPath); err != nil {
		return "", fmt.Errorf("illegal object key %q: %w", objKey, err)
	}
	return localPath, nil
}

// checkNoSymlinkEscape verifies that localPath is not itself a symlink and that its nearest
// existing ancestor does not resolve outside dest. dest need not exist yet.
func checkNoSymlinkEscape(dest, localPath string) error {
	if localPath == dest {
		return nil
	}
	resolvedDest, err := filepath.EvalSymlinks(dest)
	if errors.Is(err, fs.ErrNotExist) {
		// nothing exists under dest yet, so nothing can redirect the write
		return nil
	}
	if err != nil {
		return err
	}
	if fi, lstatErr := os.Lstat(localPath); lstatErr == nil && fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%q is a symlink", localPath)
	}
	ancestor := filepath.Dir(localPath)
	for {
		_, lstatErr := os.Lstat(ancestor)
		if lstatErr == nil {
			break
		}
		if !errors.Is(lstatErr, fs.ErrNotExist) {
			return lstatErr
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			break
		}
		ancestor = parent
	}
	resolvedAncestor, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return err
	}
	if resolvedAncestor != resolvedDest && !strings.HasPrefix(resolvedAncestor, resolvedDest+string(os.PathSeparator)) {
		return fmt.Errorf("%q resolves outside %q through a symlink", ancestor, dest)
	}
	return nil
}
