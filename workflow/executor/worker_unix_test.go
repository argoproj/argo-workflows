//go:build !windows

package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
)

func TestRunMain_NonZeroExitIsFailureNotError(t *testing.T) {
	ctx, we, sink := newWorkerExecutor(t, wfv1.Template{Container: &apiv1.Container{Command: []string{"sh", "-c", "exit 3"}}}, time.Time{})
	require.NoError(t, runMain.Run(ctx, we))
	require.Len(t, sink.failures, 1)
	assert.Equal(t, "exit status 3", sink.failures[0])
}

func TestRunMain_DriverErrorReturned(t *testing.T) {
	ctx, we, sink := newWorkerExecutor(t, wfv1.Template{Container: &apiv1.Container{Command: []string{"/no/such/binary"}}}, time.Time{})
	require.Error(t, runMain.Run(ctx, we))
	assert.Empty(t, sink.failures)
}

func TestRunMain_DeadlineKillsChild(t *testing.T) {
	ctx, we, sink := newWorkerExecutor(t, wfv1.Template{Container: &apiv1.Container{Command: []string{"sh", "-c", "exec sleep 30"}}}, time.Now().Add(200*time.Millisecond))
	start := time.Now()
	require.NoError(t, runMain.Run(ctx, we))
	assert.Less(t, time.Since(start), 5*time.Second)
	require.NotEmpty(t, sink.failures)
	assert.Equal(t, "Step exceeded its deadline", sink.failures[0])
}

func TestRunMain_StagesOutputParameter(t *testing.T) {
	out := filepath.Join(t.TempDir(), "p")
	tmpl := wfv1.Template{
		Container: &apiv1.Container{Command: []string{"sh", "-c", `printf "$X" > ` + out}, Env: []apiv1.EnvVar{{Name: "X", Value: "v"}}},
		Outputs:   wfv1.Outputs{Parameters: []wfv1.Parameter{{Name: "p", ValueFrom: &wfv1.ValueFrom{Path: out}}}},
	}
	ctx, we, _ := newWorkerExecutor(t, tmpl, time.Time{})
	require.NoError(t, runMain.Run(ctx, we))
	staged, err := os.ReadFile(filepath.Join(we.paths.VarRunArgo, "outputs", "parameters", out))
	require.NoError(t, err)
	assert.Equal(t, "v", string(staged))
}

type blockingSink struct {
	recordingSink
	entered chan struct{}
	release chan struct{}
}

func (s *blockingSink) Fail(ctx context.Context, err error) {
	select {
	case s.entered <- struct{}{}:
	default:
	}
	<-s.release
	s.recordingSink.Fail(ctx, err)
}

func TestRunMain_DeadlineKillsBeforeReporting(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "terminated")
	ctx, we, _ := newWorkerExecutor(t, wfv1.Template{Container: &apiv1.Container{Command: []string{"sh", "-c",
		`trap 'touch ` + marker + `; exit 1' TERM; sleep 30 & wait`}}}, time.Now().Add(200*time.Millisecond))
	sink := &blockingSink{entered: make(chan struct{}, 1), release: make(chan struct{})}
	we.sink = sink
	done := make(chan error, 1)
	go func() { done <- runMain.Run(ctx, we) }()

	assert.Eventually(t, func() bool { _, err := os.Stat(marker); return err == nil }, 5*time.Second, 20*time.Millisecond,
		"main must get SIGTERM while the sink is still blocked")
	close(sink.release)
	require.NoError(t, <-done)
	assert.Equal(t, []string{"Step exceeded its deadline"}, sink.failures)
}

func startSleeper(t *testing.T) *exec.Cmd {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "sleep", "60")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	return cmd
}

type driverFunc func(context.Context, maindriver.Task, maindriver.OutputStager) (int, error)

func (f driverFunc) Run(ctx context.Context, task maindriver.Task, stager maindriver.OutputStager) (int, error) {
	return f(ctx, task, stager)
}

func TestRunMain_DeadlineAfterMainExitsIsIgnored(t *testing.T) {
	sleeper := startSleeper(t)
	ctx, we, sink := newWorkerExecutor(t, wfv1.Template{Container: &apiv1.Container{Command: []string{"true"}}}, time.Now().Add(100*time.Millisecond))
	we.mainDriver = driverFunc(func(_ context.Context, task maindriver.Task, _ maindriver.OutputStager) (int, error) {
		task.OnStart(sleeper.Process.Pid)
		task.OnExit()
		time.Sleep(400 * time.Millisecond) // staging outlasts the deadline
		return 0, nil
	})
	require.NoError(t, runMain.Run(ctx, we))
	assert.Empty(t, sink.failures)
	assert.NoError(t, sleeper.Process.Signal(syscall.Signal(0)), "the pid must not have been signalled")
}

func TestRunMain_PanicEndsWatch(t *testing.T) {
	sleeper := startSleeper(t)
	ctx, we, sink := newWorkerExecutor(t, wfv1.Template{Container: &apiv1.Container{Command: []string{"true"}}}, time.Time{})
	ctx, cancel := context.WithCancel(ctx)
	we.mainDriver = driverFunc(func(_ context.Context, task maindriver.Task, _ maindriver.OutputStager) (int, error) {
		task.OnStart(sleeper.Process.Pid)
		panic("stager blew up")
	})
	func() {
		defer func() { _ = recover() }()
		_ = runMain.Run(ctx, we)
	}()
	cancel() // the task ends; a leaked watcher would now report "worker stopped" and kill
	time.Sleep(300 * time.Millisecond)
	assert.Empty(t, sink.failures)
	assert.NoError(t, sleeper.Process.Signal(syscall.Signal(0)))
}

func TestRunMain_DeadlineEscalatesToSIGKILL(t *testing.T) {
	ctx, we, sink := newWorkerExecutor(t, wfv1.Template{Container: &apiv1.Container{Command: []string{"sh", "-c",
		`trap '' TERM; while true; do sleep 0.1; done`}}}, time.Now().Add(200*time.Millisecond))
	start := time.Now()
	require.NoError(t, runMain.Run(ctx, we))
	assert.Less(t, time.Since(start), 5*time.Second)
	assert.Equal(t, []string{"Step exceeded its deadline"}, sink.failures)
}

func TestRunMain_WorkerStopStopsMain(t *testing.T) {
	ctx, we, sink := newWorkerExecutor(t, wfv1.Template{Container: &apiv1.Container{Command: []string{"sh", "-c", "exec sleep 30"}}}, time.Time{})
	ctx, cancel := context.WithCancel(ctx)
	time.AfterFunc(200*time.Millisecond, cancel)
	start := time.Now()
	require.NoError(t, runMain.Run(ctx, we))
	assert.Less(t, time.Since(start), 5*time.Second)
	assert.Equal(t, []string{"worker stopped"}, sink.failures)
}
