package executor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
)

// activeDeadlineExceededMessage is the kubelet's message for a pod it
// stopped at its activeDeadlineSeconds, which the controller makes the
// pod's node's.
const activeDeadlineExceededMessage = "Pod was active on the node longer than the specified deadline"

// This file is how a pool pod's task is stopped from outside main, as the
// kubelet and the controller stop a pod's: at its deadline, when the
// controller interrupts it or its claim is lost, and when the worker is
// terminated, which releases the task for another worker to run.

// poolStop is why a pool pod's task was stopped from outside main, which
// decides what is reported for it.
type poolStop int

const (
	// poolNotStopped: main ran the task to its end, which is the outcome.
	poolNotStopped poolStop = iota
	// poolDeadline: the task passed its deadline, and fails as a pod's does.
	// What it produced is still collected.
	poolDeadline
	// poolReleased: the worker was terminated, so it gives the task back:
	// the claim is deleted and nothing is reported.
	poolReleased
	// poolInterrupted: the controller ended the node. The claim is
	// completed to confirm the task stopped, with main's log, archived as a
	// stopped pod's is, and nothing else the task produced.
	poolInterrupted
	// poolLost: the claim was lost, so nothing is reported.
	poolLost
)

// overrides reports whether s replaces prev as why the task stopped: the
// first reason stands, but for the controller's interrupt, which overrides
// a deadline or a release, and a lost claim, which overrides any other.
func (s poolStop) overrides(prev poolStop) bool {
	switch {
	case prev == poolNotStopped:
		return true
	case s == poolLost:
		return prev != poolLost
	case s == poolInterrupted:
		return prev == poolDeadline || prev == poolReleased
	}
	return false
}

// reportsOutputs reports whether what main produced is collected and
// reported.
func (s poolStop) reportsOutputs() bool {
	return s == poolNotStopped || s == poolDeadline
}

// reportsLogs reports whether main's log is archived and reported, which
// it is whenever the task's node takes anything from its result.
func (s poolStop) reportsLogs() bool {
	return s.reportsOutputs() || s == poolInterrupted
}

// stopReason is what main is told when the task is stopped for s.
func (s poolStop) stopReason() poolpod.StopReason {
	switch s {
	case poolDeadline:
		return poolpod.StopDeadline
	case poolReleased:
		return poolpod.StopRelease
	case poolInterrupted:
		return poolpod.StopInterrupt
	}
	return poolpod.StopLost
}

func (s poolStop) String() string {
	switch s {
	case poolNotStopped:
		return "not stopped"
	case poolDeadline:
		return "deadline"
	case poolReleased:
		return "released"
	case poolInterrupted:
		return "interrupted"
	}
	return "lost"
}

// poolProcess is a pool pod's worker's process-wide state: how a task stops
// the worker, and when the worker was terminated.
type poolProcess struct {
	// stopWorker is ProcessConfig.StopWorker.
	stopWorker func(cause error)
	// releaseReserve is ProcessConfig.ReleaseReserve.
	releaseReserve time.Duration
	// terminatedCh is closed by Terminate, at terminatedAt.
	terminatedCh  chan struct{}
	terminateOnce sync.Once
	terminatedAt  time.Time
}

// Terminate tells the process's tasks the worker was terminated, as by
// SIGTERM: a pool pod's task main hasn't taken is released at once, and
// one main is still running at the end of the grace period, less the
// release reserve, is stopped and released. Calling it again does nothing.
func (p *Process) Terminate() {
	p.pool.terminateOnce.Do(func() {
		p.pool.terminatedAt = time.Now()
		close(p.pool.terminatedCh)
	})
}

// terminatedChan is closed once the process is terminated.
func (p *Process) terminatedChan() <-chan struct{} {
	return p.pool.terminatedCh
}

// terminated reports whether the process was terminated.
func (p *Process) terminated() bool {
	select {
	case <-p.pool.terminatedCh:
		return true
	default:
		return false
	}
}

// releaseAt is when a terminated process releases the task main is still
// running: the end of the grace period less the release reserve. Call it
// only once terminated.
func (p *Process) releaseAt() time.Time {
	return p.pool.terminatedAt.Add(p.terminationGracePeriod - p.pool.releaseReserve)
}

// stoppableSink is a sink that says when its task must stop from outside
// the process (PooledSink).
type stoppableSink interface {
	Stopped() <-chan struct{}
	StopReason() dispatch.StopReason
}

// releasableSink is a sink whose task can be given back (PooledSink).
type releasableSink interface {
	Release(ctx context.Context) error
}

// sinkStopped is closed when the controller interrupts the task or its
// claim is lost, or nil if the sink can't tell.
func (we *WorkflowExecutor) sinkStopped() <-chan struct{} {
	if s, ok := we.sink.(stoppableSink); ok {
		return s.Stopped()
	}
	return nil
}

// sinkStop is why the sink says the task must stop, if it does.
func (we *WorkflowExecutor) sinkStop() poolStop {
	s, ok := we.sink.(stoppableSink)
	if !ok {
		return poolNotStopped
	}
	switch s.StopReason() {
	case dispatch.StopInterrupted:
		return poolInterrupted
	case dispatch.StopLost:
		return poolLost
	}
	return poolNotStopped
}

// deadlineTimer fires at the task's deadline; it is nil if the task has
// none. Call stop when done with it.
func (we *WorkflowExecutor) deadlineTimer() (c <-chan time.Time, stop func()) {
	if we.Deadline.IsZero() {
		return nil, func() {}
	}
	t := time.NewTimer(time.Until(we.Deadline))
	return t.C, func() { t.Stop() }
}

// pendingStop is why the task must stop before main has it, if it must:
// a lost or interrupted claim, a deadline passed or, failing those, the
// worker terminated, which releases a task main hasn't taken at once.
func (we *WorkflowExecutor) pendingStop() poolStop {
	if stop := we.sinkStop(); stop != poolNotStopped {
		return stop
	}
	if !we.Deadline.IsZero() && !time.Now().Before(we.Deadline) {
		return poolDeadline
	}
	if we.terminated() {
		return poolReleased
	}
	return poolNotStopped
}

// poolPrepareContext is ctx, also cancelled once the task must stop before
// main has it (pendingStop), so preparing it, such as loading a large input
// artifact, doesn't hold the task up.
func (we *WorkflowExecutor) poolPrepareContext(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)
	deadline, stopDeadline := we.deadlineTimer()
	stopped, terminated := we.sinkStopped(), we.terminatedChan()
	go func() {
		defer stopDeadline()
		select {
		case <-ctx.Done():
			return
		case <-deadline:
		case <-stopped:
		case <-terminated:
		}
		cancel()
	}()
	return ctx, cancel
}

// stopPoolTask records that the task must stop for stop
// (recordPoolStop), then makes sure main never starts it, if it hasn't, or
// else tells main to stop it.
func (we *WorkflowExecutor) stopPoolTask(ctx context.Context, task poolpod.Task, stop poolStop) {
	logger := logging.RequireLoggerFromContext(ctx).WithField("stop", stop.String())
	we.recordPoolStop(ctx, stop)
	withdrew, reason, err := task.WithdrawOrStop(stop.stopReason())
	switch {
	case err != nil:
		// Main runs the task to its end.
		logger.WithError(err).Error(ctx, "Failed to stop the task")
	case withdrew:
		logger.Info(ctx, "Withdrew the task before main took it")
	default:
		logger.WithField("reason", reason).Info(ctx, "Stopped the task")
	}
}

// recordPoolStop records that the task must stop for stop, unless it
// already stopped for a reason that stop doesn't override. A task stopped at
// its deadline fails with a pod's message for it (deadlineMessage).
func (we *WorkflowExecutor) recordPoolStop(ctx context.Context, stop poolStop) {
	if stop.overrides(we.poolTask.stop) {
		we.poolTask.stop = stop
		if stop == poolDeadline {
			we.sink.Fail(context.WithoutCancel(ctx), taskFailure{errors.New(we.deadlineMessage())})
		}
	}
}

// deadlineMessage is a pod's message for passing the task's deadline: the
// kubelet's when it is the task's activeDeadlineSeconds that ended, which
// the kubelet enforces for a pod, and otherwise the executor's, which
// enforces the absolute deadline in a pod too.
func (we *WorkflowExecutor) deadlineMessage() string {
	if !we.activeDeadline.IsZero() && we.activeDeadline.Equal(we.Deadline) {
		return activeDeadlineExceededMessage
	}
	return common.DeadlineExceededMessage
}

// waitPoolMain waits until main is done with the task, stopping it from
// outside main as the reasons to arise: its deadline, the controller's
// interrupt or a lost claim, and the worker's termination. Termination
// releases the task at once if main hasn't taken it, and otherwise only if
// main is still running it once the release is due, at the end of the
// pod's grace period less the release reserve: a task done by then is
// collected and reported as usual.
func (we *WorkflowExecutor) waitPoolMain(ctx context.Context, task poolpod.Task) (poolpod.Status, error) {
	type waited struct {
		status poolpod.Status
		err    error
	}
	waitCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan waited, 1)
	go func() {
		status, err := task.WaitMain(waitCtx)
		done <- waited{status, err}
	}()
	deadline, stopDeadline := we.deadlineTimer()
	defer stopDeadline()
	stopped, terminated := we.sinkStopped(), we.terminatedChan()
	// Started once the worker is terminated with main running the task.
	var releaseTimer *time.Timer
	defer func() {
		if releaseTimer != nil {
			releaseTimer.Stop()
		}
	}()
	var release <-chan time.Time
	for {
		select {
		case w := <-done:
			return w.status, w.err
		case <-deadline:
			deadline = nil
			we.stopPoolTask(ctx, task, poolDeadline)
		case <-stopped:
			stopped = nil
			we.stopPoolTask(ctx, task, we.sinkStop())
		case <-terminated:
			terminated = nil
			if withdrew, err := task.Withdraw(); err == nil && withdrew {
				logging.RequireLoggerFromContext(ctx).Info(ctx, "Withdrew the task before main took it")
				we.recordPoolStop(ctx, poolReleased)
				continue
			}
			releaseTimer = time.NewTimer(time.Until(we.releaseAt()))
			release = releaseTimer.C
		case <-release:
			release = nil
			we.stopPoolTask(ctx, task, poolReleased)
		}
	}
}

// releasePoolTask gives the task back, so another worker can claim it.
func (we *WorkflowExecutor) releasePoolTask(ctx context.Context) error {
	s, ok := we.sink.(releasableSink)
	if !ok {
		return errors.New("the task's result can't be released")
	}
	return UnlessClaimLost(ctx, s.Release(ctx), "The released task's claim was lost already")
}
