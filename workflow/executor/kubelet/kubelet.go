package kubelet

import (
	"io"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo/errors"
)

type KubeletExecutor struct {
	cli *kubeletClient
}

func NewKubeletExecutor() (*KubeletExecutor, error) {
	log.Infof("Creating a kubelet executor")
	cli, err := newKubeletClient()
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &KubeletExecutor{
		cli: cli,
	}, nil
}

func (k *KubeletExecutor) GetFileContents(containerID string, sourcePath string) (string, error) {
	return "", errors.Errorf(errors.CodeNotImplemented, "GetFileContents() is not implemented in the kubelet executor.")
}

func (k *KubeletExecutor) CopyFile(containerID string, sourcePath string, destPath string) error {
	return errors.Errorf(errors.CodeNotImplemented, "CopyFile() is not implemented in the kubelet executor.")
}

func (k *KubeletExecutor) GetOutputStream(containerID string, combinedOutput bool) (io.ReadCloser, error) {
	if !combinedOutput {
		log.Warn("non combined output unsupported")
	}
	return k.cli.GetLogStream(containerID)
}

func (k *KubeletExecutor) WaitInit() error {
	return nil
}

// Wait for the container to complete
func (k *KubeletExecutor) Wait(containerID string) error {
	return k.cli.WaitForTermination(containerID, 0)
}

// Kill kills a list of containerIDs first with a SIGTERM then with a SIGKILL after a grace period
func (k *KubeletExecutor) Kill(containerIDs []string) error {
	for _, containerID := range containerIDs {
		err := k.cli.KillGracefully(containerID)
		if err != nil {
			return err
		}
	}
	return nil
}
