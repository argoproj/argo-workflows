//go:build !windows

// These tests need the protocol's lock probes, which are Linux-only, as
// loop mode is (osspecific.LockHeld).

package poolpod

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

// The worker sees main as up exactly while main holds its liveness lock,
// and WaitMainUp returns once it does.
func TestPool_MainUp(t *testing.T) {
	ctx := t.Context()
	p := NewPool(t.TempDir())
	up, err := p.MainUp()
	require.NoError(t, err)
	assert.False(t, up, "main hasn't started")

	waited := make(chan error, 1)
	go func() { waited <- p.WaitMainUp(ctx) }()
	select {
	case waitErr := <-waited:
		t.Fatalf("WaitMainUp returned before main was up: %v", waitErr)
	case <-time.After(3 * mainUpInterval):
	}

	lock, err := p.HoldMain(ctx)
	require.NoError(t, err)
	select {
	case waitErr := <-waited:
		require.NoError(t, waitErr)
	case <-time.After(5 * time.Second):
		t.Fatal("WaitMainUp didn't return once main was up")
	}
	up, err = p.MainUp()
	require.NoError(t, err)
	assert.True(t, up)

	require.NoError(t, lock.Close())
	up, err = p.MainUp()
	require.NoError(t, err)
	assert.False(t, up, "main's lock is released when it stops")

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, p.WaitMainUp(cancelled), context.Canceled)
}

func TestTake_RunsAnOfferedTask(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")

	run, err := task.Take(t.Context())
	require.NoError(t, err)
	assert.Equal(t, Status{State: Running}, status(t, task))
	taken, err := os.ReadFile(filepath.Join(task.Dir(), takenFile))
	require.NoError(t, err)
	assert.Equal(t, "main", string(taken))

	require.NoError(t, run.Finish(3, ""))
	assert.Equal(t, Status{State: Exited, ExitCode: 3}, status(t, task))
	held, err := osspecific.LockHeld(filepath.Join(task.Dir(), LockFile))
	require.NoError(t, err)
	assert.False(t, held, "Finish releases the lock")
}

// A task main fails itself, such as for an input path that exists in the
// image, reaches the worker with main's message as well as its exit code.
func TestFinish_WithAMessage(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")
	run, err := task.Take(t.Context())
	require.NoError(t, err)

	require.NoError(t, run.Finish(1, "input path /etc/hosts exists in the image"))
	assert.Equal(t, Status{State: Exited, ExitCode: 1, Message: "input path /etc/hosts exists in the image"}, status(t, task))
}

// The same once main has the lock and finds `taken` already there: the
// worker removing the task directory between main's create and its read of
// `taken` is a task not offered. A dangling `taken` stands for the removal.
func TestTake_RefusesATaskRemovedAsTakenIsRead(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "t")
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone", takenFile), filepath.Join(task.Dir(), takenFile)); err != nil {
		t.Skipf("needs symlinks: %v", err)
	}

	_, err := task.Take(t.Context())
	require.ErrorIs(t, err, ErrNotOffered)
	held, err := osspecific.LockHeld(filepath.Join(task.Dir(), LockFile))
	require.NoError(t, err)
	assert.False(t, held)
}

// A withdrawn task is skipped and lock released.
func TestTake_SkipsAWithdrawnTask(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")

	withdrawn, err := task.Withdraw()
	require.NoError(t, err)
	require.True(t, withdrawn)
	assert.Equal(t, Status{State: Withdrawn}, status(t, task))

	_, err = task.Take(t.Context())
	require.ErrorIs(t, err, ErrWithdrawn)
	held, err := osspecific.LockHeld(filepath.Join(task.Dir(), LockFile))
	require.NoError(t, err)
	assert.False(t, held)
	assert.Equal(t, Status{State: Withdrawn}, status(t, task))
}

func TestWithdraw_FailsOnceMainHasTheTask(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")
	run, err := task.Take(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = run.Finish(0, "") })

	withdrawn, err := task.Withdraw()
	require.NoError(t, err)
	assert.False(t, withdrawn)
	assert.Equal(t, Status{State: Running}, status(t, task))
}

// A task main hasn't taken is withdrawn; one it has is asked to stop.
func TestWithdrawOrStop(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")
	withdrew, _, err := task.WithdrawOrStop(StopLost)
	require.NoError(t, err)
	assert.True(t, withdrew)

	task = offered(t, p, "b")
	run, err := task.Take(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = run.Finish(0, "") })
	withdrew, stood, err := task.WithdrawOrStop(StopLost)
	require.NoError(t, err)
	assert.False(t, withdrew)
	assert.Equal(t, StopLost, stood)
}

// A restarted main doesn't re-run a taken task.
func TestTake_SkipsATaskAlreadyTaken(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")
	run, err := task.Take(t.Context())
	require.NoError(t, err)
	require.NoError(t, run.Finish(0, ""))

	_, err = task.Take(t.Context())
	require.ErrorIs(t, err, ErrTaken)
	assert.Equal(t, Status{State: Exited}, status(t, task))
}

func TestStatus_MainDied(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")
	run, err := task.Take(t.Context())
	require.NoError(t, err)

	// Releasing the lock without writing exitcode is what the kernel does
	// when main dies.
	require.NoError(t, run.lock.Close())
	assert.Equal(t, Status{State: Died}, status(t, task))

	_, err = task.Take(t.Context())
	require.ErrorIs(t, err, ErrTaken, "a restarted main doesn't re-run the task its predecessor died in")
	assert.Equal(t, Status{State: Died}, status(t, task))
}

func TestWaitMain_ReturnsWhenMainFinishes(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")

	done := make(chan Status, 1)
	go func() {
		s, err := task.WaitMain(t.Context())
		assert.NoError(t, err)
		done <- s
	}()

	run, err := task.Take(t.Context())
	require.NoError(t, err)
	select {
	case s := <-done:
		t.Fatalf("WaitMain returned %v while main was running the task", s)
	case <-time.After(100 * time.Millisecond):
	}

	require.NoError(t, run.Finish(7, ""))
	select {
	case s := <-done:
		assert.Equal(t, Status{State: Exited, ExitCode: 7}, s)
	case <-time.After(5 * time.Second):
		t.Fatal("WaitMain did not return after main finished")
	}
}

// Finish writes exitcode before it releases the lock, so a worker woken by
// the lock's release never mistakes a finished task for a dead main.
func TestWaitMain_NeverSeesAFinishedTaskAsDied(t *testing.T) {
	p := newPool(t)
	for i := range 100 {
		task := offered(t, p, fmt.Sprintf("task-%d", i))
		run, err := task.Take(t.Context())
		require.NoError(t, err)
		done := make(chan Status, 1)
		go func() {
			s, err := task.WaitMain(t.Context())
			assert.NoError(t, err)
			done <- s
		}()
		require.NoError(t, run.Finish(i, ""))
		assert.Equal(t, Status{State: Exited, ExitCode: i}, <-done)
	}
}

// Whoever creates taken first wins, and nobody ever reads a half-written
// taken.
func TestTaken_FirstCreatorWins(t *testing.T) {
	p := newPool(t)
	for i := range 200 {
		task := offered(t, p, fmt.Sprintf("task-%d", i))
		var (
			wg        sync.WaitGroup
			run       *Run
			takeErr   error
			withdrawn bool
		)
		start := make(chan struct{})
		wg.Go(func() {
			<-start
			run, takeErr = task.Take(t.Context())
		})
		wg.Go(func() {
			<-start
			var err error
			withdrawn, err = task.Withdraw()
			assert.NoError(t, err)
		})
		close(start)
		wg.Wait()

		if withdrawn {
			require.ErrorIs(t, takeErr, ErrWithdrawn)
			assert.Equal(t, Status{State: Withdrawn}, status(t, task))
		} else {
			require.NoError(t, takeErr)
			assert.Equal(t, Status{State: Running}, status(t, task))
			require.NoError(t, run.Finish(0, ""))
		}
	}
}

// Main dying mid-task, however it dies, leaves the task Died, so it fails
// and is never re-run.
func TestStatus_MainKilled(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), "GO_POOLPOD_HELPER=take", "GO_POOLPOD_TASK="+task.Dir()) //nolint:forbidigo // env is the only channel to a re-exec'd test helper process
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	line, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "TAKEN\n", line)
	assert.Equal(t, Status{State: Running}, status(t, task))

	done := make(chan Status, 1)
	go func() {
		s, err := task.WaitMain(t.Context())
		assert.NoError(t, err)
		done <- s
	}()
	require.NoError(t, cmd.Process.Kill())
	select {
	case s := <-done:
		assert.Equal(t, Status{State: Died}, s)
	case <-time.After(5 * time.Second):
		t.Fatal("WaitMain did not return after main was killed")
	}
}
