//go:build !windows

package osspecific

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// Acquire takes an exclusive, non-blocking file lock. The lock is released
// when the returned file is closed or the process exits.
func Acquire(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock file %q: %w", path, err)
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("acquire exclusive lock on %q: %w", path, err)
	}
	return f, nil
}

// WaitForSharedLock blocks until the exclusive holder of path releases
// (including by process death). On ctx cancel it returns ctx.Err() promptly;
// close(fd) does not interrupt a blocked flock(2) on Linux, so the inner
// goroutine is left to exit when the holder eventually releases.
func WaitForSharedLock(ctx context.Context, path string) error {
	f, err := os.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open lock file %q: %w", path, err)
	}

	// The goroutine below owns f. Closing it from the cancel path would race
	// with the Fd() call there, and worse, free an fd number that another
	// open could reuse before the blocked flock(2) returns.
	done := make(chan error, 1)
	go func() {
		defer func() { _ = f.Close() }()
		fd := int(f.Fd())
		err := unix.Flock(fd, unix.LOCK_SH)
		if err == nil {
			_ = unix.Flock(fd, unix.LOCK_UN)
		}
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("wait for shared lock on %q: %w", path, err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
