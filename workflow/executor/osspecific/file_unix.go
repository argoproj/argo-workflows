//go:build !windows

package osspecific

import "os"

// replaceFile renames src over dst.
func replaceFile(src, dst string) error {
	return os.Rename(src, dst)
}
