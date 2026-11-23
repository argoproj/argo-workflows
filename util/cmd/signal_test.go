package cmd

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	shutdownContextHelperEnv = "ARGO_TEST_SHUTDOWN_CONTEXT_HELPER"
	// helperExits exits 0 once the context is done.
	helperExits = "exits"
	// helperHangs never finishes shutting down, so only a second signal stops it.
	helperHangs = "hangs"
)

// TestShutdownContextHelper is not a test: it is the child process for
// TestShutdownContext, so the signal never reaches the test process itself.
func TestShutdownContextHelper(t *testing.T) {
	mode := os.Getenv(shutdownContextHelperEnv)
	if mode == "" {
		t.Skip("helper process for TestShutdownContext")
	}
	ctx, stop := ShutdownContext(context.Background())
	_, _ = os.Stdout.WriteString("ready\n")
	<-ctx.Done()
	if mode == helperHangs {
		_, _ = os.Stdout.WriteString("shutting down\n")
		time.Sleep(time.Minute)
	}
	stop()
	os.Exit(0)
}

// startHelper starts the helper in mode and waits until it has installed the signal handler.
func startHelper(t *testing.T, mode string) (*exec.Cmd, *bufio.Reader, <-chan error) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestShutdownContextHelper$")
	cmd.Env = append(os.Environ(), shutdownContextHelperEnv+"="+mode)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	out := bufio.NewReader(stdout)
	line, err := out.ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "ready\n", line)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	return cmd, out, done
}

func TestShutdownContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM cannot be sent on Windows")
	}
	for _, sig := range []os.Signal{syscall.SIGTERM, os.Interrupt} {
		t.Run(sig.String(), func(t *testing.T) {
			cmd, _, done := startHelper(t, helperExits)
			require.NoError(t, cmd.Process.Signal(sig))

			select {
			case err := <-done:
				require.NoError(t, err, "process should exit 0 after %v", sig)
			case <-time.After(10 * time.Second):
				t.Fatalf("process did not exit within 10s of %v", sig)
			}
		})
	}
}

func TestShutdownContextSecondSignalKills(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM cannot be sent on Windows")
	}
	for _, sig := range []os.Signal{syscall.SIGTERM, os.Interrupt} {
		t.Run(sig.String(), func(t *testing.T) {
			cmd, out, done := startHelper(t, helperHangs)
			require.NoError(t, cmd.Process.Signal(sig))
			line, err := out.ReadString('\n')
			require.NoError(t, err)
			require.Equal(t, "shutting down\n", line)

			// The default behaviour is restored asynchronously once the context is done,
			// so keep signalling until the process dies rather than racing that.
			ticker := time.NewTicker(50 * time.Millisecond)
			defer ticker.Stop()
			timeout := time.After(10 * time.Second)
			for {
				select {
				case err := <-done:
					var exitErr *exec.ExitError
					require.ErrorAs(t, err, &exitErr, "process should be killed by %v", sig)
					status, ok := exitErr.Sys().(syscall.WaitStatus)
					require.True(t, ok)
					assert.True(t, status.Signaled(), "process should be killed by a signal, got %v", err)
					assert.Equal(t, sig, status.Signal())
					return
				case <-ticker.C:
					if err := cmd.Process.Signal(sig); !errors.Is(err, os.ErrProcessDone) {
						require.NoError(t, err)
					}
				case <-timeout:
					t.Fatalf("second %v did not kill the process within 10s", sig)
				}
			}
		})
	}
}
