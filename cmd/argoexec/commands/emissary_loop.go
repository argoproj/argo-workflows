package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"go.opentelemetry.io/otel/propagation"
	"k8s.io/apimachinery/pkg/util/wait"

	argoexecexecutor "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoerrors "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/poolpod"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// This file is `argoexec emissary --loop`, the main container of a hot
// worker pool pod. Where the one-task emissary runs the command it is given
// once and exits, the loop runs one task after another in the same
// container, each handed over by the pod's worker container through the
// poolpod file protocol, and makes sure no task sees another's files or
// processes: each task's input links and declared outputs exist only while
// it runs, and every process it started is killed before the next begins.

const (
	// loopExitCodeMainFailed is the exit code of a task main failed itself,
	// with a message saying why, as the emissary's exit code when the
	// command produced none.
	loopExitCodeMainFailed = 64
	// loopExitCodeStopped is the exit code of a task stopped before its
	// command started: what a command killed by SIGTERM exits with.
	loopExitCodeStopped = 128 + 15
	// loopReapInterval is how often main reaps between tasks.
	loopReapInterval = time.Second
	// loopExitSweepTimeout is how long main keeps killing what a finished
	// task left before it gives up: a process SIGKILL can't end, such as
	// one in uninterruptible sleep on a hung volume, makes main exit and
	// its container restart rather than hang on the task.
	loopExitSweepTimeout = 30 * time.Second
)

// loopConfig is everything `emissary --loop` reads from its environment,
// passed in so the loop is testable without env or package-level state.
type loopConfig struct {
	// Base is the pool pod's shared directories: PodPaths in a pool pod.
	// The pool directory and each task's paths are beneath them
	// (TaskPaths.PoolTask), and their contents are never a task's own.
	Base wfexecutor.TaskPaths
	// ReleaseReserve is the time the worker keeps back at the end of the
	// pod's grace period to release its task. A stopped task's processes
	// get half of it between SIGTERM and SIGKILL.
	ReleaseReserve time.Duration
	// Environ is main's own environment. Every task's command inherits it,
	// under the task's own env, and the task's $(VAR) references resolve
	// against it.
	Environ []string
	// Driver runs each task's command.
	Driver maindriver.MainDriver
	// Tracer starts each task's main container span.
	Tracer *tracing.Tracing
	// Namespace is the pool pod's namespace, an attribute of each task's
	// span. Empty if unknown.
	Namespace string
}

// runLoopCommand is `emissary --loop`: runLoop, configured from the
// environment of a pool pod's main container.
func runLoopCommand(ctx context.Context) error {
	reserve, err := argoexecexecutor.ReleaseReserve()
	if err != nil {
		return err
	}
	tracer, err := tracing.New(ctx, `argoexec`)
	if err != nil {
		return fmt.Errorf("initialize tracing: %w", err)
	}
	defer func() {
		if err := tracer.Shutdown(context.WithoutCancel(ctx)); err != nil {
			logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "Failed to shutdown tracing")
		}
	}()
	namespace, _ := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	return runLoop(ctx, loopConfig{
		Base:           wfexecutor.PodPaths(),
		ReleaseReserve: reserve,
		Environ:        os.Environ(),
		Driver:         maindriver.Container{},
		Tracer:         tracer,
		Namespace:      string(namespace),
	})
}

// runLoop runs the pool's tasks one after another until ctx ends, which is
// SIGTERM. With no task taken it returns at once; a task already taken
// runs to its end, unless the worker stops it, and runLoop returns after
// it. It returns an error only when it can't go on, and then main exits
// and its container restarts, which also kills any process a task left
// that main couldn't.
func runLoop(ctx context.Context, cfg loopConfig) error {
	logger := logging.RequireLoggerFromContext(ctx)
	terminated := func() error {
		logger.Info(ctx, "terminated with no task taken, exiting")
		return nil
	}
	if runtime.GOOS != "linux" {
		return fmt.Errorf("emissary --loop runs only on Linux: %w", errors.ErrUnsupported)
	}
	// A process a task daemonises must stay main's descendant, so that
	// main can find and kill it once the task is over.
	if err := poolpod.BecomeSubreaper(); err != nil {
		return err
	}
	// The worker may run as another user: it reads what main stages and
	// main's message and exit code.
	osspecific.AllowGrantingAccessToEveryone()
	pool := poolpod.NewPool(cfg.Base.PoolDir())
	if err := pool.Init(); err != nil {
		return fmt.Errorf("create the pool directory: %w", err)
	}
	// Held until main exits: the worker claims tasks only while it is.
	mainLock, err := pool.HoldMain(ctx)
	if ctx.Err() != nil {
		return terminated()
	}
	if err != nil {
		return fmt.Errorf("take main's liveness lock: %w", err)
	}
	defer func() { _ = mainLock.Close() }()
	reaper := startReaper(ctx)
	defer func() { reaper.stop(ctx) }()

	var prev poolpod.Task
	for {
		task, err := pool.WaitNext(ctx, prev)
		if ctx.Err() != nil {
			return terminated()
		}
		if err != nil {
			return fmt.Errorf("wait for the next task: %w", err)
		}
		prev = task
		taskLogger := logger.WithField("task", task.Dir())
		run, err := take(logging.WithLogger(ctx, taskLogger), task)
		if err != nil && ctx.Err() != nil {
			return terminated()
		}
		if err != nil {
			return err
		}
		if run == nil {
			continue
		}
		// The task's command is waited for by reaping, which must be the
		// only reaping going on.
		reaper.stop(ctx)
		// SIGTERM doesn't stop a task: only the worker does, through stop.
		taskCtx := logging.WithLogger(context.WithoutCancel(ctx), taskLogger)
		code, message, sweepErr := cfg.runTask(taskCtx, run)
		if message != "" {
			taskLogger.WithField("message", message).Error(ctx, "main failed the task")
		}
		if err := run.Finish(code, message); err != nil {
			// The lock is released anyway, so the worker sees main as
			// having died in the task.
			taskLogger.WithError(err).Error(ctx, "failed to write the task's exit code")
		}
		taskLogger.WithField("exitCode", code).Info(ctx, "task done")
		if sweepErr != nil {
			return fmt.Errorf("a process task %s started is still running: %w", task.Dir(), sweepErr)
		}
		reaper = startReaper(ctx)
		if ctx.Err() != nil {
			logger.Info(ctx, "terminated during the task, exiting now it is done")
			return nil
		}
	}
}

// take takes the task for main, and returns nil with a nil error when main
// must skip it. A task main took is run even if ctx ended as main took it:
// SIGTERM doesn't stop a task, only the worker does.
func take(ctx context.Context, task poolpod.Task) (*poolpod.Run, error) {
	run, err := task.Take(ctx)
	switch {
	case err == nil:
		return run, nil
	case errors.Is(err, poolpod.ErrNotOffered), errors.Is(err, poolpod.ErrWithdrawn), errors.Is(err, poolpod.ErrTaken):
		// A restarted main finds the task its predecessor took as taken,
		// and so never runs it again.
		logging.RequireLoggerFromContext(ctx).WithError(err).Info(ctx, "skipping task")
		return nil, nil
	}
	return nil, fmt.Errorf("take task %s: %w", task.Dir(), err)
}

// runTask runs one task that main has taken, and returns its exit code and
// main's message, which is empty unless main failed the task itself. A
// non-nil error means a process the task started couldn't be killed, or
// was still alive loopExitSweepTimeout after the command exited, so main
// must not run another task; the task's files are then left, and its
// outputs not staged, as that process may still be writing them.
func (cfg loopConfig) runTask(ctx context.Context, run *poolpod.Run) (code int, message string, sweepErr error) {
	logger := logging.RequireLoggerFromContext(ctx)
	failed := func(err error) (int, string, error) {
		return loopExitCodeMainFailed, err.Error(), nil
	}
	// The worker may have stopped the task before main took it.
	if reason, stopped, err := run.StopRequested(); err != nil {
		return failed(err)
	} else if stopped {
		logger.WithField("reason", reason).Info(ctx, "task stopped before it started")
		return loopExitCodeStopped, "", nil
	}
	task := run.Task()
	paths, err := cfg.Base.PoolTask(filepath.Base(task.Dir()))
	if err != nil {
		return failed(err)
	}
	if filepath.Clean(task.Dir()) != paths.VarRunArgo {
		return failed(fmt.Errorf("task directory %s is not in the pool directory %s", task.Dir(), filepath.Dir(paths.VarRunArgo)))
	}
	spec, err := task.Spec()
	if err != nil {
		return failed(err)
	}
	// The task's span, as a pod's main container's: a child of the task's
	// own trace context, and the parent the command sees.
	ctx, span := cfg.Tracer.StartRunMainContainer(withTaskTraceContext(ctx, spec.Env), taskEnv(spec.Env, common.EnvVarWorkflowName), cfg.Namespace)
	defer span.End()
	command, env := spec.Resolve(cfg.Environ)
	if command, err = onTaskPath(command, env); err != nil {
		return failed(err)
	}
	env = injectTraceParent(ctx, env)
	// In the task's directory, which the worker removes after the task,
	// rather than /tmp, where they would outlive it.
	if command, err = maindriver.OffloadLargeArgs(ctx, command, paths.VarRunArgo); err != nil {
		return failed(err)
	}
	workDir := paths.ContainerDir(common.MainContainerName)
	if err = poolpod.MakeSharedDir(workDir); err != nil {
		return failed(fmt.Errorf("create %s: %w", workDir, err))
	}

	iso := poolpod.NewIsolation(cfg.volumes(spec.Template))
	// After the command's processes are gone and its outputs are staged:
	// Driver.Run stages them before it returns.
	defer func() {
		if sweepErr != nil {
			// What a live process may still be writing; main exits, and its
			// container's restart takes them.
			logger.Info(ctx, "leaving the task's files, as a process it started is still running")
			return
		}
		if cleanupErr := iso.Cleanup(); cleanupErr != nil {
			// A path left in place fails the next task that uses it, so
			// no task sees another's.
			logger.WithError(cleanupErr).Error(ctx, "failed to remove the task's files")
		}
	}()
	if err = isolate(iso, spec.Template, paths.InputArtifacts); err != nil {
		return failed(err)
	}

	stopWatch := func() {}
	mainTask := maindriver.Task{
		NodeID:              spec.NodeID,
		Template:            spec.Template,
		Command:             command,
		Env:                 env,
		IncludeScriptOutput: spec.IncludeScriptOutput,
		WorkDir:             workDir,
		OnStart: func(int) {
			stopWatch = cfg.watchStop(ctx, run)
		},
		// Before the outputs are staged, so nothing the task started can
		// still be writing them, as when a pod's main container exits.
		OnExit: func() {
			stopWatch()
			sweepCtx, cancel := context.WithTimeout(ctx, loopExitSweepTimeout)
			defer cancel()
			sweepErr = poolpod.Sweep(sweepCtx, 0)
		},
	}
	stager := sweptStager{
		OutputStager: maindriver.PodStager{VarRunArgo: paths.VarRunArgo, ContainerName: common.MainContainerName, Template: spec.Template, SharedDirs: true},
		sweepErr:     &sweepErr,
	}
	code, err = cfg.Driver.Run(ctx, mainTask, stager)
	var exited argoerrors.Exited
	if err != nil && !errors.As(err, &exited) {
		// Not the command's own exit: it couldn't start, or its outputs
		// couldn't be staged. Never report that as success.
		if code == 0 {
			code = loopExitCodeMainFailed
		}
		return code, err.Error(), sweepErr
	}
	return code, "", sweepErr
}

// sweptStager stages a task's outputs only once the sweep after its command
// has killed every process it started, so none can still be writing them.
type sweptStager struct {
	maindriver.OutputStager
	sweepErr *error
}

func (s sweptStager) Stage(ctx context.Context, nodeID string, out maindriver.Output) error {
	if *s.sweepErr != nil {
		return fmt.Errorf("not staged, as a process the task started is still running: %w", *s.sweepErr)
	}
	return s.OutputStager.Stage(ctx, nodeID, out)
}

// onTaskPath returns command with a bare command name found on the PATH in
// env, the task's, as the container runtime finds a container's command on
// its env's PATH; Container.Run would search main's own PATH, the pool
// pod's. A command naming a directory is left for Container.Run.
func onTaskPath(command, env []string) ([]string, error) {
	if len(command) == 0 || strings.ContainsRune(command[0], filepath.Separator) {
		return command, nil
	}
	var path string
	for _, e := range slices.Backward(env) {
		if v, ok := strings.CutPrefix(e, "PATH="); ok {
			path = v
			break
		}
	}
	for _, dir := range filepath.SplitList(path) {
		if !filepath.IsAbs(dir) {
			continue
		}
		name := filepath.Join(dir, command[0])
		if fi, err := os.Stat(name); err == nil && fi.Mode().IsRegular() && fi.Mode().Perm()&0o111 != 0 {
			return append([]string{name}, command[1:]...), nil
		}
	}
	return nil, fmt.Errorf("command %q not found on the task's PATH %q", command[0], path)
}

// withTaskTraceContext returns ctx continuing the trace context in a task's
// env, its TRACEPARENT and TRACESTATE, or ctx unchanged when it has none, as
// a pod's executor reads its own.
func withTaskTraceContext(ctx context.Context, env []poolpod.EnvVar) context.Context {
	carrier := tracing.TraceContextFromEnv(taskEnvLookup(env))
	if carrier == nil {
		return ctx
	}
	return propagation.TraceContext{}.Extract(ctx, propagation.MapCarrier(carrier))
}

// taskEnv is the value of the last of a task's env entries called name, as
// written, or empty.
func taskEnv(env []poolpod.EnvVar, name string) string {
	value, _ := taskEnvLookup(env)(name)
	return value
}

// taskEnvLookup looks up a task's env entries as os.LookupEnv looks up a
// process's: the last entry called name wins.
func taskEnvLookup(env []poolpod.EnvVar) func(string) (string, bool) {
	return func(name string) (string, bool) {
		for _, e := range slices.Backward(env) {
			if e.Name == name {
				return e.Value, true
			}
		}
		return "", false
	}
}

// volumes are the mount paths in main whose contents are never a task's
// own: the template's volume mounts and the directories main shares with
// the worker. The controller's /tmp mount is the worker's, not main's, so
// main's /tmp is the image's, cleaned up like any other path.
func (cfg loopConfig) volumes(tmpl *wfv1.Template) []string {
	volumes := []string{cfg.Base.VarRunArgo, cfg.Base.InputArtifacts, cfg.Base.Staging}
	for _, m := range tmpl.GetVolumeMounts() {
		volumes = append(volumes, m.MountPath)
	}
	return volumes
}

// isolate reserves the task's declared output paths and links its input
// artifacts into place from inputsDir. A path that already exists outside a
// volume fails the task and is left as it is, as is anything an earlier
// task left there. Outputs come first: an input linked at an output's path
// is the output's starting point.
func isolate(iso *poolpod.Isolation, tmpl *wfv1.Template, inputsDir string) error {
	for _, out := range maindriver.DeclaredOutputs(tmpl) {
		if err := iso.Reserve(out.Path); err != nil {
			return fmt.Errorf("output %s %q: %w", out.Kind, out.Name, err)
		}
	}
	for _, a := range tmpl.Inputs.Artifacts {
		if a.Path == "" {
			continue
		}
		staged := filepath.Join(inputsDir, a.Name)
		if _, err := os.Lstat(staged); errors.Is(err, os.ErrNotExist) {
			// An optional artifact that wasn't there, or one the worker
			// loaded straight onto a volume.
			continue
		} else if err != nil {
			return fmt.Errorf("input artifact %q: %w", a.Name, err)
		}
		if err := iso.Symlink(staged, a.Path); err != nil {
			return fmt.Errorf("input artifact %q: %w", a.Name, err)
		}
	}
	return nil
}

// watchStop stops the task when the worker writes stop: SIGTERM to every
// process the task started, then SIGKILL to any left after half the
// release reserve. The returned function ends the watch, and any stopping
// in progress, and waits for it.
func (cfg loopConfig) watchStop(ctx context.Context, run *poolpod.Run) func() {
	logger := logging.RequireLoggerFromContext(ctx)
	return goUntilStopped(ctx, func(ctx context.Context) {
		reason, err := run.WaitStop(ctx)
		if err != nil {
			if ctx.Err() == nil {
				logger.WithError(err).Error(ctx, "failed to watch for the task being stopped")
			}
			return
		}
		logger.WithField("reason", reason).Info(ctx, "stopping the task")
		if err := poolpod.Sweep(ctx, cfg.ReleaseReserve/2); err != nil && ctx.Err() == nil {
			logger.WithError(err).Error(ctx, "failed to stop the task")
		}
	})
}

// reaper reaps the zombies a task's orphans leave, between tasks. While a
// task's command runs, waiting for it reaps them.
type reaper struct {
	stopReaping func()
}

func startReaper(ctx context.Context) *reaper {
	return &reaper{stopReaping: goUntilStopped(ctx, func(ctx context.Context) {
		wait.UntilWithContext(ctx, reap, loopReapInterval)
	})}
}

// stop stops reaping, once anything already exited has been reaped. It is
// safe to call more than once.
func (r *reaper) stop(ctx context.Context) {
	r.stopReaping()
	reap(ctx)
}

func reap(ctx context.Context) {
	if _, err := poolpod.Reap(); err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "failed to reap")
	}
}
