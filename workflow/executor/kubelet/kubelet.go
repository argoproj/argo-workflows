package kubelet

import (
	"context"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo/v2/errors"
)

type KubeletExecutor struct {
	cli     *kubeletClient
	podName string
}

func NewKubeletExecutor(namespace, podName string) (*KubeletExecutor, error) {
	log.Infof("Creating a kubelet executor")
	cli, err := newKubeletClient(namespace, podName)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &KubeletExecutor{
		cli:     cli,
		podName: podName,
	}, nil
}

func (k *KubeletExecutor) GetFileContents(containerName string, sourcePath string) (string, error) {
	return "", errors.Errorf(errors.CodeNotImplemented, "GetFileContents() is not implemented in the kubelet executor.")
}

func (k *KubeletExecutor) CopyFile(containerName string, sourcePath string, destPath string, compressionLevel int) error {
	return errors.Errorf(errors.CodeNotImplemented, "CopyFile() is not implemented in the kubelet executor.")
}

func (k *KubeletExecutor) GetOutputStream(ctx context.Context, containerName string, combinedOutput bool) (io.ReadCloser, error) {
	if !combinedOutput {
		log.Warn("non combined output unsupported")
	}
	return k.cli.GetLogStream(containerName)
}

func (k *KubeletExecutor) GetExitCode(ctx context.Context, containerName string) (string, error) {
	log.Infof("Getting exit code of %s", containerName)
	_, status, err := k.cli.GetContainerStatus(ctx, containerName)
	if err != nil {
		return "", errors.InternalWrapError(err, "Could not get container status")
	}
	if status != nil && status.State.Terminated != nil {
		return fmt.Sprint(status.State.Terminated.ExitCode), nil
	}
	return "", nil
}

// Wait for the container to complete
func (k *KubeletExecutor) Wait(ctx context.Context) error {
	return k.cli.WaitForTermination(ctx, "main", 0)
}

// Kill kills a list of containers first with a SIGTERM then with a SIGKILL after a grace period
func (k *KubeletExecutor) Kill(ctx context.Context, containerNames []string) error {
	for _, containerName := range containerNames {
		err := k.cli.KillGracefully(ctx, containerName)
		if err != nil {
			return err
		}
	}
	return nil
}
