package file

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWaitForCreate_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	require.NoError(t, WaitForCreate(ctx, path))
}

func TestWaitForCreate_AppearsLater(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "appears")

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = os.WriteFile(path, []byte("hi"), 0o644)
	}()

	require.NoError(t, WaitForCreate(ctx, path))
}

func TestWaitForCreate_ContextCancelled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "never")

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	err := WaitForCreate(ctx, path)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestWaitForCreate_ParentMissingIsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing-parent", "file")

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()

	err := WaitForCreate(ctx, path)
	require.Error(t, err)
}

func TestWatchFile_FiresOnCreateAndWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var fires int32
	done := make(chan struct{})
	go func() {
		_ = WatchFile(ctx, path, func() {
			atomic.AddInt32(&fires, 1)
		})
		close(done)
	}()

	// Give the watcher time to install before writing.
	time.Sleep(100 * time.Millisecond)

	require.NoError(t, os.WriteFile(path, []byte("a"), 0o644))
	require.NoError(t, os.WriteFile(path, []byte("ab"), 0o644))

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&fires) >= 2
	}, 2*time.Second, 10*time.Millisecond)

	cancel()
	<-done
}

func TestWatchFile_FiresWhenAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(path, []byte("hi"), 0o644))

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var fires int32
	done := make(chan struct{})
	go func() {
		_ = WatchFile(ctx, path, func() {
			atomic.AddInt32(&fires, 1)
		})
		close(done)
	}()

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&fires) >= 1
	}, 2*time.Second, 10*time.Millisecond)

	cancel()
	<-done
}
