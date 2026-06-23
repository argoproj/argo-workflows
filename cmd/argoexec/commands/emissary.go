package commands

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

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

			// Liveness lock: the kernel releases it on process death for any
			// reason, so dependents waiting on a shared lock wake up even if
			// this process is OOM-killed or SIGKILLed before writing exitcode.
			lockFile, err := osspecific.Acquire(filepath.Join(varRunArgo, "ctr", containerName, "lock"))
			if err != nil {
				return fmt.Errorf("failed to acquire container lock: %w", err)
			}
			defer func() { _ = lockFile.Close() }()

			// Ready marker must be written after the lock is held so dependents
			// don't attempt a shared-lock acquire before it is in place.
			if err = os.WriteFile(filepath.Join(varRunArgo, "ctr", containerName, "ready"), nil, 0o644); err != nil {
				return fmt.Errorf("failed to write ready marker: %w", err)
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

			data, err := os.ReadFile(varRunArgo + "/template")
			if err != nil {
				return fmt.Errorf("failed to read template: %w", err)
			}

			if err = json.Unmarshal(data, template); err != nil {
				return fmt.Errorf("failed to unmarshal template: %w", err)
			}

			// setup signal handlers
			signals := make(chan os.Signal, 1)
			defer close(signals)
			signal.Notify(signals)
			defer signal.Reset()

			if waitErr := waitForDependencies(ctx, logger, signals); waitErr != nil {
				return waitErr
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

				go func() {
					for s := range signals {
						if osspecific.CanIgnoreSignal(s) {
							logger.WithField("signal", s).Debug(ctx, "ignore signal")
							continue
						}

						logger.WithField("signal", s).Debug(ctx, "forwarding signal")
						_ = osspecific.Kill(command.Process.Pid, s.(syscall.Signal))
					}
				}()
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

			if cmdErr == nil {
				exitCode = 0
			} else if exitError, ok := cmdErr.(argoerrors.Exited); ok {
				if exitError.ExitCode() >= 0 {
					exitCode = exitError.ExitCode()
				} else {
					exitCode = 137 // SIGTERM
				}
			}

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

// waitForDependencies blocks on each of the current container's
// containerSet dependencies. SIGTERM and SIGKILL received during the wait
// cancel it and produce exit code 143 or 137 respectively.
func waitForDependencies(ctx context.Context, logger logging.Logger, signals <-chan os.Signal) error {
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
		logger.WithField("src", srcPath).WithError(err).Error(ctx, "cannot save parameter, does not exist")
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
