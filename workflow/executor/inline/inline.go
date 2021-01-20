package inline

import (
	"context"
	"io"
	"io/ioutil"
	"os"

	"github.com/argoproj/argo/util/archive"
	"github.com/argoproj/argo/workflow/executor"
)

type inline struct{}

func NewInlineExecutor() (executor.ContainerRuntimeExecutor, error) {
	return &inline{}, nil
}

func (i inline) GetFileContents(_ string, sourcePath string) (string, error) {
	data, err := ioutil.ReadFile(sourcePath)
	return string(data), err
}

func (i inline) CopyFile(_ string, sourcePath string, destPath string, compressionLevel int) error {
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	return archive.TarGzToWriter(sourcePath, compressionLevel, destFile)
}

func (i inline) GetOutputStream(_ context.Context, _ string, combinedOutput bool) (io.ReadCloser, error) {
	return os.Open("/var/argo/stdout") // TODO - combinedOutput
}

func (i inline) GetExitCode(_ context.Context, _ string) (string, error) {
	data, err := ioutil.ReadFile("/var/argo/exitcode")
	return string(data), err
}

func (i inline) WaitInit() error {
	panic("not supported")
}

func (i inline) Wait(context.Context, string) error {
	panic("not supported")
}

func (i inline) Kill(context.Context, []string) error {
	panic("not supported")
}
