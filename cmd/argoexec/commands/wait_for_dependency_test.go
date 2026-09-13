package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

const depName = "x"

// setupDep creates tmp/ctr/depName, acquires its lock, and writes the ready
// marker. The caller closes the returned file to simulate the dep exiting.
func setupDep(t *testing.T, tmp string) *os.File {
	t.Helper()
	depDir := filepath.Join(tmp, "ctr", depName)
	require.NoError(t, os.MkdirAll(depDir, 0o777))
	f, err := osspecific.Acquire(filepath.Join(depDir, "lock"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(depDir, "ready"), nil, 0o644))
	return f
}

func useTempVarRunArgo(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	orig := varRunArgo
	varRunArgo = tmp
	t.Cleanup(func() { varRunArgo = orig })
	return tmp
}

func TestWaitForDependency_Success(t *testing.T) {
	tmp := useTempVarRunArgo(t)
	lock := setupDep(t, tmp)

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = os.WriteFile(filepath.Join(tmp, "ctr", depName, "exitcode"), []byte("0"), 0o644)
		_ = lock.Close()
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	require.NoError(t, waitForDependency(ctx, depName))
}

func TestWaitForDependency_DepFailed(t *testing.T) {
	tmp := useTempVarRunArgo(t)
	lock := setupDep(t, tmp)

	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = os.WriteFile(filepath.Join(tmp, "ctr", depName, "exitcode"), []byte("1"), 0o644)
		_ = lock.Close()
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	err := waitForDependency(ctx, depName)
	require.ErrorContains(t, err, "exited with non-zero code: 1")
}

func TestWaitForDependency_DepDiedWithoutExitCode(t *testing.T) {
	tmp := useTempVarRunArgo(t)
	lock := setupDep(t, tmp)

	go func() {
		time.Sleep(100 * time.Millisecond)
		// Release without writing exitcode: simulates a SIGKILLed dep.
		_ = lock.Close()
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	err := waitForDependency(ctx, depName)
	require.ErrorContains(t, err, "died without reporting exit code")
}

func TestWaitForDependency_ContextCancel(t *testing.T) {
	tmp := useTempVarRunArgo(t)
	lock := setupDep(t, tmp)
	t.Cleanup(func() { _ = lock.Close() })

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- waitForDependency(ctx, depName) }()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("waitForDependency did not return within 2s of ctx cancel")
	}
}

// TestWaitForDependency_LockOutlivesExitCode pins the ordering the liveness
// lock relies on: releasing it is the signal that the exit code is readable,
// so runEmissary must not drop the lock until after the exitcode file is
// written. A dependent woken early sees no exitcode and wrongly reports the
// container as having died without reporting one.
func TestWaitForDependency_LockOutlivesExitCode(t *testing.T) {
	tmp := useTempVarRunArgo(t)
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "template"), []byte(`{}`), 0o600))

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	// Watches "main" exactly as a sibling declaring a dependency on it would.
	done := make(chan error, 1)
	go func() { done <- waitForDependency(ctx, "main") }()

	require.NoError(t, run("exit 0"))

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("waitForDependency did not return after the container exited")
	}
}
