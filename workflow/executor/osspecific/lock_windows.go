//go:build windows

package osspecific

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// Locked range covers the whole file: offset 0, length MAXDWORD:MAXDWORD.
const (
	lockBytesLow  uint32 = 0xffffffff
	lockBytesHigh uint32 = 0xffffffff
)

// Acquire takes an exclusive, non-blocking file lock. The lock is released
// when the returned file is closed or the process exits.
func Acquire(path string) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open lock file %q: %w", path, err)
	}
	h := windows.Handle(f.Fd())
	var ov windows.Overlapped
	if err := windows.LockFileEx(h, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, lockBytesLow, lockBytesHigh, &ov); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("acquire exclusive lock on %q: %w", path, err)
	}
	return f, nil
}

// WaitForSharedLock blocks until the exclusive holder of path releases
// (including by process death, which on Windows includes TerminateProcess).
// On ctx cancel, CancelIoEx aborts the pending lock and ctx.Err() is
// returned.
func WaitForSharedLock(ctx context.Context, path string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return fmt.Errorf("lock file path %q: %w", path, err)
	}
	// The handle must be opened for overlapped I/O: on a synchronous handle
	// (which is what os.OpenFile creates) a contended LockFileEx blocks inside
	// the call instead of returning ERROR_IO_PENDING, so it can never be
	// cancelled. Share mode must allow the exclusive holder's open too.
	h, err := windows.CreateFile(p,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return fmt.Errorf("open lock file %q: %w", path, err)
	}

	// Manual-reset event so overlapped I/O completion can be waited on.
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		_ = windows.CloseHandle(h)
		return fmt.Errorf("create event: %w", err)
	}
	ov := windows.Overlapped{HEvent: event}

	lockErr := windows.LockFileEx(h, 0, 0, lockBytesLow, lockBytesHigh, &ov)

	release := func() {
		_ = windows.UnlockFileEx(h, 0, lockBytesLow, lockBytesHigh, &ov)
		_ = windows.CloseHandle(event)
		_ = windows.CloseHandle(h)
	}

	if lockErr == nil {
		release()
		return nil
	}
	if !errors.Is(lockErr, windows.ERROR_IO_PENDING) {
		_ = windows.CloseHandle(event)
		_ = windows.CloseHandle(h)
		return fmt.Errorf("wait for shared lock on %q: %w", path, lockErr)
	}

	// GetOverlappedResult with wait=true blocks on ov.HEvent and reports the
	// operation's own status, so an aborted lock surfaces as an error rather
	// than as a signalled event.
	done := make(chan error, 1)
	go func() {
		var n uint32
		done <- windows.GetOverlappedResult(h, &ov, &n, true)
	}()

	select {
	case werr := <-done:
		release()
		if werr != nil {
			return fmt.Errorf("wait for shared lock on %q: %w", path, werr)
		}
		return nil
	case <-ctx.Done():
		_ = windows.CancelIoEx(h, &ov)
		<-done
		release()
		return ctx.Err()
	}
}
