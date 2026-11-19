// Package poolpodtest plays a pool pod's worker in tests of main, through
// the poolpod calls the worker makes.
package poolpodtest

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
)

// Offer does what the worker does once it has staged a task: it creates
// the task directory dir, writes spec as its task file and offers it in
// pool. It fails the test on any error.
func Offer(t testing.TB, pool poolpod.Pool, dir string, spec poolpod.TaskSpec) poolpod.Task {
	t.Helper()
	task := Prepare(t, dir, spec)
	require.NoError(t, pool.Offer(task))
	return task
}

// Prepare is Offer without the offer, for a test that withdraws or stops the
// task before main can see it.
func Prepare(t testing.TB, dir string, spec poolpod.TaskSpec) poolpod.Task {
	t.Helper()
	task := poolpod.NewTask(dir)
	require.NoError(t, task.Create())
	require.NoError(t, task.WriteSpec(spec))
	return task
}

// StageInput writes content as the input artifact name, staged in the
// task's input artifacts directory dir, as the worker's Prepare loads an
// artifact that isn't on a volume.
func StageInput(t testing.TB, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

// WaitMain waits up to timeout for main to be done with task, and returns
// its status: Withdrawn, Exited or Died.
func WaitMain(t testing.TB, task poolpod.Task, timeout time.Duration) poolpod.Status {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), timeout)
	defer cancel()
	s, err := task.WaitMain(ctx)
	require.NoError(t, err, "waiting for main to be done with %s", task.Dir())
	return s
}

// WaitRunning waits up to timeout for main to have taken task and still be
// running it.
func WaitRunning(t testing.TB, task poolpod.Task, timeout time.Duration) {
	t.Helper()
	require.Eventually(t, func() bool {
		s, err := task.Status()
		return err == nil && s.State == poolpod.Running
	}, timeout, 10*time.Millisecond, "main never ran %s", task.Dir())
}
