package entrypoint

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/argoproj/argo/util/archive"
	"github.com/argoproj/argo/workflow/executor"
	execcommon "github.com/argoproj/argo/workflow/executor/common"
)

/**

* The controller replaces the command with `/var/argo/entrypoint ${command}`.
* The init container copies `entrypoint` to `/var/argo/`.
* Entrypoint binary runs the original command as a sub-process, capturing stdout/stderr
* If it `signal` appears in `/var/argo/` than the signal from that file is sent to the sub-process
* On completion it writes `exitcode`, `stdout` and `stderr` to `/var/argo/`.
* If there are parameters or artifacts that needs copying, they're copied to `/var/argo/outputs/${path}`.
* This executor uses those files to coordinate.
 */
type EntrypointExecutor struct{}

func NewEntrypointExecutor() (executor.ContainerRuntimeExecutor, error) {
	return &EntrypointExecutor{}, nil
}

func (i EntrypointExecutor) GetFileContents(_ string, sourcePath string) (string, error) {
	data, err := ioutil.ReadFile(sourcePath)
	return string(data), err
}

func (i EntrypointExecutor) CopyFile(_ string, sourcePath string, destPath string, compressionLevel int) error {
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	return archive.TarGzToWriter(filepath.Join("/var/argo/outputs", sourcePath), compressionLevel, destFile)
}

func (i EntrypointExecutor) GetOutputStream(_ context.Context, _ string, combinedOutput bool) (io.ReadCloser, error) {
	return os.Open("/var/argo/stdout") // TODO - combinedOutput
}

func (i EntrypointExecutor) GetExitCode(_ context.Context, _ string) (string, error) {
	data, err := ioutil.ReadFile("/var/argo/exitcode")
	return string(data), err
}

func (i EntrypointExecutor) hasExited(ctx context.Context) bool {
	_, err := i.GetExitCode(ctx, "")
	return err == nil
}

func (i EntrypointExecutor) WaitInit() error {
	return nil
}

func (i EntrypointExecutor) Wait(ctx context.Context, _ string) error {
	t := time.NewTimer(3 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if i.hasExited(ctx) {
				return nil
			}
		}
	}
}

func (i EntrypointExecutor) Kill(ctx context.Context, _ []string) error {
	for _, signal := range []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL} {
		if err := ioutil.WriteFile("/var/argo/signal", []byte(strconv.Itoa(int(signal))), 0600); err != nil {
			return err
		}
		ctx, cancel := context.WithCancel(ctx)
		time.AfterFunc(execcommon.KillGracePeriod*time.Second, cancel)

		err := i.Wait(ctx, "")
		if err == nil {
			return nil
		} else if err != context.Canceled {
			return err
		}
	}
	panic("should not be possible")
}
