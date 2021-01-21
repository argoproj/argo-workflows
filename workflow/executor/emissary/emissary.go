package emissary

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/executor"
	execcommon "github.com/argoproj/argo/workflow/executor/common"
	"github.com/argoproj/argo/workflow/util/path"
)

type emissary struct {
	template *wfv1.Template
}

func New(t *wfv1.Template) (executor.ContainerRuntimeExecutor, error) {
	return &emissary{t}, nil
}

func (e *emissary) Init() error {
	if err := copyEmissaryBinary(); err != nil {
		return err
	}
	if err := e.writeTemplate(); err != nil {
		return err
	}
	return nil
}

func copyEmissaryBinary() error {
	name, err := path.Search("emissary")
	if err != nil {
		return err
	}
	in, err := os.Open(name)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile("/var/argo/emissary", os.O_RDWR|os.O_CREATE, 0500) // r-x------
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	return err
}

func (e emissary) writeTemplate() error {
	data, err := json.Marshal(e.template)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("/var/argo/template", data, 0400) // chmod -r--------
}

func (e emissary) GetFileContents(_ string, sourcePath string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join("/var/argo/outputs/parameters", sourcePath))
	return string(data), err
}

func (e emissary) CopyFile(_ string, sourcePath string, destPath string, _ int) error {
	// this implementation is very different, because we expect the emissary binary has already compressed the file
	// so no compression can or needs to be implemented here
	// TODO - warn the user we ignored compression
	sourceFile := filepath.Join("/var/argo/outputs/artifacts", sourcePath+".tgz")
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

type multiReaderCloser struct {
	io.Reader
	closer []io.Closer
}

func (m *multiReaderCloser) Close() error {
	for _, c := range m.closer {
		if err := c.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (e emissary) GetOutputStream(_ context.Context, _ string, combinedOutput bool) (io.ReadCloser, error) {
	stdout, err := os.Open("/var/argo/stdout")
	if !combinedOutput {
		return stdout, err
	}
	if err != nil {
		return nil, err
	}
	stderr, err := os.Open("/var/argo/stderr")
	if err != nil {
		return nil, err
	}
	return &multiReaderCloser{Reader: io.MultiReader(stdout, stderr), closer: []io.Closer{stdout, stderr}}, err
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
	t := time.NewTicker(2 * time.Second)
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
	if err := ioutil.WriteFile("/var/argo/signal", []byte(strconv.Itoa(int(syscall.SIGTERM))), 0600); err != nil {
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
	return ioutil.WriteFile("/var/argo/signal", []byte(strconv.Itoa(int(syscall.SIGKILL))), 0600)
}
