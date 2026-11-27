//go:build !windows

package commands

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func startSignalTestProcess(t *testing.T) (*exec.Cmd, <-chan struct{}) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	require.NoError(t, cmd.Start())
	done := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(done)
	}()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		<-done
	})
	return cmd, done
}

func requireTerminated(t *testing.T, cmd *exec.Cmd, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
		require.Equal(t, syscall.SIGTERM, cmd.ProcessState.Sys().(syscall.WaitStatus).Signal())
	case <-time.After(5 * time.Second):
		t.Fatal("child did not receive SIGTERM")
	}
}

func TestForwardSignals_ParentCancellation(t *testing.T) {
	cmd, done := startSignalTestProcess(t)
	// Keep process I/O outside synctest: only the forwarder's channel operations
	// need to be synchronized to reproduce cancellation before signal delivery.
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
		defer cancel()
		signals := make(chan os.Signal, 1)
		stop := forwardSignals(ctx, signals, cmd.Process.Pid, false)
		defer stop()

		cancel()
		synctest.Wait()
		// signal.NotifyContext may cancel the parent before signal.Notify
		// delivers the same SIGTERM to this channel.
		signals <- syscall.SIGTERM
		synctest.Wait()
		require.Empty(t, signals)
	})
	if t.Failed() {
		return
	}
	requireTerminated(t, cmd, done)
}

func TestForwardSignals_StopBeforeNextAttempt(t *testing.T) {
	first, firstDone := startSignalTestProcess(t)
	second, secondDone := startSignalTestProcess(t)
	synctest.Test(t, func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		signals := make(chan os.Signal, 1)
		stopFirst := forwardSignals(ctx, signals, first.Process.Pid, false)
		synctest.Wait()
		stopFirst()

		stopSecond := forwardSignals(ctx, signals, second.Process.Pid, false)
		defer stopSecond()
		signals <- syscall.SIGTERM
		synctest.Wait()
		require.Empty(t, signals)
	})
	if t.Failed() {
		return
	}
	requireTerminated(t, second, secondDone)
	select {
	case <-firstDone:
		t.Fatal("stopped forwarder signalled the previous attempt")
	default:
	}
}
