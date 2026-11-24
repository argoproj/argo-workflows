//go:build windows

package osspecific

import (
	"errors"
	"io/fs"
	"os"
)

// replaceFile renames src over dst. Windows refuses to replace a read-only
// file, so dst is made writable first.
func replaceFile(src, dst string) error {
	if err := os.Chmod(dst, 0o666); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return os.Rename(src, dst)
}
