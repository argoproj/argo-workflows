package commands

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/archive"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util/path"
)

var (
	varArgo       = "/var/run/argo"
	containerName = os.Getenv("ARGO_CONTAINER_NAME")
	template      = &wfv1.Template{}
)

func NewEmissaryCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "emissary",
		SilenceUsage: true, // this prevents confusing usage message being printed when we SIGTERM
		RunE: func(cmd *cobra.Command, args []string) error {
			exitCode := 64

			defer func() {
				err := ioutil.WriteFile(varArgo+"/ctr/"+containerName+"/exitcode", []byte(strconv.Itoa(exitCode)), 0600)
				if err != nil {
					println(fmt.Errorf("failed to write exit code: %w", err))
				}
			}()

			// this also indicates we've started
			if err := os.MkdirAll(varArgo+"/ctr/"+containerName, 0700); err != nil {
				return fmt.Errorf("failed to create ctr directory: %w", err)
			}

			name, args := args[0], args[1:]

			signals := make(chan os.Signal, 1)
			defer close(signals)
			signal.Notify(signals)
			defer signal.Reset()
			go func() {
				for s := range signals {
					if s != syscall.SIGCHLD {
						_ = syscall.Kill(-os.Getpid(), s.(syscall.Signal))
					}
				}
			}()

			data, err := ioutil.ReadFile(varArgo + "/template")
			if err != nil {
				return fmt.Errorf("failed to read template: %w", err)
			}

			if err := json.Unmarshal(data, template); err != nil {
				return fmt.Errorf("failed to unmarshal template: %w", err)
			}

			name, err = path.Search(name)
			if err != nil {
				return fmt.Errorf("failed to find name in PATH: %w", err)
			}

			command := exec.Command(name, args...)
			command.Env = os.Environ()
			command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

			stdout, err := os.Create(varArgo + "/ctr/" + containerName + "/stdout")
			if err != nil {
				return fmt.Errorf("failed to open stdout: %w", err)
			}
			defer func() { _ = stdout.Close() }()
			command.Stdout = io.MultiWriter(os.Stdout, stdout)

			stderr, err := os.Create(varArgo + "/ctr/" + containerName + "/stderr")
			if err != nil {
				return fmt.Errorf("failed to open stderr: %w", err)
			}
			defer func() { _ = stderr.Close() }()
			command.Stderr = io.MultiWriter(os.Stderr, stderr)

			if err := command.Start(); err != nil {
				return err
			}

			go func() {
				for {
					data, _ := ioutil.ReadFile(varArgo + "/ctr/" + containerName + "/signal")
					_ = os.Remove(varArgo + "/ctr/" + containerName + "/signal")
					s, _ := strconv.Atoi(string(data))
					if s > 0 {
						_ = syscall.Kill(command.Process.Pid, syscall.Signal(s))
					}
					time.Sleep(2 * time.Second)
				}
			}()

			cmdErr := command.Wait()

			if cmdErr == nil {
				exitCode = 0
			} else if exitError, ok := cmdErr.(*exec.ExitError); ok {
				if exitError.ExitCode() >= 0 {
					exitCode = exitError.ExitCode()
				} else {
					exitCode = 137 // SIGTERM
				}
			}

			if err := stderr.Close(); err != nil {
				return fmt.Errorf("failed to close stderr: %w", err)
			}
			if err := stdout.Close(); err != nil {
				return fmt.Errorf("failed to close stdout: %w", err)
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
				println("not saving outputs - not main container")
			}

			return cmdErr // this is the error returned from cmd.Wait(), which maybe an exitError
		},
	}
}

func saveArtifact(srcPath string) error {
	if common.FindOverlappingVolume(template, srcPath) != nil {
		println("no need to save artifact - on overlapping volume", srcPath)
		return nil
	}
	if _, err := os.Stat(srcPath); os.IsNotExist(err) { // might be optional, so we ignore
		println("cannot save artifact", srcPath, err)
		return nil
	}
	dstPath := varArgo + "/outputs/artifacts/" + srcPath + ".tgz"
	println(srcPath, "->", dstPath)
	z := filepath.Dir(dstPath)
	if err := os.MkdirAll(z, 0700); err != nil { // chmod rwx------
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
		println("no need to save parameter - on overlapping volume", srcPath)
		return nil
	}
	src, err := os.Open(srcPath)
	if os.IsNotExist(err) { // might be optional, so we ignore
		println("cannot save parameter", srcPath, err)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", srcPath, err)
	}
	defer func() { _ = src.Close() }()
	dstPath := varArgo + "/outputs/parameters/" + srcPath
	println(srcPath, "->", dstPath)
	z := filepath.Dir(dstPath)
	if err := os.MkdirAll(z, 0700); err != nil { // chmod rwx------
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
