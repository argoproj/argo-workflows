package kubelet

import (
	"context"
	"io"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo-workflows/v3/errors"
)

type KubeletExecutor struct {
	cli *kubeletClient
}

func NewKubeletExecutor(namespace, podName string) (*KubeletExecutor, error) {
	log.Infof("Creating a kubelet executor")
	cli, err := newKubeletClient(namespace, podName)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &KubeletExecutor{
		cli: cli,
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

// Wait for the container to complete
func (k *KubeletExecutor) Wait(ctx context.Context, containerNames []string) error {
	return k.cli.WaitForTermination(ctx, containerNames, 0)
}

// Kill kills a list of containers first with a SIGTERM then with a SIGKILL after a grace period
func (k *KubeletExecutor) Kill(ctx context.Context, containerNames []string, terminationGracePeriodDuration time.Duration) error {
	return k.cli.KillGracefully(ctx, containerNames, terminationGracePeriodDuration)
}

func (k *KubeletExecutor) ListContainerNames(ctx context.Context) ([]string, error) {
	pod, err := k.cli.getPod()
	if err != nil {
		return nil, err
	}
	var containerNames []string
	for _, c := range pod.Status.ContainerStatuses {
		containerNames = append(containerNames, c.Name)
	}
	return containerNames, nil
}
