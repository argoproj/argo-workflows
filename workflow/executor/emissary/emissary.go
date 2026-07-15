package emissary

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/file"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
)

/*
This executor works very differently to the others. It mounts an empty-dir on all containers at `/var/run/argo`. The main container command is replaces by a new binary `argoexec` which starts the original command in a sub-process and when it is finished, captures the outputs:

The argoexec binary and the template are delivered differently depending on the pod layout:

* Legacy layout: the init container creates `/var/run/argo/argoexec` (the binary, copied from the `argoexec` image) and `/var/run/argo/template` (a JSON encoding of the template).
* Init-less layout (initlessPod enabled): there is no init container. The binary is mounted into containers from the `argoexec` image volume at `/argo-bin/bin/argoexec`, and the `supervisor` container — which runs concurrently with `main` — writes `/var/run/argo/template` and signals progress via a single `/var/run/argo/status` marker whose first line is a state token (`RUNNING` heartbeat, `READY` on success, or `FAILED` plus the reason on pre-main failure). Main's emissary blocks on that marker before reading the template, and treats a stale heartbeat as a dead supervisor.

In the main container, the emissary creates these files:

* `/var/run/argo/ctr/${containerName}/exitcode` The container exit code.
* `/var/run/argo/ctr/${containerName}/combined` A copy of stdout+stderr (if needed).
* `/var/run/argo/ctr/${containerName}/stdout`  A copy of stdout (if needed).

If the container is named `main` it also copies base-layer artifacts to the shared volume:

* `/var/run/argo/outputs/parameters/${path}` All output parameters are copied here, e.g. `/tmp/message` is moved to `/var/run/argo/outputs/parameters/tmp/message`.
* `/var/run/argo/outputs/artifacts/${path}.tgz` All output artifacts are copied here, e.g. `/tmp/message` is moved to /var/run/argo/outputs/artifacts/tmp/message.tgz`.

The auxiliary container (`wait` in the legacy layout, `supervisor` in the init-less layout) can create one file itself, used for terminating the sub-process:

* `/var/run/argo/ctr/${containerName}/signal` The emissary binary listens to changes in this file, and signals the sub-process with the value found in this file.
*/
type emissary struct{}

func New() (executor.ContainerRuntimeExecutor, error) {
	return &emissary{}, nil
}

func (e *emissary) Init(t wfv1.Template) error {
	osspecific.AllowGrantingAccessToEveryone()
	if err := copyBinary(); err != nil {
		return err
	}
	return e.WriteTemplate(t)
}

// WriteTemplate writes the template JSON to the shared volume. It is the
// subset of Init that the init-less supervisor needs — the argoexec binary
// is delivered via a Kubernetes image volume, so no copy step is required.
func (e emissary) WriteTemplate(t wfv1.Template) error {
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return os.WriteFile(common.VarRunArgoPath+"/template", data, 0o444) // chmod -r--r--r--
}

func (e emissary) GetFileContents(_ string, sourcePath string) (string, error) {
	data, err := os.ReadFile(filepath.Clean(filepath.Join(common.VarRunArgoPath, "outputs", "parameters", sourcePath)))
	return string(data), err
}

func (e emissary) CopyFile(ctx context.Context, containerName string, sourcePath string, destPath string, _ int) error {
	// this implementation is very different, because we expect the emissary binary has already compressed the file
	// so no compression can or needs to be implemented here
	// TODO - warn the user we ignored compression?
	sourceFile := filepath.Join(common.VarRunArgoPath, "outputs", "artifacts", strings.TrimSuffix(sourcePath, "/")+".tgz")
	logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"source": sourceFile, "dest": destPath}).Info(ctx, "Copying file")
	src, err := os.Open(filepath.Clean(sourceFile))
	if err != nil {
		// If compressed file does not exist then the source artifact did not exist
		// and we throw an Argo NotFound error to handle optional artifacts upstream
		if os.IsNotExist(err) {
			return argoerrors.New(argoerrors.CodeNotFound, err.Error())
		}
		return err
	}
	defer func() { _ = src.Close() }()
	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()
	_, err = io.Copy(dst, src)
	if closeErr := dst.Close(); closeErr != nil {
		return closeErr
	}
	return err
}

func (e emissary) GetOutputStream(_ context.Context, containerName string, combinedOutput bool) (io.ReadCloser, error) {
	name := "stdout"
	if combinedOutput {
		name = "combined"
	}
	return os.Open(filepath.Clean(filepath.Join(common.VarRunArgoPath, "ctr", containerName, name)))
}

func (e emissary) Wait(ctx context.Context, containerNames []string) error {
	// Zero the umask so MkdirAll below creates the directory with mode
	// 0o777 — peer containers may run as different users and need to
	// write exit code / log files inside it.
	osspecific.AllowGrantingAccessToEveryone()
	exitCodePaths := make([]string, 0, len(containerNames))
	for _, containerName := range containerNames {
		dir := filepath.Join(common.VarRunArgoPath, "ctr", containerName)
		// The peer container will MkdirAll this directory too, but it may
		// not have started yet; pre-creating it lets us install the inotify
		// watch on the parent immediately.
		if err := os.MkdirAll(dir, 0o777); err != nil {
			return err
		}
		exitCodePaths = append(exitCodePaths, filepath.Join(dir, "exitcode"))
	}
	g, gctx := errgroup.WithContext(ctx)
	for _, exitCodePath := range exitCodePaths {
		g.Go(func() error { return file.WaitForCreate(gctx, exitCodePath) })
	}
	return g.Wait()
}

func (e emissary) Kill(ctx context.Context, containerNames []string, terminationGracePeriodDuration time.Duration) error {
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(logging.Fields{"terminationGracePeriodDuration": terminationGracePeriodDuration, "containerNames": containerNames}).Info(ctx, "emissary: killing containers")
	for _, containerName := range containerNames {
		// allow write-access by other users, because other containers
		// should delete the signal after receiving it
		signalPath := filepath.Join(common.VarRunArgoPath, "ctr", containerName, "signal")
		signalDir := filepath.Dir(signalPath)
		logger.WithFields(logging.Fields{
			"containerName": containerName,
			"signalPath":    signalPath,
		}).Debug(ctx, "Sending SIGTERM to container")
		if err := os.MkdirAll(signalDir, 0o777); err != nil {
			logger.WithField("signalDir", signalDir).WithError(err).Error(ctx, "failed to create signal directory")
			return err
		}
		if err := os.WriteFile(signalPath, []byte(strconv.Itoa(int(syscall.SIGTERM))), 0o666); err != nil {
			return err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, terminationGracePeriodDuration)
	defer cancel()
	err := e.Wait(ctx, containerNames)
	if err == nil {
		return nil
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	for _, containerName := range containerNames {
		// allow write-access by other users, because other containers
		// should delete the signal after receiving it
		signalPath := filepath.Join(common.VarRunArgoPath, "ctr", containerName, "signal")
		logger.WithFields(logging.Fields{
			"containerName": containerName,
			"signalPath":    signalPath,
		}).Debug(ctx, "Sending SIGKILL to container")
		if err := os.WriteFile(signalPath, []byte(strconv.Itoa(int(syscall.SIGKILL))), 0o666); err != nil {
			return err
		}
	}
	// Old context has expired; detach from its cancellation so the SIGKILL
	// wait outlives the grace period, but preserve trace/logger values for
	// observability during incident debugging.
	ctx, cancel = context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return e.Wait(ctx, containerNames)
}
