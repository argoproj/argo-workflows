//go:build !windows

// These tests play main, whose lock probes (osspecific.LockHeld) use flock
// and so don't build on Windows.

package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
)

// The task's exit code is collected once main is done with it.
func TestRunPoolMain(t *testing.T) {
	root := t.TempDir()
	sink := &phaseSink{}
	ctx, we := newPoolExecutor(t, root, "task-1", TaskConfig{
		NodeID:   "node-1",
		Template: wfv1.Template{Container: &apiv1.Container{Command: []string{"true"}}},
	}, sink)
	require.NoError(t, poolpod.NewPool(filepath.Join(poolBase(root).VarRunArgo, poolpod.DirName)).Init())
	require.NoError(t, createPoolTask.Run(ctx, we))
	require.NoError(t, writePoolTask.Run(ctx, we))

	// Plays main: takes the task once it is offered and exits 2.
	pool := poolpod.NewPool(filepath.Join(poolBase(root).VarRunArgo, poolpod.DirName))
	mainDone := make(chan error, 1)
	go func() {
		task, err := pool.WaitNext(ctx, poolpod.Task{})
		if err != nil {
			mainDone <- err
			return
		}
		run, err := task.Take(ctx)
		if err != nil {
			mainDone <- err
			return
		}
		mainDone <- run.Finish(2, "")
	}()

	require.NoError(t, we.runPoolMain(ctx))
	require.NoError(t, <-mainDone)
	assert.False(t, we.poolTask.busy)
	require.NotNil(t, we.capturedOutputs().ExitCode)
	assert.Equal(t, "2", *we.capturedOutputs().ExitCode)
	assert.Equal(t, []string{"main: Error (exit code 2)"}, sink.failures)
	current, ok, err := pool.Current()
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, we.paths.VarRunArgo, current.Dir())
}

// progressSink records the progress reported.
type progressSink struct {
	recordingSink
	progress chan wfv1.Progress
}

func (s *progressSink) Progress(_ context.Context, progress wfv1.Progress) error {
	s.progress <- progress
	return nil
}

// The task's progress, which main writes to the file in the task's own
// directory it is told of, as the pool's source gives each task, is
// reported through the sink while main runs the task.
func TestRunPoolMain_ReportsProgress(t *testing.T) {
	root := t.TempDir()
	paths, err := poolBase(root).PoolTask("task-1")
	require.NoError(t, err)
	progressFile := filepath.Join(paths.VarRunArgo, "progress")
	sink := &progressSink{progress: make(chan wfv1.Progress, 10)}
	ctx, we := newPoolExecutor(t, root, "task-1", TaskConfig{
		NodeID:                    "node-1",
		Template:                  wfv1.Template{Container: &apiv1.Container{Command: []string{"true"}}},
		ProgressFile:              progressFile,
		ProgressPatchTickDuration: 10 * time.Millisecond,
		ProgressFileTickDuration:  10 * time.Millisecond,
	}, sink)
	assert.Contains(t, we.mainEnv, common.EnvVarProgressFile+"="+progressFile)
	pool := poolpod.NewPool(filepath.Join(poolBase(root).VarRunArgo, poolpod.DirName))
	require.NoError(t, pool.Init())
	require.NoError(t, createPoolTask.Run(ctx, we))
	require.NoError(t, writePoolTask.Run(ctx, we))

	// Plays main: reports progress, and exits once it is reported.
	var reported wfv1.Progress
	mainDone := make(chan error, 1)
	go func() {
		task, err := pool.WaitNext(ctx, poolpod.Task{})
		if err != nil {
			mainDone <- err
			return
		}
		run, err := task.Take(ctx)
		if err != nil {
			mainDone <- err
			return
		}
		if err := os.WriteFile(progressFile, []byte("1/2\n"), 0o644); err != nil {
			mainDone <- err
			return
		}
		select {
		case reported = <-sink.progress:
		case <-time.After(10 * time.Second):
		}
		mainDone <- run.Finish(0, "")
	}()

	require.NoError(t, we.runPoolMain(ctx))
	require.NoError(t, <-mainDone)
	assert.Equal(t, wfv1.Progress("1/2"), reported, "reported while main ran the task")
}
