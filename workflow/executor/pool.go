package executor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
)

// This file is the worker container's side of a hot worker pool pod, for
// each task it claims (PoolRole): it prepares the task in directories of
// its own beneath the pool pod's shared ones (PoolTask), hands it to the
// looping main container through the poolpod file protocol, waits for main
// to run it, and collects its outputs through the task's own directory, as
// a pod's wait container does. Before and after each task it removes other
// tasks' files from the pool pod's shared directories and main's emptyDir
// volumes. Main's container, and so its writable layer (/tmp, $HOME and
// the like), outlives tasks: what a task writes there, the next task sees.

// poolTask is the state of a pool pod's task, which only the pool role's
// stages read and write.
type poolTask struct {
	// imageCommand is what main runs when the template's container has no
	// command (TaskConfig.Command).
	imageCommand []string
	// emptyDirs are where main mounts the pool pod's emptyDir volumes
	// (TaskConfig.EmptyDirs).
	emptyDirs []string
	// stop is why the task was stopped from outside main, if it was, which
	// decides what is reported for it.
	stop poolStop
	// noMainOutputs is set by the Run phase when main ended without staging
	// its outputs, because it died or failed the task before running the
	// command, so Collect skips the stages that read them.
	noMainOutputs bool
	// busy is set while main may be running the task: from when it is
	// offered until main is known to be done with it.
	busy bool
}

func newPoolTask(cfg TaskConfig) *poolTask {
	return &poolTask{imageCommand: cfg.Command, emptyDirs: cfg.EmptyDirs}
}

// runPoolTask runs the plan's Run phase, which hands the task to main and
// waits for main to run it, then, as PostMain does in a pod, its Collect
// phase on bgCtx. The stages that read main's outputs are skipped if main
// ended without staging them. An interrupted task, whose node has already
// ended, has only main's log archived, as a stopped pod's is; one released
// or whose claim was lost has nothing collected.
func (we *WorkflowExecutor) runPoolTask(ctx, bgCtx context.Context) error {
	plan := we.selectedPlan()
	we.runRecording(ctx, plan.Run, true)
	switch stop := we.poolTask.stop; {
	case stop.reportsOutputs():
		we.runRecording(bgCtx, plan.Collect, !we.poolTask.noMainOutputs)
	case stop.reportsLogs():
		we.runRecording(bgCtx, []Stage{reportPoolLogs}, false)
	}
	return we.HasError()
}

// reportPoolLogs archives main's log and reports it, unless there is none
// to archive.
func (we *WorkflowExecutor) reportPoolLogs(ctx context.Context) error {
	logs := we.SaveLogs(ctx)
	if len(logs) == 0 {
		return nil
	}
	return we.reportResult(ctx, wfv1.NodeResult{Outputs: &wfv1.Outputs{Artifacts: logs}})
}

// taskFailure is a failure of the task's own, such as main exiting non-zero:
// a pooled task failing so is Failed, as a pod whose main container fails.
// Any other failure is the worker's, so the task is Error, as a pod whose
// init, wait or supervisor container fails.
type taskFailure struct{ error }

func (f taskFailure) Unwrap() error { return f.error }

// clearPool empties main's emptyDir volumes and removes whatever other
// tasks left in the pool pod's shared directories, as a worker that
// restarted, or failed to remove a task's directories, leaves them, so the
// task sees no other task's files there. Main's writable layer isn't
// cleared. Main is between tasks whenever one is prepared, so nothing
// removed is in use.
func (we *WorkflowExecutor) clearPool() error {
	return errors.Join(we.wipeEmptyDirs(), clearPoolDirs(we.paths.taskPool().poolDirs(), filepath.Base(we.paths.VarRunArgo)))
}

// clearPoolForTask clears the pool before the task (clearPool). If it
// can't, the task, which has been claimed but not run, is released for
// another worker, and this worker stops: its restart clears the pool again
// before it claims anything, and exits again if it still can't, so the pod
// crash loops and the pool shows as degraded.
func (we *WorkflowExecutor) clearPoolForTask(ctx context.Context) error {
	err := we.clearPool()
	if err == nil {
		return nil
	}
	err = fmt.Errorf("failed to clear the pool: %w", err)
	we.recordPoolStop(ctx, poolReleased)
	we.stopWorker(ctx, err)
	return err
}

// afterPoolTask stops the worker if err, from removing the task's files or
// wiping main's emptyDir volumes after the task, says it failed, and returns
// err. The task is settled by then, but a failure leaves files the next
// task could see, so the worker stops before it claims another: its restart
// clears the pool (ClearPool).
func (we *WorkflowExecutor) afterPoolTask(ctx context.Context, err error) error {
	if err != nil {
		we.stopWorker(ctx, err)
	}
	return err
}

// ClearPool removes whatever tasks left in the shared directories of a pool
// pod whose paths are base's (PoolTask), keeping the pool directory's
// `current` and `main`. The worker runs it at start, once RecoverPool has
// settled what an earlier run left, before it claims a task. Main's
// emptyDir volumes are a task's to know of (TaskConfig.EmptyDirs), so each
// task's clearPool wipes them.
func ClearPool(base TaskPaths) error {
	return clearPoolDirs(base.poolShared().poolDirs(), "")
}

// clearPoolDirs removes everything in a pool pod's shared directories, the
// first of which is the pool directory, but the entries of the task named
// keep and the pool directory's `current` and `main`: main reads the one to
// find its next task, and holds the other while it is up.
func clearPoolDirs(shared []string, keep string) error {
	var errs []error
	for i, dir := range shared {
		isPoolDir := i == 0
		errs = append(errs, removeEntries(dir, func(entry string) bool {
			return entry == keep || (isPoolDir && (entry == poolpod.CurrentFile || entry == poolpod.MainFile))
		}))
	}
	return errors.Join(errs...)
}

// removeEntries removes everything in dir but the entries keep keeps.
func removeEntries(dir string, keep func(name string) bool) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var errs []error
	for _, e := range entries {
		if !keep(e.Name()) {
			errs = append(errs, os.RemoveAll(filepath.Join(dir, e.Name())))
		}
	}
	return errors.Join(errs...)
}

// createPoolTask creates the task's own directories. The task's directory
// in the pool directory and its input and staging directories are written
// by main too, which may run as another user.
func (we *WorkflowExecutor) createPoolTask() error {
	if err := poolpod.NewTask(we.paths.VarRunArgo).Create(); err != nil {
		return fmt.Errorf("failed to create %s: %w", we.paths.VarRunArgo, err)
	}
	for _, dir := range []string{we.paths.InputArtifacts, we.paths.Staging} {
		if err := poolpod.MakeSharedDir(dir); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}
	if err := os.MkdirAll(we.paths.Scratch, 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", we.paths.Scratch, err)
	}
	return nil
}

// poolTaskSpec is what main is told about the task: the command and env a
// pod's main container would have been given for it, beyond what the pool
// pod's main container has for every task.
func (we *WorkflowExecutor) poolTaskSpec() (poolpod.TaskSpec, error) {
	c := mainContainer(&we.Template)
	if c == nil {
		return poolpod.TaskSpec{}, fmt.Errorf("a pool runs only container and script templates, not a %s template", strings.ToLower(string(we.Template.GetType())))
	}
	spec := poolpod.TaskSpec{
		NodeID:              we.nodeID,
		Template:            &we.Template,
		Command:             slices.Clone(c.Command),
		Args:                common.MainArgs(&we.Template, we.paths.ScriptSource()),
		IncludeScriptOutput: we.IncludeScriptOutput,
	}
	if len(c.Command) == 0 {
		spec.ImageCommand = slices.Clone(we.poolTask.imageCommand)
	}
	// valueFrom entries are the pool pod's: the kubelet resolved them into
	// main's own env when the pod started.
	for _, e := range c.Env {
		if e.ValueFrom == nil {
			spec.Env = append(spec.Env, poolpod.EnvVar{Name: e.Name, Value: e.Value})
		}
	}
	// After the template's own, as a pod appends them to main's.
	for _, kv := range we.mainEnv {
		name, value, _ := strings.Cut(kv, "=")
		spec.Env = append(spec.Env, poolpod.EnvVar{Name: name, Value: value})
	}
	return spec, nil
}

// writePoolTask writes the task file main runs the task from.
func (we *WorkflowExecutor) writePoolTask() error {
	spec, err := we.poolTaskSpec()
	if err != nil {
		return err
	}
	return poolpod.NewTask(we.paths.VarRunArgo).WriteSpec(spec)
}

// runPoolMain hands the prepared task to main and waits until main is done
// with it, reporting progress meanwhile and stopping it from outside main
// when it must stop (waitPoolMain). How main ended is the task's outcome
// (mainEnded), unless it was stopped for a reason that decides what is
// reported. A task that had to stop while it was prepared is never offered.
// Until main is known to be done with the task, the task is busy: nothing
// of it is removed, and its result is left for the worker's restart to
// fail, so the controller never finishes a node whose task may still be
// running.
func (we *WorkflowExecutor) runPoolMain(ctx context.Context) error {
	task := poolpod.NewTask(we.paths.VarRunArgo)
	if stop := we.pendingStop(); stop != poolNotStopped {
		we.poolTask.noMainOutputs = true
		we.recordPoolStop(ctx, stop)
		return nil
	}
	we.poolTask.busy = true
	if err := poolpod.NewPool(we.paths.taskPool().VarRunArgo).Offer(task); err != nil {
		we.poolTask.noMainOutputs = true
		// current is replaced atomically, so main has most likely not seen
		// the task; withdrawing it makes sure.
		if withdrew, withdrawErr := task.Withdraw(); withdrawErr == nil && withdrew {
			we.poolTask.busy = false
		}
		return fmt.Errorf("failed to offer the task to main: %w", err)
	}
	stopProgress := we.startProgressMonitor(ctx)
	status, err := we.waitPoolMain(ctx, task)
	stopProgress()
	if err != nil {
		we.poolTask.noMainOutputs = true
		return fmt.Errorf("failed to wait for main: %w", err)
	}
	we.poolTask.busy = false
	return we.mainEnded(ctx, status)
}

// mainEnded records how main ended the task, as the controller reads a
// pod's main container: its exit code is the task's, and a non-zero one, or
// main dying, fails the task. Main stages the outputs once the command has
// exited, so they are there to collect unless main died or failed the task
// itself, before or instead of running the command. A task stopped from
// outside main because it was released, interrupted or its claim lost is
// reported for that, whatever its exit code, such as main's 143 for a task
// stopped before it started; one stopped at its deadline has already failed
// for it.
func (we *WorkflowExecutor) mainEnded(ctx context.Context, status poolpod.Status) error {
	// Reported on a context the task's end can't cancel.
	reportCtx := context.WithoutCancel(ctx)
	if !we.poolTask.stop.reportsOutputs() || (status.State == poolpod.Withdrawn && we.poolTask.stop != poolNotStopped) {
		we.poolTask.noMainOutputs = true
		return nil
	}
	switch status.State {
	case poolpod.Exited:
		exitCode := strconv.Itoa(status.ExitCode)
		we.capturedOutputs().ExitCode = &exitCode
		if status.Message != "" {
			we.poolTask.noMainOutputs = true
		}
		if status.ExitCode != 0 || status.Message != "" {
			we.sink.Fail(reportCtx, taskFailure{mainExitedError(status)})
		}
		return nil
	case poolpod.Died:
		we.poolTask.noMainOutputs = true
		we.sink.Fail(reportCtx, taskFailure{fmt.Errorf("%s: the container exited before the task ended", common.MainContainerName)})
		return nil
	default:
		// The worker withdraws a task only to stop it, and it didn't.
		we.poolTask.noMainOutputs = true
		return fmt.Errorf("main ended the task as %s", status.State)
	}
}

// mainExitedError is the failure of a task main exited non-zero or failed
// itself, worded as the controller words a pod's main container's, whose
// reason is Error.
func mainExitedError(status poolpod.Status) error {
	return errors.New(common.ContainerFailureMessage(common.MainContainerName, "Error", status.ExitCode, status.Message))
}

// completePoolTask completes the task's claim with everything reported, in
// one write, or gives a released task back. A claim found lost isn't the
// worker's to report: the task's node has moved on, and nothing is written.
// An interrupted task's claim is completed to confirm the task stopped,
// with only main's log. If main may still be running the task, or the
// claim couldn't be completed, the worker stops: the claim is left
// incomplete, and the worker's restart stops main's task before it fails
// the claim (RecoverPool).
func (we *WorkflowExecutor) completePoolTask(ctx context.Context) error {
	logger := logging.RequireLoggerFromContext(ctx)
	if we.poolTask.busy {
		err := errors.New("main may still be running the task")
		we.stopWorker(ctx, err)
		return err
	}
	switch we.poolTask.stop {
	case poolLost:
		logger.Info(ctx, "The task's claim was lost, so its result is not reported")
		return nil
	case poolReleased:
		if err := we.releasePoolTask(ctx); err != nil {
			err = fmt.Errorf("failed to release the task: %w", err)
			we.stopWorker(ctx, err)
			return err
		}
		logger.Info(ctx, "Released the task")
		return nil
	case poolInterrupted:
		we.sink.Fail(ctx, taskFailure{errors.New("the task was interrupted")})
	}
	if err := UnlessClaimLost(ctx, we.sink.Finalize(ctx), "The task's claim was lost, so its result is not reported"); err != nil {
		err = fmt.Errorf("failed to report the task's result: %w", err)
		we.stopWorker(ctx, err)
		return err
	}
	return nil
}

// removePoolTask removes the task's own directories, unless main may still
// be running the task.
func (we *WorkflowExecutor) removePoolTask() error {
	if we.poolTask.busy {
		return nil
	}
	var errs []error
	for _, dir := range we.paths.poolDirs() {
		if err := os.RemoveAll(dir); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove %s: %w", dir, err))
		}
	}
	return errors.Join(errs...)
}

// wipeEmptyDirs empties main's emptyDir volumes, as the worker sees them
// through MainFS, unless main may still be running the task: no task sees
// another's files in them. The mount points stay, as do other volumes
// mounted beneath them and the directories leading to those: a PVC's or
// hostPath's contents outlive tasks, as they outlive pods.
func (we *WorkflowExecutor) wipeEmptyDirs() error {
	if we.poolTask.busy {
		return nil
	}
	// main's mounts of the directories it shares with the worker are
	// mirrored under MainFS too.
	mounts := []string{
		filepath.Join(we.paths.MainFS, common.ExecutorArtifactBaseDir),
		filepath.Join(we.paths.MainFS, common.ExecutorStagingEmptyDir),
	}
	for _, m := range we.Template.GetVolumeMounts() {
		mounts = append(mounts, filepath.Join(we.paths.MainFS, m.MountPath))
	}
	var errs []error
	for _, dir := range we.poolTask.emptyDirs {
		if err := emptyDirExcept(filepath.Join(we.paths.MainFS, dir), mounts); err != nil {
			errs = append(errs, fmt.Errorf("failed to empty volume %s: %w", dir, err))
		}
	}
	return errors.Join(errs...)
}

// emptyDirExcept removes everything in dir except mounts, and the
// directories leading to them, whose contents it empties in turn.
func emptyDirExcept(dir string, mounts []string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var errs []error
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		switch {
		case slices.Contains(mounts, path):
		case slices.ContainsFunc(mounts, func(m string) bool {
			_, within := common.RelWithin(path, m)
			return within
		}):
			if e.IsDir() {
				errs = append(errs, emptyDirExcept(path, mounts))
			}
		default:
			errs = append(errs, os.RemoveAll(path))
		}
	}
	return errors.Join(errs...)
}

// stopWorker makes the worker process stop taking work and exit with cause,
// once this task's flow is over.
func (we *WorkflowExecutor) stopWorker(ctx context.Context, cause error) {
	logging.RequireLoggerFromContext(ctx).WithError(cause).Error(ctx, "Stopping the worker: only its restart can settle what it leaves")
	if we.pool.stopWorker != nil {
		we.pool.stopWorker(cause)
	}
}

// RecoverPool ends what an earlier run of this pool pod's worker container
// left, before the worker takes new work, so the task it had is never run
// again: the task main was offered is withdrawn if main hasn't taken it, or
// else stopped as a lost claim and waited for, then every claim the pod
// still holds incomplete is completed as Error with message. pool is the
// pool pod's pool directory.
func (p *Process) RecoverPool(ctx context.Context, pool poolpod.Pool, consumer dispatch.Consumer, message string) error {
	if err := settleCurrent(ctx, pool); err != nil {
		return err
	}
	var failed []dispatch.Key
	err := p.resultRetry().retryDoing(ctx, p.taskResultRetryTimeout, "failing the claims", func(ctx context.Context) error {
		keys, err := consumer.FailOwn(ctx, message)
		failed = append(failed, keys...)
		return err
	})
	for _, key := range failed {
		logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"namespace": key.Namespace, "nodeID": key.NodeID}).
			Warn(ctx, "Failed the claim of a task an earlier run of the worker had")
	}
	if err != nil {
		return fmt.Errorf("failed to fail the claims an earlier run of the worker left: %w", err)
	}
	return nil
}

// settleCurrent makes sure main never starts the task `current` names, and
// is done with it, if it has started it.
func settleCurrent(ctx context.Context, pool poolpod.Pool) error {
	task, ok, err := pool.Current()
	if err != nil || !ok {
		return err
	}
	logger := logging.RequireLoggerFromContext(ctx).WithField("task", task.Dir())
	if _, statErr := os.Stat(task.Dir()); errors.Is(statErr, fs.ErrNotExist) {
		// Removed once it was done with.
		return nil
	}
	// Stopping is harmless if main is done with it already.
	withdrew, _, err := task.WithdrawOrStop(poolpod.StopLost)
	if err != nil {
		return fmt.Errorf("task %s: %w", task.Dir(), err)
	}
	if withdrew {
		logger.Info(ctx, "Withdrew the task an earlier run of the worker offered")
		return nil
	}
	status, err := task.WaitMain(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for main to stop task %s: %w", task.Dir(), err)
	}
	logger.WithField("state", status.State).Info(ctx, "Main is done with the task an earlier run of the worker had")
	return nil
}

// mainContainer is the container a container or script template runs.
func mainContainer(tmpl *wfv1.Template) *apiv1.Container {
	if tmpl.Script != nil {
		return &tmpl.Script.Container
	}
	return tmpl.Container
}

// poolTaskFailurePhase is the phase of a pooled task that failed with err.
func poolTaskFailurePhase(err error) wfv1.NodePhase {
	if errors.As(err, new(taskFailure)) {
		return wfv1.NodeFailed
	}
	return wfv1.NodeError
}
