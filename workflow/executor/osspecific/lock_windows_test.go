//go:build windows

package osspecific

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestMain runs as the lock-holder subprocess when GO_WANT_LOCK_HELPER_PROCESS=1:
// it acquires the lock at GO_LOCK_HELPER_PATH, prints "LOCKED\n", and sleeps.
func TestMain(m *testing.M) {
	if os.Getenv("GO_WANT_LOCK_HELPER_PROCESS") == "1" {
		runLockHelper()
		return
	}
	os.Exit(m.Run())
}

func runLockHelper() {
	path := os.Getenv("GO_LOCK_HELPER_PATH")
	f, err := Acquire(path)
	if err != nil {
		_, _ = os.Stderr.WriteString("acquire failed: " + err.Error() + "\n")
		os.Exit(2)
	}
	_ = f
	_, _ = os.Stdout.WriteString("LOCKED\n")
	time.Sleep(30 * time.Second)
}

func TestAcquire_Succeeds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	f, err := Acquire(path)
	require.NoError(t, err)
	require.NotNil(t, f)
	t.Cleanup(func() { _ = f.Close() })
}

func TestAcquire_FailsIfAlreadyHeldInSameProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	f1, err := Acquire(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f1.Close() })

	_, err = Acquire(path)
	require.Error(t, err)
}

func TestWaitForSharedLock_UnblocksOnClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	f, err := Acquire(path)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		done <- WaitForSharedLock(t.Context(), path)
	}()

	select {
	case <-done:
		t.Fatal("WaitForSharedLock returned before exclusive lock was released")
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, f.Close())

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForSharedLock did not return within 2s of releasing the exclusive lock")
	}
}

func TestWaitForSharedLock_UnblocksOnProcessKill(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "GO_WANT_LOCK_HELPER_PROCESS=1", "GO_LOCK_HELPER_PATH="+path)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	line, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "LOCKED\n", line)

	done := make(chan error, 1)
	go func() {
		done <- WaitForSharedLock(t.Context(), path)
	}()

	select {
	case <-done:
		t.Fatal("WaitForSharedLock returned before subprocess died")
	case <-time.After(100 * time.Millisecond):
	}

	// os.Process.Kill maps to TerminateProcess on Windows.
	require.NoError(t, cmd.Process.Kill())

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForSharedLock did not return within 2s of subprocess TerminateProcess")
	}
}

func TestWaitForSharedLock_Cancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock")
	f, err := Acquire(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- WaitForSharedLock(ctx, path)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForSharedLock did not return within 2s of ctx cancel")
	}
}
