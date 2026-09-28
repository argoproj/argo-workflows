package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/propagation"
	"k8s.io/client-go/util/retry"

	argoexecexecutor "github.com/argoproj/argo-workflows/v4/cmd/argoexec/executor"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoerrors "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/file"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/emissary"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/maindriver"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// varRunArgo is a var, not a const, so tests can point it at a temp dir.
var varRunArgo = common.VarRunArgoPath

// traceParentEnv serialises ctx's current span as TRACEPARENT/TRACESTATE
// entries for a child process's environment.
func traceParentEnv(ctx context.Context) []string {
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)
	env := make([]string, 0, len(carrier))
	for k, v := range carrier {
		env = append(env, strings.ToUpper(k)+"="+v)
	}
	return env
}

func NewEmissaryCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "emissary",
		SilenceUsage: true, // this prevents confusing usage message being printed when we SIGTERM
		RunE: func(cmd *cobra.Command, args []string) error {
			containerName := os.Getenv(common.EnvVarContainerName)
			includeScriptOutput := os.Getenv(common.EnvVarIncludeScriptOutput) == "true" // capture stdout/combined
			return runEmissary(cmd.Context(), containerName, newPodSource(containerName, includeScriptOutput, args), maindriver.Container{})
		},
	}
}

// newPodSource is the emissary's composition root for its task. The pod
// spec delivers exactly one: the template (file, or ARGO_TEMPLATE for
// init-less templates without a supervisor), the command from argv plus any
// offloaded args file, and the environment.
func newPodSource(containerName string, includeScriptOutput bool, args []string) maindriver.TaskSource {
	return &maindriver.PodSource{
		VarRunArgo:          varRunArgo,
		TemplateEnv:         os.Getenv(common.EnvVarTemplate),
		OffloadDir:          common.EnvConfigMountPath,
		ArgsFile:            os.Getenv(common.EnvVarContainerArgsFile),
		NodeID:              os.Getenv(common.EnvVarNodeID),
		ContainerName:       containerName,
		Command:             args,
		Env:                 os.Environ(),
		IncludeScriptOutput: includeScriptOutput,
	}
}

// runEmissary is the emissary body, with everything the command parses from
// the environment passed in as parameters so it is testable without env or
// package-level state. The pod layout (markers, locks, dependency waits,
// signals, the exitcode file) lives here; running the command is the
// driver's, and the task comes from source once the pod is ready for it.
func runEmissary(ctx context.Context, containerName string, source maindriver.TaskSource, driver maindriver.MainDriver) error {
	exitCode := 64
	logger := logging.RequireLoggerFromContext(ctx)
	// Registered before the exit code defer so that it runs after it: releasing
	// the liveness lock is what tells a dependent the exit code is readable, so
	// the lock must outlive the write. Acquired further down, once the ctr
	// directory exists, hence the nil check.
	var lockFile *os.File
	defer func() {
		if lockFile != nil {
			_ = lockFile.Close()
		}
	}()
	defer func() {
		err := os.WriteFile(varRunArgo+"/ctr/"+containerName+"/exitcode", []byte(strconv.Itoa(exitCode)), 0o644)
		if err != nil {
			logger.WithError(err).Error(ctx, "failed to write exit code")
		}
	}()

	tracer, err := tracing.New(ctx, `argoexec`)
	if err != nil {
		logger.WithFatal().WithError(err).Error(ctx, "failed to initialize tracing")
		return err
	}
	defer func() {
		if deferErr := tracer.Shutdown(context.WithoutCancel(ctx)); deferErr != nil {
			logger.WithError(deferErr).Error(ctx, "Failed to shutdown tracing")
		}
	}()

	ctx = tracing.InjectTraceContext(ctx)
	workflowName := os.Getenv(common.EnvVarWorkflowName)
	namespace, _ := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	ctx, span := tracer.StartRunMainContainer(ctx, workflowName, string(namespace))
	defer span.End()

	osspecific.AllowGrantingAccessToEveryone()

	// Dir permission set to rwxrwxrwx, so that non-root wait container can also write kill signal to the folder.
	// Note it's important varRunArgo+"/ctr/" folder is writable by all, because multiple containers may want to
	// write to it with different users.
	// This also indicates we've started.
	if err = os.MkdirAll(varRunArgo+"/ctr/"+containerName, 0o777); err != nil {
		return fmt.Errorf("failed to create ctr directory: %w", err)
	}

	// Liveness lock: the kernel releases it on process death for any
	// reason, so dependents waiting on a shared lock wake up even if
	// this process is OOM-killed or SIGKILLed before writing exitcode.
	lockFile, err = osspecific.Acquire(filepath.Join(varRunArgo, "ctr", containerName, "lock"))
	if err != nil {
		// Dependents block on the ready marker before anything else, and the
		// controller no longer terminates the pod while a sibling is still
		// running, so without it they would hang. Publish it best-effort;
		// they then read the exitcode written by the defer above.
		if writeErr := os.WriteFile(filepath.Join(varRunArgo, "ctr", containerName, "ready"), nil, 0o644); writeErr != nil {
			logger.WithError(writeErr).Error(ctx, "failed to write ready marker after lock failure")
		}
		return fmt.Errorf("failed to acquire container lock: %w", err)
	}

	// Ready marker must be written after the lock is held so dependents
	// don't attempt a shared-lock acquire before it is in place.
	if err = os.WriteFile(filepath.Join(varRunArgo, "ctr", containerName, "ready"), nil, 0o644); err != nil {
		return fmt.Errorf("failed to write ready marker: %w", err)
	}

	// In init-less pod mode the supervisor, not an init container, writes
	// /var/run/argo/template. Supervisor and main start concurrently, so
	// block until supervisor signals readiness (or failure) before reading
	// the template. Gated on an env var so legacy pods are unaffected.
	waitForReady := os.Getenv(common.EnvVarWaitForReady) == "true"
	if waitForReady {
		if waitErr := waitForSupervisorReady(ctx); waitErr != nil {
			// Distinct exit code so the controller attributes the failure
			// to supervisor pre-main setup rather than the user command.
			// The process exit code (not just the exitcode file) must carry
			// the sentinel, because inferFailedReason keys off the container's
			// terminated exit code; wrap so main propagates 65 while keeping
			// waitErr's message.
			exitCode = common.ExitCodeSupervisorPreMainFailure
			logger.WithError(waitErr).Error(ctx, "supervisor failed before main container started")
			return argoerrors.NewExitErrWithCause(exitCode, waitErr)
		}
	}

	task, ok, err := source.Next(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("no task to run")
	}
	if task.Template == nil {
		return errors.New("task has no template")
	}
	// The user's process parents its spans to runMainContainer, not to the
	// controller's span the pod spec carries. Appended so it wins: os/exec
	// keeps the last value of a duplicated key. Cloned so the source's
	// slice is untouched.
	task.Env = append(slices.Clone(task.Env), traceParentEnv(ctx)...)
	template := task.Template

	// In init-less pod mode, main can't use the legacy per-artifact
	// SubPath bind mount (kubelet races the supervisor's write). The
	// input-artifacts volume is mounted whole at /argo/inputs/artifacts
	// and the emissary symlinks each input artifact into its expected
	// path once supervisor has finished writing (guaranteed by the
	// ready-marker wait above). Only `main` runs this — ContainerSet
	// children and sidecars don't get artifact paths symlinked in.
	if waitForReady && containerName == common.MainContainerName {
		if stageErr := stageInputArtifacts(ctx, template); stageErr != nil {
			// As above: propagate the sentinel as the process exit code so
			// inferFailedReason attributes this to supervisor pre-main setup.
			exitCode = common.ExitCodeSupervisorPreMainFailure
			logger.WithError(stageErr).Error(ctx, "failed to stage input artifacts before main container started")
			return argoerrors.NewExitErrWithCause(exitCode, stageErr)
		}
	}

	// setup signal handlers
	signals := make(chan os.Signal, 1)
	defer close(signals)
	signal.Notify(signals)
	defer signal.Reset()

	if waitErr := waitForDependencies(ctx, logger, template, containerName, signals); waitErr != nil {
		return waitErr
	}

	// Resolved once, before the retry loop, so a missing binary is not
	// retried. Copied rather than written in place: the slice is the
	// source's, and a source may hand the same Task out again.
	name, err := exec.LookPath(task.Command[0])
	if err != nil {
		return fmt.Errorf("failed to find name in PATH: %w", err)
	}
	task.Command = append([]string{name}, task.Command[1:]...)

	if os.Getenv("ARGO_DEBUG_PAUSE_BEFORE") == "true" {
		// User can create the file: /ctr/NAME_OF_THE_CONTAINER/before
		// in order to break out of the wait and release the container from
		// the debugging state.
		if waitErr := file.WaitForCreate(ctx, varRunArgo+"/ctr/"+containerName+"/before"); waitErr != nil {
			return fmt.Errorf("failed waiting for debug-pause-before marker: %w", waitErr)
		}
	}

	backoff, err := template.GetRetryStrategy()
	if err != nil {
		return fmt.Errorf("failed to get retry strategy: %w", err)
	}

	// The driver hands each attempt's outputs to a collector; they are the
	// template's declared paths, so staging is deferred until retries are
	// done rather than repeated per attempt.
	var outputs outputCollector
	cmdErr := retry.OnError(backoff, func(error) bool { return true }, func() error {
		// innerCtx scopes the signal forwarders and sidecar watcher to this
		// attempt, so a retry never signals an earlier attempt's pid.
		innerCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		task.OnStart = func(pid int) {
			forwardSignals(innerCtx, signals, pid, false)
			startFileSignalHandler(innerCtx, pid, containerName)
			if slices.Contains(template.GetSidecarNames(), containerName) {
				go terminateWhenMainExits(innerCtx, logger, template, containerName)
			}
		}
		outputs = nil
		var runErr error
		exitCode, runErr = driver.Run(ctx, task, &outputs)
		return runErr
	})
	logger.WithError(cmdErr).Info(ctx, "sub-process exited")

	if os.Getenv("ARGO_DEBUG_PAUSE_AFTER") == "true" {
		// User can create the file: /ctr/NAME_OF_THE_CONTAINER/after
		// in order to break out of the wait and release the container from
		// the debugging state.
		if waitErr := file.WaitForCreate(ctx, varRunArgo+"/ctr/"+containerName+"/after"); waitErr != nil {
			return fmt.Errorf("failed waiting for debug-pause-after marker: %w", waitErr)
		}
	}

	// The sink decides which containers' outputs are staged (main only).
	var sink maindriver.ResultSink = maindriver.PodSink{VarRunArgo: varRunArgo, ContainerName: containerName, Template: template}
	for _, out := range outputs {
		if err := sink.Put(ctx, task.NodeID, out); err != nil {
			return err
		}
	}

	return cmdErr // this is the error returned from cmd.Wait(), which maybe an exitError
}

// outputCollector is a ResultSink that records the outputs the driver
// declares for an attempt, so staging happens once after retries rather than
// on every attempt.
type outputCollector []maindriver.Output

func (c *outputCollector) Put(_ context.Context, _ string, out maindriver.Output) error {
	*c = append(*c, out)
	return nil
}

// terminateWhenMainExits stops this sidecar once the template's main
// containers have exited.
func terminateWhenMainExits(ctx context.Context, logger logging.Logger, template *wfv1.Template, containerName string) {
	em, err := emissary.New()
	if err != nil {
		logger.WithError(err).Error(ctx, "failed to create emissary")
		return
	}
	mainContainerNames := template.GetMainContainerNames()
	if err := em.Wait(ctx, mainContainerNames); err != nil {
		logger.WithError(err).WithFields(logging.Fields{
			"mainContainerNames": mainContainerNames,
		}).Error(ctx, "failed to wait for main container(s)")
	}

	logger.WithFields(logging.Fields{
		"mainContainerNames": mainContainerNames,
		"containerName":      containerName,
	}).Info(ctx, "main container(s) exited, terminating container")
	if err := em.Kill(ctx, []string{containerName}, argoexecexecutor.TerminationGracePeriodDuration()); err != nil {
		logger.WithField("containerName", containerName).WithError(err).Error(ctx, "failed to terminate/kill container")
	}
}

func stageInputArtifacts(ctx context.Context, tmpl *wfv1.Template) error {
	return stageInputArtifactsAt(ctx, common.ExecutorArtifactBaseDir, tmpl)
}

// stageInputArtifactsAt is the parameterized form used by tests; production
// calls stageInputArtifacts with the constants. It links each input artifact
// into place and re-enters the working directory afterwards, stepping off it
// for the duration: linking replaces the cwd when an artifact's path is the
// container's workingDir, Windows refuses to delete a directory in use as a
// working directory, and a child forked with the deleted directory as cwd
// would see getcwd() fail and relative paths resolve to nothing. The final
// chdir follows the symlink to whatever now sits at the path.
func stageInputArtifactsAt(ctx context.Context, baseDir string, tmpl *wfv1.Template) error {
	origWd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to read working directory before staging input artifacts: %w", err)
	}
	if err := os.Chdir(varRunArgo); err != nil {
		return fmt.Errorf("failed to leave working directory before staging input artifacts: %w", err)
	}
	if err := linkInputArtifactsAt(ctx, baseDir, tmpl); err != nil {
		return err
	}
	if err := os.Chdir(origWd); err != nil {
		return fmt.Errorf("failed to re-enter working directory %q after staging input artifacts (an input artifact staged at the workingDir path must be a directory): %w", origWd, err)
	}
	return nil
}

// linkInputArtifactsAt creates a symlink at each input artifact's path
// pointing to the file that supervisor wrote under /argo/inputs/artifacts/
// <name>. This replaces the legacy SubPath bind-mount-per-artifact scheme,
// which can't be used in init-less mode because kubelet pre-creates SubPath
// entries as empty directories before supervisor can write the real file.
//
// Behavior notes for workflow authors: in init-less mode art.Path is a
// symlink rather than a regular file. `cat`, `open()`, `tar`, `cp`,
// redirection, etc. all follow symlinks transparently and see identical
// content. Code that calls `lstat`/`readlink` on art.Path will observe a
// symlink rather than a regular file. `rm art.Path` removes the symlink
// only; the underlying artifact stays in the shared emptyDir.
//
// Overlapping user volumes are handled by the executor on the write side
// (supervisor writes to /mainctrfs/<art.Path> instead of /argo/inputs/
// artifacts/<name>), so no entry appears in the input-artifacts directory
// and we skip the symlink — main already sees the file at art.Path via
// its user volume.
func linkInputArtifactsAt(ctx context.Context, baseDir string, tmpl *wfv1.Template) error {
	logger := logging.RequireLoggerFromContext(ctx)
	for _, art := range tmpl.Inputs.Artifacts {
		src := filepath.Join(baseDir, art.Name)
		if _, statErr := os.Lstat(src); statErr != nil {
			if os.IsNotExist(statErr) {
				logger.WithFields(logging.Fields{"name": art.Name, "path": art.Path}).Info(ctx, "no input-artifacts entry (optional or overlap) — skipping symlink")
				continue
			}
			return fmt.Errorf("failed to stat input artifact %q at %s: %w", art.Name, src, statErr)
		}
		dst := art.Path
		if dst == "" {
			continue
		}
		if parent := filepath.Dir(dst); parent != "" && parent != "/" {
			if err := os.MkdirAll(parent, 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory for artifact %q at %s: %w", art.Name, dst, err)
			}
		}
		// If nothing exists at art.Path, just create the symlink. Creating is
		// always safe — os.Symlink returns EEXIST rather than overwriting and the
		// MkdirAll above only ever creates — so even when art.Path resolves into a
		// user volume we deliberately let the artifact land there (the user asked
		// for it). Only an *overwrite* can destroy data, and that is gated below.
		if _, err := os.Lstat(dst); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to stat artifact path %q at %s: %w", art.Name, dst, err)
			}
		} else {
			// Something is already at art.Path. Replacing it (os.RemoveAll then
			// symlink) reproduces the legacy SubPath mount's shadowing — but only
			// when it is safe. RemoveAll resolves symlinks in the parent chain, so
			// resolve the parent to find where the delete would actually land
			// (resolve the parent, not the final element, so an image symlink *at*
			// art.Path is just unlinked rather than followed). If that resolved
			// path overlaps a user-declared volume, clearing it would recurse into
			// and destroy a live PVC/hostPath/emptyDir, so refuse. Benign system
			// mounts (tmpfs /run, the overlay rootfs) are not declared user volumes
			// and so remain safe to shadow.
			realParent, evalErr := filepath.EvalSymlinks(filepath.Dir(dst))
			if evalErr != nil {
				return fmt.Errorf("failed to resolve parent of artifact path %q at %s: %w", art.Name, dst, evalErr)
			}
			resolved := filepath.Join(realParent, filepath.Base(dst))
			if mnt := common.FindOverlappingVolume(tmpl, resolved); mnt != nil {
				return fmt.Errorf("refusing to stage input artifact %q at %s: it resolves to %s inside volume mount %q (%s), and clearing it would destroy the mounted volume; change the artifact path or volume mount so they do not overlap", art.Name, dst, resolved, mnt.Name, mnt.MountPath)
			}
			if mnt := common.FindVolumeMountNestedUnderPath(tmpl, resolved); mnt != nil {
				return fmt.Errorf("refusing to stage input artifact %q at %s: it resolves to %s which contains volume mount %q (%s), and clearing it would destroy the mounted volume; change the artifact path or volume mount so they do not overlap", art.Name, dst, resolved, mnt.Name, mnt.MountPath)
			}
			if rmErr := os.RemoveAll(dst); rmErr != nil {
				return fmt.Errorf("failed to clear existing path for artifact %q at %s: %w", art.Name, dst, rmErr)
			}
		}
		if err := os.Symlink(src, dst); err != nil {
			return fmt.Errorf("failed to symlink input artifact %q (%s -> %s): %w", art.Name, dst, src, err)
		}
		logger.WithFields(logging.Fields{"name": art.Name, "src": src, "dst": dst}).Debug(ctx, "linked input artifact")
	}
	return nil
}

// waitForSupervisorReady blocks until the supervisor's status marker reports a
// terminal outcome (READY/FAILED), or until the supervisor is presumed dead.
// Used only in init-less pod mode where main and supervisor start concurrently.
// VarRunArgoPath itself is guaranteed to exist because the emissary has
// already created /var/run/argo/ctr/<name> earlier in main, which MkdirAll'd
// the full parent chain.
func waitForSupervisorReady(ctx context.Context) error {
	return waitForSupervisorReadyAt(ctx, common.StatusMarkerPath, supervisorHeartbeatTimeout, supervisorStatusPollInterval)
}

// waitForSupervisorReadyAt is the parameterized form used by tests; production
// calls waitForSupervisorReady with the constants.
//
// The supervisor rewrites the marker as RUNNING on a heartbeat (see
// startStatusHeartbeat). main treats a marker that has neither appeared nor
// advanced within timeout as a dead supervisor and fails fast rather than
// hanging to the pod deadline. An inotify watcher gives low-latency pickup of
// the terminal READY/FAILED write; a parallel ticker (period pollInterval)
// checks for staleness, because inotify cannot signal the *absence* of writes.
func waitForSupervisorReadyAt(ctx context.Context, statusPath string, timeout, pollInterval time.Duration) error {
	logger := logging.RequireLoggerFromContext(ctx)
	logger.Info(ctx, "waiting for supervisor status marker")
	start := time.Now()

	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	resCh := make(chan error, 1)
	finish := func(err error) {
		select {
		case resCh <- err:
		default: // a result already landed; first one wins
		}
		cancel()
	}

	// Low-latency terminal detection: re-evaluate on every write to the marker
	// (heartbeats and the terminal write both fire here).
	go func() {
		werr := file.WatchFile(watchCtx, statusPath, func() {
			if done, err := evaluateSupervisorStatus(statusPath, timeout, start); done {
				finish(err)
			}
		})
		if werr != nil && watchCtx.Err() == nil {
			finish(fmt.Errorf("watching supervisor status: %w", werr))
		}
	}()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-resCh:
			if err == nil {
				logger.Info(ctx, "supervisor is ready")
			}
			return err
		case <-ticker.C:
			if done, err := evaluateSupervisorStatus(statusPath, timeout, start); done {
				finish(err)
			}
		}
	}
}

// evaluateSupervisorStatus reads the status marker once and decides whether main
// can stop waiting. done=false means keep waiting. start is main's wait-start
// reference, used to bound the case where the marker never appears at all. It is
// safe to call concurrently — it only reads the filesystem.
func evaluateSupervisorStatus(statusPath string, timeout time.Duration, start time.Time) (done bool, err error) {
	fi, statErr := os.Stat(statusPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			if time.Since(start) > timeout {
				return true, fmt.Errorf("supervisor presumed dead: status marker never appeared within %s", timeout)
			}
			return false, nil
		}
		return true, fmt.Errorf("stat supervisor status: %w", statErr)
	}
	body, readErr := os.ReadFile(statusPath)
	if readErr != nil {
		// Stat just succeeded, so a read failure here means we raced the
		// supervisor's atomic rename (the old inode vanished between stat and
		// read). Treat it as transient and re-evaluate on the next tick/event
		// rather than failing the wait.
		//nolint:nilerr // deliberate: swallow the transient read error and retry
		return false, nil
	}
	token, message := parseSupervisorStatus(body)
	switch token {
	case statusReady:
		return true, nil
	case statusFailed:
		return true, fmt.Errorf("supervisor reported pre-main failure: %s", message)
	default:
		// RUNNING, or a transient/partial read: the supervisor is alive only if
		// it is still heartbeating, i.e. the marker's mtime is fresh.
		if time.Since(fi.ModTime()) > timeout {
			return true, fmt.Errorf("supervisor presumed dead: no status update within %s", timeout)
		}
		return false, nil
	}
}

// parseSupervisorStatus splits the marker into its first-line token and the
// remaining message (used by the FAILED token to carry the cause).
func parseSupervisorStatus(body []byte) (token, message string) {
	first, rest, _ := strings.Cut(string(body), "\n")
	return strings.TrimSpace(first), strings.TrimSpace(rest)
}

// exitCodeFromErr maps the result of waiting on a sub-process to a numeric exit
// code: 0 on success, the process's own exit code when it exited normally, or
// 137 when it was signalled with no usable code. For any other (non-exit) error
// the current code is preserved, matching the legacy behaviour where such errors
// left the caller's default exit code in place.
func exitCodeFromErr(cmdErr error, current int) int {
	if cmdErr == nil {
		return 0
	}
	if exitError, ok := cmdErr.(argoerrors.Exited); ok {
		if exitError.ExitCode() >= 0 {
			return exitError.ExitCode()
		}
		return 137 // SIGTERM
	}
	return current
}

// waitForDependencies blocks on each of the current container's
// containerSet dependencies. SIGTERM and SIGKILL received during the wait
// cancel it and produce exit code 143 or 137 respectively.
func waitForDependencies(ctx context.Context, logger logging.Logger, template *wfv1.Template, containerName string, signals <-chan os.Signal) error {
	var deps []string
	for _, x := range template.ContainerSet.GetGraph() {
		if x.Name == containerName {
			deps = x.Dependencies
			break
		}
	}
	if len(deps) == 0 {
		return nil
	}

	depCtx, cancelDepWait := context.WithCancel(ctx)
	defer cancelDepWait()

	signalDone := make(chan struct{})
	depSignalExitCode := make(chan int, 1)
	go func() {
		for {
			select {
			case <-signalDone:
				return
			case s, ok := <-signals:
				if !ok {
					return
				}
				if osspecific.CanIgnoreSignal(s) {
					continue
				}
				switch s {
				case osspecific.Term:
					depSignalExitCode <- 143
					cancelDepWait()
					return
				case os.Kill:
					depSignalExitCode <- 137
					cancelDepWait()
					return
				}
			}
		}
	}()

	var depErr error
	for _, y := range deps {
		logger.WithField("dependency", y).Info(ctx, "waiting for dependency")
		if err := waitForDependency(depCtx, y); err != nil {
			depErr = err
			break
		}
	}

	close(signalDone)
	select {
	case ec := <-depSignalExitCode:
		return argoerrors.NewExitErr(ec)
	default:
	}
	return depErr
}

// waitForDependency blocks until depName's ready marker exists, its lock is
// released (i.e. its process has exited for any reason), and then reads its
// exitcode file. A missing exitcode means the dep died without reporting.
func waitForDependency(ctx context.Context, depName string) error {
	depDir := filepath.Clean(varRunArgo + "/ctr/" + depName)
	// Pre-create in case the dep container hasn't started yet, so fsnotify
	// has a directory to watch.
	if err := os.MkdirAll(depDir, 0o777); err != nil {
		return fmt.Errorf("failed to create dependency dir: %w", err)
	}
	if err := file.WaitForCreate(ctx, filepath.Join(depDir, "ready")); err != nil {
		return err
	}
	if err := osspecific.WaitForSharedLock(ctx, filepath.Join(depDir, "lock")); err != nil {
		return err
	}
	data, readErr := os.ReadFile(filepath.Join(depDir, "exitcode"))
	if readErr != nil {
		return fmt.Errorf("dependency %q died without reporting exit code", depName)
	}
	code, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
	if parseErr != nil {
		return fmt.Errorf("dependency %q died without reporting exit code", depName)
	}
	if code != 0 {
		return fmt.Errorf("dependency %q exited with non-zero code: %d", depName, code)
	}
	return nil
}

func startCommand(ctx context.Context, name string, args []string, template *wfv1.Template, containerName string, includeScriptOutput bool) (*exec.Cmd, func(), error) {
	logger := logging.RequireLoggerFromContext(ctx)

	command := exec.CommandContext(ctx, name, args...)
	command.Env = os.Environ()

	var closer = func() {}
	var stdout io.Writer = os.Stdout
	var stderr io.Writer = os.Stderr

	// this may not be that important an optimisation, except for very long logs we don't want to capture
	if includeScriptOutput || template.SaveLogsAsArtifact() {
		logger.Info(ctx, "capturing logs")
		stdoutf, err := os.OpenFile(varRunArgo+"/ctr/"+containerName+"/stdout", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open stdout: %w", err)
		}
		combinedf, err := os.OpenFile(varRunArgo+"/ctr/"+containerName+"/combined", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Close stdoutf to avoid leaking the file descriptor opened above.
			_ = stdoutf.Close()
			return nil, nil, fmt.Errorf("failed to open combined: %w", err)
		}
		stdout = io.MultiWriter(stdout, stdoutf, combinedf)
		stderr = io.MultiWriter(stderr, combinedf)

		closer = func() {
			_ = stdoutf.Close()
			_ = combinedf.Close()
		}
	}

	command.Stdout = stdout
	command.Stderr = stderr

	cmdCloser, err := osspecific.StartCommand(ctx, command)
	if err != nil {
		return nil, nil, err
	}

	origCloser := closer

	closer = func() {
		cmdCloser()
		origCloser()
	}

	return command, closer, nil
}
