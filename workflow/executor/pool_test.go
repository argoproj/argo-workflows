package executor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/mocks"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// poolBase is a pool pod's shared directories under root; main's volumes
// are where the worker sees them, so MainFS is empty.
func poolBase(root string) TaskPaths {
	return TaskPaths{
		VarRunArgo:     filepath.Join(root, "var/run/argo"),
		InputArtifacts: filepath.Join(root, "argo/inputs/artifacts"),
		Staging:        filepath.Join(root, "argo/staging"),
		Scratch:        filepath.Join(root, "tmp/argo"),
	}
}

// newPoolExecutor returns the executor of a pool pod's task called name,
// whose shared directories are under root, with a sink recording into
// sink.
func newPoolExecutor(t *testing.T, root, name string, cfg TaskConfig, sink ResultSink) (context.Context, *WorkflowExecutor) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	tr, err := tracing.New(ctx, `argoexec`)
	require.NoError(t, err)
	paths, err := poolBase(root).PoolTask(name)
	require.NoError(t, err)
	p := &Process{
		Tracing:    tr,
		newRuntime: func(TaskPaths) ContainerRuntimeExecutor { return &mocks.ContainerRuntimeExecutor{} },
		pool:       poolProcess{terminatedCh: make(chan struct{})},
	}
	return ctx, p.NewExecutor(Task{Config: cfg, Paths: paths}, LayoutPool, sink)
}

func TestPlanFor_PoolLayout(t *testing.T) {
	for _, tmpl := range []*wfv1.Template{{Container: &apiv1.Container{}}, {Script: &wfv1.ScriptTemplate{}}} {
		plan := planFor(tmpl, LayoutPool, []wfv1.ArtifactPluginName{"plugin"})
		assert.Equal(t, []string{"clear-pool", "create-pool-task", "stage-files", "parallel(load-artifacts,load-artifacts-from-plugin(plugin))", "write-pool-task"}, stageNames(plan.Prepare))
		assert.Equal(t, []string{"initialize-output", "run-in-pool"}, stageNames(plan.Run))
		assert.Equal(t, []string{"capture-script-result", "save-parameters", "save-artifacts", "save-logs", "report-outputs"}, stageNames(plan.Collect))
	}
	assert.Equal(t, []string{"complete-pool-task", "remove-pool-task", "wipe-empty-dirs"}, stageNames(PoolRole.Finally),
		"the pool's plugin sidecars are shared, so they are never killed")
}

// main is told the container's command and args as written, the script it
// runs, the image's command only when the container has none, and the
// template's plain env followed by the task's own.
func TestPoolTaskSpec(t *testing.T) {
	cfg := TaskConfig{
		NodeID:              "node-1",
		WorkflowName:        "wf",
		IncludeScriptOutput: true,
		Command:             []string{"/entrypoint", "cmd"},
		Template: wfv1.Template{Script: &wfv1.ScriptTemplate{
			Container: apiv1.Container{
				Command: []string{"sh"},
				Args:    []string{"-e"},
				Env: []apiv1.EnvVar{
					{Name: "A", Value: "$(B)"},
					{Name: "FROM", ValueFrom: &apiv1.EnvVarSource{SecretKeyRef: &apiv1.SecretKeySelector{Key: "k"}}},
				},
			},
			Source: "echo hi",
		}},
	}
	_, we := newPoolExecutor(t, t.TempDir(), "task-1", cfg, &recordingSink{})
	spec, err := we.poolTaskSpec()
	require.NoError(t, err)
	assert.Equal(t, "node-1", spec.NodeID)
	assert.Equal(t, &we.Template, spec.Template)
	assert.Equal(t, []string{"sh"}, spec.Command)
	assert.Empty(t, spec.ImageCommand, "the container has a command")
	assert.Equal(t, []string{"-e", we.paths.ScriptSource()}, spec.Args)
	assert.True(t, spec.IncludeScriptOutput)
	wantEnv := []poolpod.EnvVar{{Name: "A", Value: "$(B)"}}
	for _, kv := range cfg.MainEnv() {
		name, value, _ := strings.Cut(kv, "=")
		wantEnv = append(wantEnv, poolpod.EnvVar{Name: name, Value: value})
	}
	assert.Equal(t, wantEnv, spec.Env, "valueFrom entries are main's own")

	cfg.Template = wfv1.Template{Container: &apiv1.Container{Image: "busybox"}}
	_, we = newPoolExecutor(t, t.TempDir(), "task-2", cfg, &recordingSink{})
	spec, err = we.poolTaskSpec()
	require.NoError(t, err)
	assert.Empty(t, spec.Command)
	assert.Equal(t, []string{"/entrypoint", "cmd"}, spec.ImageCommand)
	assert.Empty(t, spec.Args)

	cfg.Template = wfv1.Template{Resource: &wfv1.ResourceTemplate{}}
	_, we = newPoolExecutor(t, t.TempDir(), "task-3", cfg, &recordingSink{})
	_, err = we.poolTaskSpec()
	require.Error(t, err)
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o644))
}

// Before a task, whatever other tasks left in the pool pod's shared
// directories is removed, as are the contents of main's emptyDir volumes,
// but not other volumes mounted beneath them, nor current.
func TestClearPool(t *testing.T) {
	root := t.TempDir()
	vol := filepath.Join(root, "vol")
	tmpl := wfv1.Template{
		Container: &apiv1.Container{VolumeMounts: []apiv1.VolumeMount{
			{Name: "empty", MountPath: filepath.Join(vol, "empty")},
			{Name: "claim", MountPath: filepath.Join(vol, "empty", "sub", "claim")},
			{Name: "other", MountPath: filepath.Join(vol, "other")},
		}},
	}
	cfg := TaskConfig{NodeID: "node-1", Template: tmpl, EmptyDirs: []string{filepath.Join(vol, "empty")}}
	ctx, we := newPoolExecutor(t, root, "task-1", cfg, &recordingSink{})
	base := poolBase(root)
	pool := filepath.Join(base.VarRunArgo, poolpod.DirName)
	leftovers := []string{
		filepath.Join(pool, "old-task", "taken"),
		filepath.Join(pool, ".current-123"),
		filepath.Join(base.InputArtifacts, "old-task", "in"),
		filepath.Join(base.Staging, "old-task", "script"),
		filepath.Join(base.Scratch, "outputs", "old-task", "outputs", "logs", "main.log"),
		filepath.Join(vol, "empty", "file"),
		filepath.Join(vol, "empty", "dir", "file"),
		filepath.Join(vol, "empty", "sub", "file"),
	}
	kept := []string{
		filepath.Join(pool, poolpod.CurrentFile),
		filepath.Join(pool, poolpod.MainFile),
		filepath.Join(vol, "empty", "sub", "claim", "file"),
		filepath.Join(vol, "other", "file"),
	}
	for _, f := range append(append([]string{}, leftovers...), kept...) {
		writeFile(t, f)
	}

	require.NoError(t, clearPool.Run(ctx, we))
	require.NoError(t, createPoolTask.Run(ctx, we))

	for _, f := range leftovers {
		assert.NoFileExists(t, f)
	}
	for _, f := range kept {
		assert.FileExists(t, f)
	}
	assert.NoDirExists(t, filepath.Join(pool, "old-task"))
	assert.NoDirExists(t, filepath.Join(vol, "empty", "dir"))
	assert.DirExists(t, filepath.Join(vol, "empty"), "a mount point stays")
	for _, dir := range we.paths.poolDirs() {
		assert.DirExists(t, dir)
	}
	for _, dir := range []string{we.paths.VarRunArgo, we.paths.InputArtifacts, we.paths.Staging} {
		fi, err := os.Stat(dir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o777), fi.Mode().Perm(), "main may run as another user: %s", dir)
	}

	// After the task, its own directories go, and the emptyDir is wiped.
	writeFile(t, filepath.Join(vol, "empty", "after"))
	require.NoError(t, removePoolTask.Run(ctx, we))
	require.NoError(t, wipeEmptyDirs.Run(ctx, we))
	for _, dir := range we.paths.poolDirs() {
		assert.NoDirExists(t, dir)
	}
	assert.NoFileExists(t, filepath.Join(vol, "empty", "after"))
	assert.FileExists(t, filepath.Join(pool, poolpod.CurrentFile))
}

// undeletable makes a directory in dir holding a file that can't be removed,
// as main leaves one when it runs as another user, or skips the test where
// permissions can't stop the removal.
func undeletable(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs a user that directory permissions apply to")
	}
	locked := filepath.Join(dir, "locked")
	writeFile(t, filepath.Join(locked, "file"))
	require.NoError(t, os.Chmod(locked, 0o555))
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
}

// A pool that can't be cleared before a claimed task releases the task,
// which nothing has run, rather than failing it, and stops the worker,
// whose restart clears the pool again before it claims anything.
func TestPoolRole_ClearPoolFails(t *testing.T) {
	root := t.TempDir()
	undeletable(t, filepath.Join(poolBase(root).VarRunArgo, poolpod.DirName))
	sink := &stoppingSink{}
	ctx, we := newPoolExecutor(t, root, "task-1", TaskConfig{NodeID: "node-1", Template: wfv1.Template{Container: &apiv1.Container{}}}, sink)
	var stopped []error
	we.pool.stopWorker = func(err error) { stopped = append(stopped, err) }

	require.NoError(t, PoolRole.Run(ctx, ctx, we))
	require.NoError(t, completePoolTask.Run(ctx, we))

	assert.Equal(t, poolReleased, we.poolTask.stop)
	assert.Equal(t, 1, sink.released)
	assert.Zero(t, sink.finalized)
	assert.Empty(t, sink.failures)
	require.Len(t, stopped, 1)
	assert.ErrorContains(t, stopped[0], "failed to clear the pool")
}

// Once a task is settled, failing to remove its files or wipe main's
// emptyDir volumes stops the worker before it claims another task; the
// task's result stands.
func TestPoolFinally_CleanupFails(t *testing.T) {
	for _, tt := range []struct {
		name  string
		stage Stage
		dir   func(we *WorkflowExecutor, vol string) string
	}{
		{name: "RemovePoolTask", stage: removePoolTask, dir: func(we *WorkflowExecutor, _ string) string { return we.paths.VarRunArgo }},
		{name: "WipeEmptyDirs", stage: wipeEmptyDirs, dir: func(_ *WorkflowExecutor, vol string) string { return vol }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			vol := filepath.Join(root, "vol")
			cfg := TaskConfig{
				NodeID:    "node-1",
				Template:  wfv1.Template{Container: &apiv1.Container{VolumeMounts: []apiv1.VolumeMount{{Name: "empty", MountPath: vol}}}},
				EmptyDirs: []string{vol},
			}
			sink := &finalizingSink{}
			ctx, we := newPoolExecutor(t, root, "task-1", cfg, sink)
			var stopped []error
			we.pool.stopWorker = func(err error) { stopped = append(stopped, err) }
			require.NoError(t, createPoolTask.Run(ctx, we))
			undeletable(t, tt.dir(we, vol))
			require.NoError(t, completePoolTask.Run(ctx, we))

			require.Error(t, tt.stage.Run(ctx, we))

			assert.Equal(t, 1, sink.finalized, "the task's result stands")
			assert.Len(t, stopped, 1)
		})
	}
}

// At its start, the worker removes whatever an earlier run left in the
// pool pod's shared directories, but current and main's lock, and fails if
// it can't.
func TestClearPoolAtStart(t *testing.T) {
	root := t.TempDir()
	base := poolBase(root)
	pool := filepath.Join(base.VarRunArgo, poolpod.DirName)
	leftovers := []string{
		filepath.Join(pool, "old-task", "taken"),
		filepath.Join(base.InputArtifacts, "old-task", "in"),
		filepath.Join(base.Staging, "old-task", "script"),
		filepath.Join(base.Scratch, "outputs", "old-task", "main.log"),
	}
	kept := []string{filepath.Join(pool, poolpod.CurrentFile), filepath.Join(pool, poolpod.MainFile)}
	for _, f := range slices.Concat(leftovers, kept) {
		writeFile(t, f)
	}

	require.NoError(t, ClearPool(base))

	for _, f := range leftovers {
		assert.NoFileExists(t, f)
	}
	for _, f := range kept {
		assert.FileExists(t, f)
	}
	undeletable(t, base.Staging)
	require.Error(t, ClearPool(base))
}

// While main may still be running the task, nothing of it is removed or
// wiped, its claim is left incomplete, and the worker is stopped.
func TestPoolFinally_LeavesABusyTask(t *testing.T) {
	root := t.TempDir()
	vol := filepath.Join(root, "vol")
	cfg := TaskConfig{
		NodeID:    "node-1",
		Template:  wfv1.Template{Container: &apiv1.Container{VolumeMounts: []apiv1.VolumeMount{{Name: "empty", MountPath: vol}}}},
		EmptyDirs: []string{vol},
	}
	sink := &finalizingSink{}
	ctx, we := newPoolExecutor(t, root, "task-1", cfg, sink)
	var stopped []error
	we.pool.stopWorker = func(err error) { stopped = append(stopped, err) }
	require.NoError(t, createPoolTask.Run(ctx, we))
	writeFile(t, filepath.Join(vol, "file"))
	we.poolTask.busy = true

	require.Error(t, completePoolTask.Run(ctx, we))
	require.NoError(t, removePoolTask.Run(ctx, we))
	require.NoError(t, wipeEmptyDirs.Run(ctx, we))

	assert.Zero(t, sink.finalized, "the worker's restart fails the claim once main has stopped the task")
	assert.Len(t, stopped, 1)
	for _, dir := range we.paths.poolDirs() {
		assert.DirExists(t, dir)
	}
	assert.FileExists(t, filepath.Join(vol, "file"))
}

// finalizingSink counts Finalize calls, which fail with err.
type finalizingSink struct {
	recordingSink
	finalized int
	err       error
}

func (s *finalizingSink) Finalize(context.Context) error {
	s.finalized++
	return s.err
}

// A claim found lost when completing it isn't the worker's to report, so
// the worker goes on; any other failure to complete it stops the worker,
// whose restart fails the claim.
func TestCompletePoolTask(t *testing.T) {
	for _, tt := range []struct {
		name     string
		err      error
		wantStop bool
	}{
		{name: "Completed"},
		{name: "ClaimLost", err: dispatch.ErrClaimLost},
		{name: "GaveUp", err: errors.New("gave up writing task result"), wantStop: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sink := &finalizingSink{err: tt.err}
			ctx, we := newPoolExecutor(t, t.TempDir(), "task-1", TaskConfig{NodeID: "node-1"}, sink)
			var stopped []error
			we.pool.stopWorker = func(err error) { stopped = append(stopped, err) }

			err := completePoolTask.Run(ctx, we)

			assert.Equal(t, 1, sink.finalized)
			if tt.wantStop {
				require.ErrorIs(t, err, tt.err)
				require.Len(t, stopped, 1)
				require.ErrorIs(t, stopped[0], tt.err)
			} else {
				require.NoError(t, err)
				assert.Empty(t, stopped)
			}
		})
	}
}

// How main ended is the task's outcome, as a pod's main container's is:
// its exit code is reported, a non-zero one or main dying fails the task as
// Failed, and outputs main never staged aren't collected.
func TestMainEnded(t *testing.T) {
	for _, tt := range []struct {
		name          string
		status        poolpod.Status
		wantExitCode  string
		wantFailure   string
		wantNoOutputs bool
	}{
		{name: "Succeeded", status: poolpod.Status{State: poolpod.Exited}, wantExitCode: "0"},
		{name: "ExitedNonZero", status: poolpod.Status{State: poolpod.Exited, ExitCode: 3}, wantExitCode: "3", wantFailure: "main: Error (exit code 3)"},
		{
			name:          "MainFailedIt",
			status:        poolpod.Status{State: poolpod.Exited, ExitCode: 64, Message: "/in already exists in the container image"},
			wantExitCode:  "64",
			wantFailure:   "main: Error (exit code 64): /in already exists in the container image",
			wantNoOutputs: true,
		},
		{name: "Died", status: poolpod.Status{State: poolpod.Died}, wantFailure: "main: the container exited before the task ended", wantNoOutputs: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sink := &phaseSink{}
			ctx, we := newPoolExecutor(t, t.TempDir(), "task-1", TaskConfig{NodeID: "node-1"}, sink)

			require.NoError(t, we.mainEnded(ctx, tt.status))

			if tt.wantExitCode == "" {
				assert.Nil(t, we.capturedOutputs().ExitCode)
			} else {
				require.NotNil(t, we.capturedOutputs().ExitCode)
				assert.Equal(t, tt.wantExitCode, *we.capturedOutputs().ExitCode)
			}
			if tt.wantFailure == "" {
				assert.Empty(t, sink.failures)
			} else {
				assert.Equal(t, []string{tt.wantFailure}, sink.failures)
				assert.Equal(t, []wfv1.NodePhase{wfv1.NodeFailed}, sink.phases, "the task's own failure")
			}
			assert.Equal(t, tt.wantNoOutputs, we.poolTask.noMainOutputs)
		})
	}
	t.Run("Withdrawn", func(t *testing.T) {
		ctx, we := newPoolExecutor(t, t.TempDir(), "task-1", TaskConfig{NodeID: "node-1"}, &phaseSink{})
		require.Error(t, we.mainEnded(ctx, poolpod.Status{State: poolpod.Withdrawn}))
		assert.True(t, we.poolTask.noMainOutputs)
	})
}

// phaseSink records each failure and the phase a pooled task gets for it.
type phaseSink struct {
	recordingSink
	phases []wfv1.NodePhase
}

func (s *phaseSink) Fail(ctx context.Context, err error) {
	s.recordingSink.Fail(ctx, err)
	s.phases = append(s.phases, poolTaskFailurePhase(err))
}

// A failure of the task's own fails a pooled task as Failed; any other is
// the worker's, which makes it Error, as for a pod's containers.
func TestPoolTaskFailurePhase(t *testing.T) {
	assert.Equal(t, wfv1.NodeFailed, poolTaskFailurePhase(taskFailure{errors.New("exit")}))
	assert.Equal(t, wfv1.NodeError, poolTaskFailurePhase(errors.New("load-artifacts: not found")))
}
