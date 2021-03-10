package emissary

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/executor"
)

/*
This executor works very differently to the others. It mounts an empty-dir on all containers at `/var/run/argo`. The main container command is replaces by a new binary `argoexec` which starts the original command in a sub-process and when it is finished, captures the outputs:

The init container creates these files:

* `/var/run/argo/argoexec` The binary, copied from the `argoexec` image.
* `/var/run/argo/template` A JSON encoding of the template.

In the main container, the emissary creates these files:

* `/var/run/argo/ctr/${containerName}/exitcode` The container exit code.
* `/var/run/argo/ctr/${containerName}/stderr` A copy of stderr (if needed).
* `/var/run/argo/ctr/${containerName}/stdout`  A copy of stdout (if needed).

If the container is named `main` it also copies base-layer artifacts to the shared volume:

* `/var/run/argo/outputs/parameters/${path}` All output parameters are copied here, e.g. `/tmp/message` is moved to `/var/run/argo/outputs/parameters/tmp/message`.
* `/var/run/argo/outputs/artifacts/${path}.tgz` All output artifacts are copied here, e.g. `/tmp/message` is moved to /var/run/argo/outputs/artifacts/tmp/message.tgz`.

*/
type emissary struct{}

func New() (executor.ContainerRuntimeExecutor, error) {
	return &emissary{}, nil
}

func (e *emissary) Init(t wfv1.Template) error {
	if err := copyBinary(); err != nil {
		return err
	}
	if err := e.writeTemplate(t); err != nil {
		return err
	}
	return nil
}

func (e emissary) writeTemplate(t wfv1.Template) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("/var/run/argo/template", data, 0400) // chmod -r--------
}

func (e emissary) GetFileContents(_ string, sourcePath string) (string, error) {
	data, err := ioutil.ReadFile(filepath.Join("/var/run/argo/outputs/parameters", sourcePath))
	return string(data), err
}

func (e emissary) CopyFile(_ string, sourcePath string, destPath string, _ int) error {
	// this implementation is very different, because we expect the emissary binary has already compressed the file
	// so no compression can or needs to be implemented here
	// TODO - warn the user we ignored compression?
	sourceFile := filepath.Join("/var/run/argo/outputs/artifacts", sourcePath+".tgz")
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
	if err := dst.Close(); err != nil {
		return err
	}
	return err
}

func (e emissary) GetOutputStream(_ context.Context, containerName string, combinedOutput bool) (io.ReadCloser, error) {
	names := []string{"stdout"}
	if combinedOutput {
		names = append(names, "stderr")
	}
	var files []io.ReadCloser
	for _, name := range names {
		f, err := os.Open("/var/run/argo/ctr/" + containerName + "/" + name)
		if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return newMultiReaderCloser(files...), nil
}

func (e emissary) GetExitCode(_ context.Context, containerName string) (string, error) {
	data, err := ioutil.ReadFile("/var/run/argo/ctr/" + containerName + "/exitcode")
	if err != nil {
		return "", err
	}
	exitCode, err := strconv.Atoi(string(data))
	return strconv.Itoa(exitCode), err
}

func (e emissary) Wait(ctx context.Context, containerNames []string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if e.isComplete(containerNames) {
				return nil
			}
			time.Sleep(3 * time.Second)
		}
	}
}

func (e emissary) isComplete(containerNames []string) bool {
	for _, containerName := range containerNames {
		_, err := os.Stat("/var/run/argo/ctr/" + containerName + "/exitcode")
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
