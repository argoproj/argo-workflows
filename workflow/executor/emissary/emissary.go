package emissary

import (
	"compress/gzip"
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo/workflow/executor"
	execcommon "github.com/argoproj/argo/workflow/executor/common"
)

type emissary struct{}

func New() (executor.ContainerRuntimeExecutor, error) {
	return &emissary{}, nil
}

func (e emissary) GetFileContents(_ string, sourcePath string) (string, error) {
	sourceFile := filepath.Join("/var/argo/outputs", sourcePath+".tgz")
	log.Infof("%s -> ", sourceFile)
	src, err := os.Open(sourceFile)
	if err != nil {
		return "", err
	}
	defer func() { _ = src.Close() }()
	r, err := gzip.NewReader(src)
	defer func() { _ = r.Close() }()
	data, err := ioutil.ReadAll(r)
	return string(data), err
}

func (e emissary) CopyFile(_ string, sourcePath string, destPath string, _ int) error {
	// this implementation is very different, because we expect the emissary binary has already compressed the file
	// so no compression can or needs to be implemented here
	// TODO - warn the user we ignored compression
	sourceFile := filepath.Join("/var/argo/outputs", sourcePath+".tgz")
	log.Infof("%s -> %s", sourceFile, destPath)
	src, err := os.Open(sourceFile)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()
	_, err = io.Copy(dst, src)
	return err
}

func (e emissary) GetOutputStream(_ context.Context, _ string, _ bool) (io.ReadCloser, error) {
	return os.Open("/var/argo/stdout") // TODO - we could support if we wanted combinedOutput
}

func (e emissary) GetExitCode(_ context.Context, _ string) (string, error) {
	data, err := ioutil.ReadFile("/var/argo/exitcode")
	return string(data), err
}

func (e emissary) hasExited(ctx context.Context) bool {
	_, err := e.GetExitCode(ctx, "")
	return err == nil
}

func (e emissary) WaitInit() error {
	return nil
}

func (e emissary) Wait(ctx context.Context, _ string) error {
	t := time.NewTimer(3 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if e.hasExited(ctx) {
				return nil
			}
		}
	}
}

func (e emissary) Kill(ctx context.Context, _ []string) error {
	for _, signal := range []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL} {
		if err := ioutil.WriteFile("/var/argo/signal", []byte(strconv.Itoa(int(signal))), 0600); err != nil {
			return err
		}
		ctx, cancel := context.WithCancel(ctx)
		time.AfterFunc(execcommon.KillGracePeriod*time.Second, cancel)
		err := e.Wait(ctx, "")
		if err == nil {
			return nil
		} else if err != context.Canceled {
			return err
		}
	}
	panic("should not be possible")
}
