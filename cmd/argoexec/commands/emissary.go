package commands

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/propagation"
	"k8s.io/client-go/util/retry"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/archive"
	argoerrors "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/file"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/emissary"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

var (
	varRunArgo          = common.VarRunArgoPath
	containerName       = os.Getenv(common.EnvVarContainerName)
	includeScriptOutput = os.Getenv(common.EnvVarIncludeScriptOutput) == "true" // capture stdout/combined
	template            = &wfv1.Template{}
)

func injectTraceParent(ctx context.Context) {
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(ctx, carrier)

	for k, v := range carrier {
		os.Setenv(strings.ToUpper(k), v)
	}
}

func NewEmissaryCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "emissary",
		SilenceUsage: true, // this prevents confusing usage message being printed when we SIGTERM
		RunE: func(cmd *cobra.Command, args []string) error {
			exitCode := 64
			ctx := cmd.Context()
			logger := logging.RequireLoggerFromContext(ctx)
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
			injectTraceParent(ctx)

			osspecific.AllowGrantingAccessToEveryone()

			// Dir permission set to rwxrwxrwx, so that non-root wait container can also write kill signal to the folder.
			// Note it's important varRunArgo+"/ctr/" folder is writable by all, because multiple containers may want to
			// write to it with different users.
			// This also indicates we've started.
			if err = os.MkdirAll(varRunArgo+"/ctr/"+containerName, 0o777); err != nil {
				return fmt.Errorf("failed to create ctr directory: %w", err)
			}

			name, args := args[0], args[1:]

			// Check if args were offloaded to a file (for large args that exceed exec limit)
			if argsFile := os.Getenv(common.EnvVarContainerArgsFile); argsFile != "" {
				logger.WithField("argsFile", argsFile).Info(ctx, "Reading container args from file")
				argsData, readErr := os.ReadFile(argsFile)
				if readErr != nil {
					return fmt.Errorf("failed to read container args file %s: %w", argsFile, readErr)
				}
				var fileArgs []string
				if err = json.Unmarshal(argsData, &fileArgs); err != nil {
					return fmt.Errorf("failed to unmarshal container args: %w", err)
				}
				args = append(args, fileArgs...)
				logger.WithField("count", len(fileArgs)).Info(ctx, "Loaded container args from file")

				// Check for a large args and offload to file if needed
				// This avoids the exec() "argument list too long" error
				// Downstream programs should support @filename for parsing large args
				for i := 0; i < len(args); i++ {
					if len(args[i]) > common.MaxEnvVarLen {
						filePath := fmt.Sprintf("/tmp/argo_arg_%d.txt", i)
						if err = os.WriteFile(filePath, []byte(args[i]), 0o644); err != nil {
							return fmt.Errorf("failed to write large arg %d to file: %w", i, err)
						}
						logger.WithFields(logging.Fields{
							"argIndex": i,
							"size":     len(args[i]),
							"filePath": filePath,
						}).Info(ctx, "Offloaded large argument to file. Downstream program must support @filename syntax")
						args[i] = "@" + filePath
					}
				}
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

			data, err := readTemplate()
			if err != nil {
				return fmt.Errorf("failed to read template: %w", err)
			}

			if err = json.Unmarshal(data, template); err != nil {
				return fmt.Errorf("failed to unmarshal template: %w", err)
			}

			// In init-less pod mode, main can't use the legacy per-artifact
			// SubPath bind mount (kubelet races the supervisor's write). The
			// input-artifacts volume is mounted whole at /argo/inputs/artifacts
			// and the emissary symlinks each input artifact into its expected
			// path once supervisor has finished writing (guaranteed by the
			// ready-marker wait above). Only `main` runs this — ContainerSet
			// children and sidecars don't get artifact paths symlinked in.
			if waitForReady && containerName == common.MainContainerName {
				if linkErr := linkInputArtifacts(ctx, template); linkErr != nil {
					// As above: propagate the sentinel as the process exit code so
					// inferFailedReason attributes this to supervisor pre-main setup.
					exitCode = common.ExitCodeSupervisorPreMainFailure
					logger.WithError(linkErr).Error(ctx, "failed to stage input artifacts before main container started")
					return argoerrors.NewExitErrWithCause(exitCode, linkErr)
				}
			}

			// setup signal handlers
			signals := make(chan os.Signal, 1)
			defer close(signals)
			signal.Notify(signals)
			defer signal.Reset()

			for _, x := range template.ContainerSet.GetGraph() {
				if x.Name == containerName {
					for _, y := range x.Dependencies {
						logger.WithField("dependency", y).Info(ctx, "waiting for dependency")
						depDir := filepath.Clean(varRunArgo + "/ctr/" + y)
						// The dependency container will MkdirAll this too, but may not have
						// started yet; pre-create it so we can install an inotify watch on it.
						if err = os.MkdirAll(depDir, 0o777); err != nil {
							return fmt.Errorf("failed to create dependency dir: %w", err)
						}
						depExitPath := filepath.Join(depDir, "exitcode")
						code, waitErr := waitForDependencyExitCode(ctx, depExitPath, signals)
						if waitErr != nil {
							return waitErr
						}
						exitCode = code
						if exitCode != 0 {
							return fmt.Errorf("dependency %q exited with non-zero code: %d", y, exitCode)
						}
					}
				}
			}

			name, err = exec.LookPath(name)
			if err != nil {
				return fmt.Errorf("failed to find name in PATH: %w", err)
			}

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

			cmdErr := retry.OnError(backoff, func(error) bool { return true }, func() error {
				command, closer, err := startCommand(ctx, name, args, template)
				if err != nil {
					return fmt.Errorf("failed to start command: %w", err)
				}
				defer closer()

				forwardSignals(ctx, signals, command.Process.Pid, false)
				pid := command.Process.Pid
				innerCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				startFileSignalHandler(innerCtx, pid)
				for _, sidecarName := range template.GetSidecarNames() {
					if sidecarName == containerName {
						em, err := emissary.New()
						if err != nil {
							return fmt.Errorf("failed to create emissary: %w", err)
						}

						go func() {
							mainContainerNames := template.GetMainContainerNames()
							err = em.Wait(innerCtx, mainContainerNames)
							if err != nil {
								logger.WithError(err).WithFields(logging.Fields{
									"mainContainerNames": mainContainerNames,
								}).Error(innerCtx, "failed to wait for main container(s)")
							}

							logger.WithFields(logging.Fields{
								"mainContainerNames": mainContainerNames,
								"containerName":      containerName,
							}).Info(innerCtx, "main container(s) exited, terminating container")
							err = em.Kill(innerCtx, []string{containerName}, executor.GetTerminationGracePeriodDuration())
							if err != nil {
								logger.WithField("containerName", containerName).WithError(err).Error(innerCtx, "failed to terminate/kill container")
							}
						}()

						break
					}
				}

				return osspecific.Wait(command.Process)
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

			exitCode = exitCodeFromErr(cmdErr, exitCode)

			if containerName == common.MainContainerName {
				for _, x := range template.Outputs.Parameters {
					if x.ValueFrom != nil && x.ValueFrom.Path != "" {
						if err := saveParameter(ctx, x.ValueFrom.Path); err != nil {
							return err
						}
					}
				}
				for _, x := range template.Outputs.Artifacts {
					if x.Path != "" {
						if err := saveArtifact(ctx, x.Path); err != nil {
							return err
						}
					}
				}
			} else {
				logger.Info(ctx, "not saving outputs - not main container")
			}

			return cmdErr // this is the error returned from cmd.Wait(), which maybe an exitError
		},
	}
}

// readTemplate returns the serialized template JSON. It prefers
// /var/run/argo/template (legacy: init container wrote it; init-less with
// supervisor: supervisor wrote it), and falls back to the ARGO_TEMPLATE env
// var when the file is absent. This covers the init-less case for templates
// that don't run a supervisor (data, resource-without-logs) — the controller
// sets ARGO_TEMPLATE directly on main in that case.
//
// Offload-sentinel resolution is shared with the legacy init container via
// common.ResolveTemplateEnvValue.
func readTemplate() ([]byte, error) {
	return readTemplateAt(varRunArgo+"/template", common.EnvConfigMountPath)
}

// readTemplateAt is the path-parameterized form used by tests; production
// calls readTemplate with the constants.
func readTemplateAt(filePath, offloadDir string) ([]byte, error) {
	if data, err := os.ReadFile(filePath); err == nil {
		return data, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	envVal, ok := os.LookupEnv(common.EnvVarTemplate)
	if !ok {
		return nil, fmt.Errorf("neither %s nor %s is available", filePath, common.EnvVarTemplate)
	}
	return common.ResolveTemplateEnvValue(envVal, offloadDir)
}

// linkInputArtifacts creates a symlink at each input artifact's path pointing
// to the file that supervisor wrote under /argo/inputs/artifacts/<name>.
// This replaces the legacy SubPath bind-mount-per-artifact scheme, which
// can't be used in init-less mode because kubelet pre-creates SubPath
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
func linkInputArtifacts(ctx context.Context, tmpl *wfv1.Template) error {
	return linkInputArtifactsAt(ctx, common.ExecutorArtifactBaseDir, tmpl)
}

// linkInputArtifactsAt is the parameterized form used by tests; production calls
// linkInputArtifacts with the constants.
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

// waitForDependencyExitCode blocks until the given dependency exitcode file is
// written, or until a SIGTERM/SIGKILL signal is received. It uses inotify on
// the parent directory rather than polling.
//
// We deliberately do not select on ctx.Done() here: argoexec's root context is
// bound to SIGTERM via signal.NotifyContext in main.go, so when SIGTERM
// arrives both ctx.Done() and the signals channel fire simultaneously. If
// select picked the ctx.Done() arm, we'd return context.Canceled instead of
// exit code 143 — breaking tests like TestSignaledContainerSet that assert on
// the 143 / 137 exit codes. Consuming signals directly gives us the correct
// exit code; if the outer ctx is ever cancelled without a corresponding
// signal, the parent process will escalate to SIGKILL.
func waitForDependencyExitCode(ctx context.Context, depExitPath string, signals <-chan os.Signal) (int, error) {
	type result struct {
		code int
		err  error
	}
	results := make(chan result, 1)

	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		err := file.WatchFile(watchCtx, depExitPath, func() {
			data, readErr := os.ReadFile(depExitPath)
			if readErr != nil {
				return
			}
			code, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if parseErr != nil {
				// File created but not yet fully written; await the next event.
				return
			}
			select {
			case results <- result{code: code}:
			default:
			}
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			select {
			case results <- result{err: err}:
			default:
			}
		}
	}()

	for {
		select {
		case s := <-signals:
			switch s {
			case osspecific.Term:
				// exit with 128 + 15 (SIGTERM)
				return 0, argoerrors.NewExitErr(143)
			case os.Kill:
				// exit with 128 + 9 (SIGKILL)
				return 0, argoerrors.NewExitErr(137)
			}
		case r := <-results:
			return r.code, r.err
		}
	}
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

func startCommand(ctx context.Context, name string, args []string, template *wfv1.Template) (*exec.Cmd, func(), error) {
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

func saveArtifact(ctx context.Context, srcPath string) error {
	logger := logging.RequireLoggerFromContext(ctx)

	if common.FindOverlappingVolume(template, srcPath) != nil {
		logger.WithField("srcPath", srcPath).Info(ctx, "no need to save artifact - on overlapping volume")
		return nil
	}
	if _, err := os.Stat(srcPath); os.IsNotExist(err) { // might be optional, so we ignore
		logger.WithField("srcPath", srcPath).WithError(err).Warn(ctx, "cannot save artifact")
		return nil
	}
	dstPath := filepath.Join(varRunArgo, "/outputs/artifacts/", strings.TrimSuffix(srcPath, "/")+".tgz")
	logger.WithFields(logging.Fields{
		"src": srcPath,
		"dst": dstPath,
	}).Info(ctx, "saving artifact")
	z := filepath.Dir(dstPath)
	if err := os.MkdirAll(z, 0o755); err != nil { // chmod rwxr-xr-x
		return fmt.Errorf("failed to create directory %s: %w", z, err)
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination %s: %w", dstPath, err)
	}
	defer func() { _ = dst.Close() }()
	if err = archive.TarGzToWriter(ctx, srcPath, gzip.DefaultCompression, dst); err != nil {
		return fmt.Errorf("failed to tarball the output %s to %s: %w", srcPath, dstPath, err)
	}
	if err = dst.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", dstPath, err)
	}
	return nil
}

func saveParameter(ctx context.Context, srcPath string) error {
	logger := logging.RequireLoggerFromContext(ctx)

	if common.FindOverlappingVolume(template, srcPath) != nil {
		logger.WithField("src", srcPath).Info(ctx, "no need to save parameter - on overlapping volume")
		return nil
	}
	src, err := os.Open(filepath.Clean(srcPath))
	if os.IsNotExist(err) { // might be optional, so we ignore
		logger.WithField("src", srcPath).WithError(err).Warn(ctx, "cannot save parameter, does not exist")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", srcPath, err)
	}
	defer func() { _ = src.Close() }()
	dstPath := varRunArgo + "/outputs/parameters/" + srcPath
	logger.WithFields(logging.Fields{
		"src": srcPath,
		"dst": dstPath,
	}).Info(ctx, "saving parameter")
	z := filepath.Dir(dstPath)
	if mkdirErr := os.MkdirAll(z, 0o755); mkdirErr != nil { // chmod rwxr-xr-x
		return fmt.Errorf("failed to create directory %s: %w", z, mkdirErr)
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", srcPath, err)
	}
	defer func() { _ = dst.Close() }()
	if _, err = io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", srcPath, dstPath, err)
	}
	if err = dst.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", dstPath, err)
	}
	return nil
}
