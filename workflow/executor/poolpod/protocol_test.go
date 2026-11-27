//go:build !windows

package poolpod

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPool(t *testing.T) Pool {
	t.Helper()
	p := NewPool(filepath.Join(t.TempDir(), "pool"))
	require.NoError(t, p.Init())
	return p
}

// newTask creates a task directory with the given name, as the worker
// would before writing the task's files into it.
func offered(t *testing.T, p Pool, name string) Task {
	t.Helper()
	task := newTask(t, name)
	require.NoError(t, p.Offer(task))
	return task
}

func status(t *testing.T, task Task) Status {
	t.Helper()
	s, err := task.Status()
	require.NoError(t, err)
	return s
}

func TestPool_InitMakesASharedDirectory(t *testing.T) {
	p := newPool(t)
	fi, err := os.Stat(p.Dir())
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o777), fi.Mode().Perm(), "main and worker may run as different users")
	require.NoError(t, p.Init(), "either side may initialise the pool directory, in either order")
}

func TestPool_CurrentBeforeAnyOffer(t *testing.T) {
	_, ok, err := newPool(t).Current()
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestPool_OfferPublishesAReadyTask(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")

	got, ok, err := p.Current()
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, task, got)
	assert.FileExists(t, filepath.Join(task.Dir(), readyFile))
	assert.Equal(t, Status{State: Offered}, status(t, task))
}

func TestPool_OfferRefusesAMissingTaskDirectory(t *testing.T) {
	p := newPool(t)
	require.Error(t, p.Offer(NewTask(filepath.Join(t.TempDir(), "missing"))))
	_, ok, err := p.Current()
	require.NoError(t, err)
	assert.False(t, ok, "current must never name a task directory that isn't complete")
}

func TestTask_CreateMakesASharedDirectory(t *testing.T) {
	task := newTask(t, "a")
	fi, err := os.Stat(task.Dir())
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o777), fi.Mode().Perm(), "main writes taken, lock and exitcode into the worker's task directory")
}

func TestTake_RefusesATaskNotOffered(t *testing.T) {
	task := newTask(t, "a")
	_, err := task.Take(t.Context())
	require.ErrorIs(t, err, ErrNotOffered)

	_, err = NewTask(filepath.Join(t.TempDir(), "deleted")).Take(t.Context())
	require.ErrorIs(t, err, ErrNotOffered, "a task directory the worker has already deleted")
}

// A task directory the worker removes while main is taking the task, once
// main has seen its ready, is a task not offered, not main's failure: here
// the lock's directory is gone by the time main creates it.
func TestTake_RefusesATaskRemovedAsItIsTaken(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "t")
	if err := os.Symlink(filepath.Join(t.TempDir(), "gone", LockFile), filepath.Join(task.Dir(), LockFile)); err != nil {
		t.Skipf("needs symlinks: %v", err)
	}

	_, err := task.Take(t.Context())
	require.ErrorIs(t, err, ErrNotOffered)
}

func TestWithdraw_IsIdempotent(t *testing.T) {
	task := offered(t, newPool(t), "a")
	for range 2 {
		withdrawn, err := task.Withdraw()
		require.NoError(t, err)
		assert.True(t, withdrawn, "a restarted worker withdrawing its own withdrawn task again still has it withdrawn")
	}
}

func TestWaitMain_ReturnsAtOnceForAWithdrawnTask(t *testing.T) {
	task := offered(t, newPool(t), "a")
	_, err := task.Withdraw()
	require.NoError(t, err)
	s, err := task.WaitMain(t.Context())
	require.NoError(t, err)
	assert.Equal(t, Status{State: Withdrawn}, s)
}

func TestWaitMain_Cancellation(t *testing.T) {
	task := offered(t, newPool(t), "a")
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err := task.WaitMain(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestCreateExclusive_ManyCreators(t *testing.T) {
	dir := t.TempDir()
	const n = 32
	var wg sync.WaitGroup
	created := make([]bool, n)
	start := make(chan struct{})
	for i := range n {
		wg.Go(func() {
			<-start
			var err error
			created[i], err = createExclusive(dir, "taken", []byte(fmt.Sprintf("creator-%d", i)))
			assert.NoError(t, err)
		})
	}
	// A regression that never creates taken fails here rather than hanging
	// until the test binary's timeout.
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	readers := make(chan string, n)
	for range n {
		wg.Go(func() {
			<-start
			for {
				data, err := os.ReadFile(filepath.Join(dir, "taken"))
				if err == nil {
					readers <- string(data)
					return
				}
				if !errors.Is(err, os.ErrNotExist) {
					t.Errorf("read taken: %v", err)
					return
				}
				if ctx.Err() != nil {
					assert.Fail(t, "taken was never created")
					return
				}
			}
		})
	}
	close(start)
	wg.Wait()
	close(readers)

	winners := 0
	winner := ""
	for i, c := range created {
		if c {
			winners++
			winner = fmt.Sprintf("creator-%d", i)
		}
	}
	require.Equal(t, 1, winners)
	for r := range readers {
		assert.Equal(t, winner, r, "a reader saw taken before its content was complete")
	}
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no temporary files are left behind")
}

// current is replaced atomically: a reader sees one complete path or
// another, never a partial or empty one.
func TestCurrent_UpdatesAreAtomic(t *testing.T) {
	p := newPool(t)
	base := t.TempDir()
	const n = 200
	tasks := make([]Task, n)
	valid := map[string]bool{}
	for i := range n {
		// Paths of different lengths, so a torn write would show.
		tasks[i] = NewTask(filepath.Join(base, fmt.Sprintf("task-%d-%s", i, strings.Repeat("x", i%37))))
		require.NoError(t, tasks[i].Create())
		valid[tasks[i].Dir()] = true
	}

	ctx, cancel := context.WithCancel(t.Context())
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for ctx.Err() == nil {
				got, ok, err := p.Current()
				if err != nil {
					t.Errorf("read current: %v", err)
					return
				}
				if ok && !valid[got.Dir()] {
					assert.Failf(t, "torn read of current", "%q", got.Dir())
					return
				}
			}
		})
	}
	for _, task := range tasks {
		require.NoError(t, p.Offer(task))
	}
	cancel()
	wg.Wait()

	got, ok, err := p.Current()
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, tasks[n-1], got)
	entries, err := os.ReadDir(p.Dir())
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no temporary files are left behind")
}

func TestWaitNext(t *testing.T) {
	p := newPool(t)

	got := make(chan Task, 1)
	go func() {
		task, err := p.WaitNext(t.Context(), Task{})
		assert.NoError(t, err)
		got <- task
	}()
	select {
	case task := <-got:
		t.Fatalf("WaitNext returned %v before any offer", task)
	case <-time.After(100 * time.Millisecond):
	}
	a := offered(t, p, "a")
	select {
	case task := <-got:
		assert.Equal(t, a, task)
	case <-time.After(5 * time.Second):
		t.Fatal("WaitNext did not see the offer")
	}

	// The current task is returned at once unless it is the one passed: a
	// restarted main looks at current before it waits.
	task, err := p.WaitNext(t.Context(), Task{})
	require.NoError(t, err)
	assert.Equal(t, a, task)

	go func() {
		task, err := p.WaitNext(t.Context(), a)
		assert.NoError(t, err)
		got <- task
	}()
	select {
	case task := <-got:
		t.Fatalf("WaitNext returned %v, the task it was told to skip", task)
	case <-time.After(100 * time.Millisecond):
	}
	b := offered(t, p, "b")
	select {
	case task := <-got:
		assert.Equal(t, b, task)
	case <-time.After(5 * time.Second):
		t.Fatal("WaitNext did not see the second offer")
	}
}

func TestWaitNext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err := newPool(t).WaitNext(ctx, Task{})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// A current main can't read is an error, not a wait for an offer.
func TestWaitNext_ReadError(t *testing.T) {
	p := newPool(t)
	require.NoError(t, os.WriteFile(filepath.Join(p.Dir(), CurrentFile), nil, 0o644))
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	_, err := p.WaitNext(ctx, Task{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is empty")
}

func TestStop(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")
	run, err := task.Take(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = run.Finish(0, "") })

	_, ok, err := run.StopRequested()
	require.NoError(t, err)
	assert.False(t, ok)

	got := make(chan StopReason, 1)
	go func() {
		r, waitErr := run.WaitStop(t.Context())
		assert.NoError(t, waitErr)
		got <- r
	}()
	select {
	case r := <-got:
		t.Fatalf("WaitStop returned %q before stop was written", r)
	case <-time.After(100 * time.Millisecond):
	}

	reason, err := task.Stop(StopDeadline)
	require.NoError(t, err)
	assert.Equal(t, StopDeadline, reason)
	select {
	case r := <-got:
		assert.Equal(t, StopDeadline, r)
	case <-time.After(5 * time.Second):
		t.Fatal("WaitStop did not see stop")
	}

	reason, err = task.Stop(StopRelease)
	require.NoError(t, err)
	assert.Equal(t, StopDeadline, reason, "the first reason written stands")
	reason, ok, err = run.StopRequested()
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, StopDeadline, reason)
}

func TestStop_WritesEachReason(t *testing.T) {
	for _, reason := range []StopReason{StopDeadline, StopInterrupt, StopLost, StopRelease} {
		task := newTask(t, string(reason))
		got, err := task.Stop(reason)
		require.NoError(t, err)
		assert.Equal(t, reason, got)
		data, err := os.ReadFile(filepath.Join(task.Dir(), stopFile))
		require.NoError(t, err)
		assert.Equal(t, string(reason), string(data))
	}
	_, err := newTask(t, "bad").Stop("bored")
	require.Error(t, err)
}

func TestWaitStop_Cancellation(t *testing.T) {
	task := offered(t, newPool(t), "a")
	run, err := task.Take(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = run.Finish(0, "") })
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err = run.WaitStop(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestStatus_RejectsUnknownTakenContent(t *testing.T) {
	task := offered(t, newPool(t), "a")
	require.NoError(t, os.WriteFile(filepath.Join(task.Dir(), takenFile), []byte("someone"), 0o644))
	_, err := task.Status()
	require.Error(t, err)
}

func TestStatus_RejectsAMalformedExitCode(t *testing.T) {
	p := newPool(t)
	task := offered(t, p, "a")
	run, err := task.Take(t.Context())
	require.NoError(t, err)
	require.NoError(t, run.lock.Close())
	require.NoError(t, os.WriteFile(filepath.Join(task.Dir(), exitCodeFile), []byte("x"), 0o644))
	_, err = task.Status()
	require.Error(t, err)
}
