package cmd

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const shutdownContextHelperEnv = "ARGO_TEST_SHUTDOWN_CONTEXT_HELPER"

// TestShutdownContextHelper is not a test: it is the child process for
// TestShutdownContext, so the signal never reaches the test process itself.
func TestShutdownContextHelper(t *testing.T) {
	if os.Getenv(shutdownContextHelperEnv) != "1" {
		t.Skip("helper process for TestShutdownContext")
	}
	ctx, stop := ShutdownContext(context.Background())
	_, _ = os.Stdout.WriteString("ready\n")
	<-ctx.Done()
	stop()
	os.Exit(0)
}

func TestShutdownContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM cannot be sent on Windows")
	}
	for _, sig := range []os.Signal{syscall.SIGTERM, os.Interrupt} {
		t.Run(sig.String(), func(t *testing.T) {
			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestShutdownContextHelper$")
			cmd.Env = append(os.Environ(), shutdownContextHelperEnv+"=1")
			stdout, err := cmd.StdoutPipe()
			require.NoError(t, err)
			require.NoError(t, cmd.Start())

			line, err := bufio.NewReader(stdout).ReadString('\n')
			require.NoError(t, err)
			require.Equal(t, "ready\n", line)
			require.NoError(t, cmd.Process.Signal(sig))

			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			select {
			case err := <-done:
				require.NoError(t, err, "process should exit 0 after %v", sig)
			case <-time.After(10 * time.Second):
				_ = cmd.Process.Kill()
				t.Fatalf("process did not exit within 10s of %v", sig)
			}
		})
	}
}
