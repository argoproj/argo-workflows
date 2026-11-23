//go:build !linux

package osspecific

import "os"

// IsExecutable reports whether path is a regular file with an execute bit
// set. Only Linux, where loop mode runs, checks this process's access.
func IsExecutable(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0o111 != 0
}
