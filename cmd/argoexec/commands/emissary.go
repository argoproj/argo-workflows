package commands

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/retry"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/archive"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	osspecific "github.com/argoproj/argo-workflows/v3/workflow/executor/os-specific"
)

var (
	varRunArgo          = common.VarRunArgoPath
	containerName       = os.Getenv(common.EnvVarContainerName)
	includeScriptOutput = os.Getenv(common.EnvVarIncludeScriptOutput) == "true" // capture stdout/combined
	template            = &wfv1.Template{}
	logger              = log.WithField("argo", true)
)

func NewEmissaryCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "emissary",
		SilenceUsage: true, // this prevents confusing usage message being printed when we SIGTERM
		RunE: func(cmd *cobra.Command, args []string) error {
			exitCode := 64

			defer func() {
				err := ioutil.WriteFile(varRunArgo+"/ctr/"+containerName+"/exitcode", []byte(strconv.Itoa(exitCode)), 0o644)
				if err != nil {
					logger.Error(fmt.Errorf("failed to write exit code: %w", err))
				}
			}()

			osspecific.AllowGrantingAccessToEveryone()

			// Dir permission set to rwxrwxrwx, so that non-root wait container can also write kill signal to the folder.
			// Note it's important varRunArgo+"/ctr/" folder is writable by all, because multiple containers may want to
			// write to it with different users.
			// This also indicates we've started.
			if err := os.MkdirAll(varRunArgo+"/ctr/"+containerName, 0o777); err != nil {
				return fmt.Errorf("failed to create ctr directory: %w", err)
			}

			name, args := args[0], args[1:]

			signals := make(chan os.Signal, 1)
			defer close(signals)
			signal.Notify(signals)
			defer signal.Reset()
			go func() {
				for s := range signals {
					if !osspecific.IsSIGCHLD(s) {
						_ = osspecific.Kill(-os.Getpid(), s.(syscall.Signal))
					}
				}
			}()

			data, err := ioutil.ReadFile(varRunArgo + "/template")
			if err != nil {
				return fmt.Errorf("failed to read template: %w", err)
			}

			if err := json.Unmarshal(data, template); err != nil {
				return fmt.Errorf("failed to unmarshal template: %w", err)
			}

			for _, x := range template.ContainerSet.GetGraph() {
				if x.Name == containerName {
					for _, y := range x.Dependencies {
						logger.Infof("waiting for dependency %q", y)
						for {
							data, err := ioutil.ReadFile(filepath.Clean(varRunArgo + "/ctr/" + y + "/exitcode"))
							if os.IsNotExist(err) {
								time.Sleep(time.Second)
								continue
							}
							exitCode, err := strconv.Atoi(string(data))
							if err != nil {
								return fmt.Errorf("failed to read exit-code of dependency %q: %w", y, err)
							}
							if exitCode != 0 {
								return fmt.Errorf("dependency %q exited with non-zero code: %d", y, exitCode)
							}
							break
						}
					}
				}
			}

			name, err = exec.LookPath(name)
			if err != nil {
				return fmt.Errorf("failed to find name in PATH: %w", err)
			}

			if _, ok := os.LookupEnv("ARGO_DEBUG_PAUSE_BEFORE"); ok {
				for {
					// User can create the file: /ctr/NAME_OF_THE_CONTAINER/before
					// in order to break out of the sleep and release the container from
					// the debugging state.
					if _, err := os.Stat(varRunArgo + "/ctr/" + containerName + "/before"); os.IsNotExist(err) {
						time.Sleep(time.Second)
						continue
					}
					break
				}
			}

			backoff, err := template.GetRetryStrategy()
			if err != nil {
				return fmt.Errorf("failed to get retry strategy: %w", err)
			}

			var command *exec.Cmd
			var stdout *os.File
			var combined *os.File
			cmdErr := retry.OnError(backoff, func(error) bool { return true }, func() error {
				if stdout != nil {
					stdout.Close()
				}
				if combined != nil {
					combined.Close()
				}
				command, stdout, combined, err = createCommand(name, args, template)
				if err != nil {
					return fmt.Errorf("failed to create command: %w", err)
				}

				if err := command.Start(); err != nil {
					return err
				}

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						default:
							data, _ := ioutil.ReadFile(filepath.Clean(varRunArgo + "/ctr/" + containerName + "/signal"))
							_ = os.Remove(varRunArgo + "/ctr/" + containerName + "/signal")
							s, _ := strconv.Atoi(string(data))
							if s > 0 {
								_ = osspecific.Kill(command.Process.Pid, syscall.Signal(s))
							}
							time.Sleep(2 * time.Second)
						}
					}
				}()
				return command.Wait()
			})
			defer stdout.Close()
			defer combined.Close()

			if _, ok := os.LookupEnv("ARGO_DEBUG_PAUSE_AFTER"); ok {
				for {
					// User can create the file: /ctr/NAME_OF_THE_CONTAINER/after
					// in order to break out of the sleep and release the container from
					// the debugging state.
					if _, err := os.Stat(varRunArgo + "/ctr/" + containerName + "/after"); os.IsNotExist(err) {
						time.Sleep(time.Second)
						continue
					}
					break
				}
			}

			if cmdErr == nil {
				exitCode = 0
			} else if exitError, ok := cmdErr.(*exec.ExitError); ok {
				if exitError.ExitCode() >= 0 {
					exitCode = exitError.ExitCode()
				} else {
					exitCode = 137 // SIGTERM
				}
			}

			if containerName == common.MainContainerName {
				for _, x := range template.Outputs.Parameters {
					if x.ValueFrom != nil && x.ValueFrom.Path != "" {
						if err := saveParameter(x.ValueFrom.Path); err != nil {
							return err
						}
					}
				}
				for _, x := range template.Outputs.Artifacts {
					if x.Path != "" {
						if err := saveArtifact(x.Path); err != nil {
							return err
						}
					}
				}
			} else {
				logger.Info("not saving outputs - not main container")
			}

			return cmdErr // this is the error returned from cmd.Wait(), which maybe an exitError
		},
	}
}

func createCommand(name string, args []string, template *wfv1.Template) (*exec.Cmd, *os.File, *os.File, error) {
	command := exec.Command(name, args...)
	command.Env = os.Environ()
	command.SysProcAttr = &syscall.SysProcAttr{}
	osspecific.Setpgid(command.SysProcAttr)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	var stdout *os.File
	var combined *os.File
	var err error
	// this may not be that important an optimisation, except for very long logs we don't want to capture
	if includeScriptOutput || template.SaveLogsAsArtifact() {
		logger.Info("capturing logs")
		stdout, err = os.OpenFile(varRunArgo+"/ctr/"+containerName+"/stdout", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to open stdout: %w", err)
		}
		combined, err = os.OpenFile(varRunArgo+"/ctr/"+containerName+"/combined", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to open combined: %w", err)
		}
		command.Stdout = io.MultiWriter(os.Stdout, stdout, combined)
		command.Stderr = io.MultiWriter(os.Stderr, combined)
	}
	return command, stdout, combined, nil
}

func saveArtifact(srcPath string) error {
	if common.FindOverlappingVolume(template, srcPath) != nil {
		logger.Infof("no need to save artifact - on overlapping volume: %s", srcPath)
		return nil
	}
	if _, err := os.Stat(srcPath); os.IsNotExist(err) { // might be optional, so we ignore
		logger.WithError(err).Errorf("cannot save artifact %s", srcPath)
		return nil
	}
	dstPath := filepath.Join(varRunArgo, "/outputs/artifacts/", strings.TrimSuffix(srcPath, "/")+".tgz")
	logger.Infof("%s -> %s", srcPath, dstPath)
	z := filepath.Dir(dstPath)
	if err := os.MkdirAll(z, 0o755); err != nil { // chmod rwxr-xr-x
		return fmt.Errorf("failed to create directory %s: %w", z, err)
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination %s: %w", dstPath, err)
	}
	defer func() { _ = dst.Close() }()
	if err = archive.TarGzToWriter(srcPath, gzip.DefaultCompression, dst); err != nil {
		return fmt.Errorf("failed to tarball the output %s to %s: %w", srcPath, dstPath, err)
	}
	if err = dst.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", dstPath, err)
	}
	return nil
}

func saveParameter(srcPath string) error {
	if common.FindOverlappingVolume(template, srcPath) != nil {
		logger.Infof("no need to save parameter - on overlapping volume: %s", srcPath)
		return nil
	}
	src, err := os.Open(filepath.Clean(srcPath))
	if os.IsNotExist(err) { // might be optional, so we ignore
		logger.WithError(err).Errorf("cannot save parameter %s", srcPath)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", srcPath, err)
	}
	defer func() { _ = src.Close() }()
	dstPath := varRunArgo + "/outputs/parameters/" + srcPath
	logger.Infof("%s -> %s", srcPath, dstPath)
	z := filepath.Dir(dstPath)
	if err := os.MkdirAll(z, 0o755); err != nil { // chmod rwxr-xr-x
		return fmt.Errorf("failed to create directory %s: %w", z, err)
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
