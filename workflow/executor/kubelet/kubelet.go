package kubelet

import (
	"github.com/argoproj/argo/errors"
	log "github.com/sirupsen/logrus"
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
	b, err := k.cli.GetFileContents(containerID, sourcePath)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func (k *KubeletExecutor) CopyFile(containerID string, sourcePath string, destPath string) error {
	return k.cli.CopyArchive(containerID, sourcePath, destPath)
}

// GetOutput returns the entirety of the container output as a string
// Used to capturing script results as an output parameter
func (k *KubeletExecutor) GetOutput(containerID string) (string, error) {
	return k.cli.GetContainerLogs(containerID)
}

// Logs copies logs to a given path
func (k *KubeletExecutor) Logs(containerID, path string) error {
	return k.cli.SaveLogsToFile(containerID, path)
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
