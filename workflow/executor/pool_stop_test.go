package executor

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/mocks"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
)

// The controller's interrupt and a lost claim decide what is reported
// whatever stopped the task before; otherwise the first reason stands, and
// only a deadline that stands fails the task.
func TestRecordPoolStop(t *testing.T) {
	for _, tt := range []struct {
		name         string
		stops        []poolStop
		want         poolStop
		wantFailures []string
	}{
		{name: "Deadline", stops: []poolStop{poolDeadline}, want: poolDeadline, wantFailures: []string{common.DeadlineExceededMessage}},
		{name: "DeadlineThenRelease", stops: []poolStop{poolDeadline, poolReleased}, want: poolDeadline, wantFailures: []string{common.DeadlineExceededMessage}},
		{name: "ReleaseThenDeadline", stops: []poolStop{poolReleased, poolDeadline}, want: poolReleased},
		{name: "DeadlineThenInterrupt", stops: []poolStop{poolDeadline, poolInterrupted}, want: poolInterrupted, wantFailures: []string{common.DeadlineExceededMessage}},
		{name: "ReleaseThenLost", stops: []poolStop{poolReleased, poolLost}, want: poolLost},
		{name: "LostThenInterrupt", stops: []poolStop{poolLost, poolInterrupted}, want: poolLost},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sink := &phaseSink{}
			// An absolute deadline: activeDeadlineSeconds' is the kubelet's.
			ctx, we := newPoolExecutor(t, t.TempDir(), "task-1", TaskConfig{NodeID: "node-1", Deadline: time.Now()}, sink)
			for _, stop := range tt.stops {
				we.recordPoolStop(ctx, stop)
			}
			assert.Equal(t, tt.want, we.poolTask.stop)
			assert.Equal(t, tt.wantFailures, sink.failures)
			for _, phase := range sink.phases {
				assert.Equal(t, wfv1.NodeFailed, phase, "a deadline is the task's own failure")
			}
		})
	}
}

// Which reason to stop a task overrides which is explicit: the first
// stands, but the controller's interrupt overrides a deadline or a
// release, and a lost claim overrides any other.
func TestPoolStopOverrides(t *testing.T) {
	stops := []poolStop{poolDeadline, poolReleased, poolInterrupted, poolLost}
	overrides := map[[2]poolStop]bool{
		{poolInterrupted, poolDeadline}: true,
		{poolInterrupted, poolReleased}: true,
		{poolLost, poolDeadline}:        true,
		{poolLost, poolReleased}:        true,
		{poolLost, poolInterrupted}:     true,
	}
	for _, stop := range stops {
		assert.True(t, stop.overrides(poolNotStopped), "%s on a task not stopped", stop)
		for _, prev := range stops {
			assert.Equal(t, overrides[[2]poolStop{stop, prev}], stop.overrides(prev), "%s after %s", stop, prev)
		}
	}
}

// A task stopped at its activeDeadlineSeconds fails with the kubelet's
// message for a pod it stopped then, and one stopped at its absolute
// deadline with the executor's, as a pod's do.
func TestDeadlineMessage(t *testing.T) {
	end := time.Now()
	for _, tt := range []struct {
		name string
		cfg  TaskConfig
		want string
	}{
		{name: "ActiveDeadlineSeconds", cfg: TaskConfig{Deadline: end, ActiveDeadline: end}, want: "Pod was active on the node longer than the specified deadline"},
		{name: "Absolute", cfg: TaskConfig{Deadline: end}, want: "Step exceeded its deadline"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sink := &phaseSink{}
			tt.cfg.NodeID = "node-1"
			ctx, we := newPoolExecutor(t, t.TempDir(), "task-1", tt.cfg, sink)
			we.recordPoolStop(ctx, poolDeadline)
			assert.Equal(t, []string{tt.want}, sink.failures)
			assert.Equal(t, []wfv1.NodePhase{wfv1.NodeFailed}, sink.phases)
		})
	}
}

// A task stopped because it was released, interrupted or its claim lost is
// reported for that, whatever its exit code, including main's 143 for a
// task stopped before it started, and nothing it produced is collected.
func TestMainEnded_Stopped(t *testing.T) {
	for _, stop := range []poolStop{poolReleased, poolInterrupted, poolLost} {
		for _, status := range []poolpod.Status{{State: poolpod.Exited, ExitCode: 143}, {State: poolpod.Withdrawn}} {
			t.Run(stop.String()+"/"+status.State.String(), func(t *testing.T) {
				sink := &phaseSink{}
				ctx, we := newPoolExecutor(t, t.TempDir(), "task-1", TaskConfig{NodeID: "node-1"}, sink)
				we.poolTask.stop = stop
				require.NoError(t, we.mainEnded(ctx, status))
				assert.Empty(t, sink.failures)
				assert.Nil(t, we.capturedOutputs().ExitCode)
				assert.True(t, we.poolTask.noMainOutputs)
			})
		}
	}
	t.Run("DeadlineKeepsItsFailure", func(t *testing.T) {
		sink := &phaseSink{}
		ctx, we := newPoolExecutor(t, t.TempDir(), "task-1", TaskConfig{NodeID: "node-1"}, sink)
		we.recordPoolStop(ctx, poolDeadline)
		require.NoError(t, we.mainEnded(ctx, poolpod.Status{State: poolpod.Exited, ExitCode: 143}))
		require.NotNil(t, we.capturedOutputs().ExitCode)
		assert.Equal(t, "143", *we.capturedOutputs().ExitCode)
		assert.Equal(t, common.DeadlineExceededMessage, sink.failures[0], "the first failure stands")
	})
}

// stoppingSink is a sink that can be stopped, released and finalized, and
// counts each.
type stoppingSink struct {
	phaseSink
	stopped   chan struct{}
	reason    dispatch.StopReason
	released  int
	finalized int
	// releaseErr is what Release returns.
	releaseErr error
}

func (s *stoppingSink) Stopped() <-chan struct{}        { return s.stopped }
func (s *stoppingSink) StopReason() dispatch.StopReason { return s.reason }
func (s *stoppingSink) Release(context.Context) error   { s.released++; return s.releaseErr }
func (s *stoppingSink) Finalize(context.Context) error  { s.finalized++; return nil }

// What is reported for a stopped task: a released one is given back, a
// lost one reports nothing, and an interrupted one is completed to confirm
// it stopped.
func TestCompletePoolTask_Stopped(t *testing.T) {
	for _, tt := range []struct {
		stop          poolStop
		wantReleased  int
		wantFinalized int
	}{
		{stop: poolReleased, wantReleased: 1},
		{stop: poolLost},
		{stop: poolInterrupted, wantFinalized: 1},
		{stop: poolDeadline, wantFinalized: 1},
	} {
		t.Run(tt.stop.String(), func(t *testing.T) {
			sink := &stoppingSink{}
			ctx, we := newPoolExecutor(t, t.TempDir(), "task-1", TaskConfig{NodeID: "node-1"}, sink)
			we.poolTask.stop = tt.stop
			require.NoError(t, completePoolTask.Run(ctx, we))
			assert.Equal(t, tt.wantReleased, sink.released)
			assert.Equal(t, tt.wantFinalized, sink.finalized)
		})
	}
}

// What is collected for a task stopped from outside main: everything for
// one at its deadline, only main's log for an interrupted one, as a stopped
// pod's log is archived, and nothing for one released or whose claim was
// lost.
func TestRunPoolTask_CollectsForTheStop(t *testing.T) {
	for _, tt := range []struct {
		stop          poolStop
		wantCollected bool
		wantLogs      bool
	}{
		{stop: poolNotStopped, wantCollected: true},
		{stop: poolDeadline, wantCollected: true},
		{stop: poolInterrupted, wantLogs: true},
		{stop: poolReleased},
		{stop: poolLost},
	} {
		t.Run(tt.stop.String(), func(t *testing.T) {
			ctx, we := newPoolExecutor(t, t.TempDir(), "task-1", TaskConfig{
				NodeID:   "node-1",
				Template: wfv1.Template{Container: &apiv1.Container{}, ArchiveLocation: &wfv1.ArtifactLocation{ArchiveLogs: new(true)}},
			}, &recordingSink{})
			runtime := &mocks.ContainerRuntimeExecutor{}
			// main's log is gone, so nothing is archived or reported.
			runtime.On("GetOutputStream", mock.Anything, common.MainContainerName, true).Return(nil, fs.ErrNotExist).Maybe()
			we.RuntimeExecutor = runtime
			we.poolTask.stop = tt.stop
			collected := false
			we.plan = &Plan{Collect: []Stage{{Name: "collect", Run: func(context.Context, *WorkflowExecutor) error {
				collected = true
				return nil
			}}}}

			require.NoError(t, we.runPoolTask(ctx, ctx))

			assert.Equal(t, tt.wantCollected, collected)
			if tt.wantLogs {
				runtime.AssertCalled(t, "GetOutputStream", mock.Anything, common.MainContainerName, true)
			} else {
				runtime.AssertNotCalled(t, "GetOutputStream", mock.Anything, common.MainContainerName, true)
			}
		})
	}
}

// A released task's claim found lost isn't the worker's to give back, so
// the worker goes on; any other failure to release it leaves the claim
// incomplete, so the worker stops, and its restart fails the claim.
func TestCompletePoolTask_ReleaseFails(t *testing.T) {
	for _, tt := range []struct {
		name     string
		err      error
		wantStop bool
	}{
		{name: "Released"},
		{name: "ClaimLost", err: dispatch.ErrClaimLost},
		{name: "Failed", err: errors.New("forbidden"), wantStop: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sink := &stoppingSink{releaseErr: tt.err}
			ctx, we := newPoolExecutor(t, t.TempDir(), "task-1", TaskConfig{NodeID: "node-1"}, sink)
			we.poolTask.stop = poolReleased
			var stopped []error
			we.pool.stopWorker = func(err error) { stopped = append(stopped, err) }

			err := completePoolTask.Run(ctx, we)

			assert.Equal(t, 1, sink.released)
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

// A task that must stop while it is prepared is never offered to main: a
// terminated worker releases it, and one past its deadline fails.
func TestRunPoolMain_StoppedBeforeTheOffer(t *testing.T) {
	for _, tt := range []struct {
		name         string
		setup        func(*WorkflowExecutor, *stoppingSink)
		want         poolStop
		wantFailures []string
	}{
		{name: "Terminated", setup: func(we *WorkflowExecutor, _ *stoppingSink) { we.Terminate() }, want: poolReleased},
		{name: "Deadline", setup: func(we *WorkflowExecutor, _ *stoppingSink) { we.Deadline = time.Now().Add(-time.Second) }, want: poolDeadline, wantFailures: []string{common.DeadlineExceededMessage}},
		{name: "Lost", setup: func(_ *WorkflowExecutor, s *stoppingSink) { s.reason = dispatch.StopLost; close(s.stopped) }, want: poolLost},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			sink := &stoppingSink{stopped: make(chan struct{})}
			ctx, we := newPoolExecutor(t, root, "task-1", TaskConfig{NodeID: "node-1"}, sink)
			tt.setup(we, sink)
			require.NoError(t, createPoolTask.Run(ctx, we))

			require.NoError(t, we.runPoolMain(ctx))

			assert.Equal(t, tt.want, we.poolTask.stop)
			assert.Equal(t, tt.wantFailures, sink.failures)
			assert.False(t, we.poolTask.busy)
			_, offered, err := poolpod.NewPool(filepath.Join(poolBase(root).VarRunArgo, poolpod.DirName)).Current()
			require.NoError(t, err)
			assert.False(t, offered, "main never sees it")
		})
	}
}

// Once terminated, a task main hasn't taken is withdrawn and released at
// once.
func TestWaitPoolMain_TerminatedBeforeMainTakesIt(t *testing.T) {
	root := t.TempDir()
	sink := &stoppingSink{}
	ctx, we := newPoolExecutor(t, root, "task-1", TaskConfig{NodeID: "node-1"}, sink)
	we.terminationGracePeriod = time.Hour
	task := poolpod.NewTask(we.paths.VarRunArgo)
	require.NoError(t, task.Create())
	go we.Terminate()

	status, err := we.waitPoolMain(ctx, task)

	require.NoError(t, err)
	assert.Equal(t, poolpod.Withdrawn, status.State)
	assert.Equal(t, poolReleased, we.poolTask.stop)
}
