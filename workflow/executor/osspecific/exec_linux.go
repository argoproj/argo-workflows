package osspecific

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// IsExecutable reports whether path is a regular file this process may
// execute, as exec.LookPath decides it: by the process's effective IDs, and
// by the file's mode bits only where the kernel can't say (ENOSYS, or EPERM
// from a seccomp filter).
func IsExecutable(path string) bool {
	fi, err := os.Stat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return false
	}
	err = unix.Faccessat(unix.AT_FDCWD, path, unix.X_OK, unix.AT_EACCESS)
	if errors.Is(err, unix.ENOSYS) || errors.Is(err, unix.EPERM) {
		return fi.Mode().Perm()&0o111 != 0
	}
	return err == nil
}
