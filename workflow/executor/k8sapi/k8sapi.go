package k8sapi

import (
	"github.com/argoproj/argo/errors"
	log "github.com/sirupsen/logrus"
)

type K8sAPIExecutor struct {
	client *k8sAPIClient
}

func NewK8sAPIExecutor() (*K8sAPIExecutor, error) {
	log.Infof("Creating a K8sAPI executor")
	client, err := newK8sAPIClient()
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &K8sAPIExecutor{
		client: client,
	}, nil
}

func (k *K8sAPIExecutor) GetFileContents(containerID string, sourcePath string) (string, error) {
	log.Infof("Getting file contents of %s:%s", containerID, sourcePath)
	return k.client.getFileContents(containerID, sourcePath)
}

func (k *K8sAPIExecutor) CopyFile(containerID string, sourcePath string, destPath string) error {
	return k.client.copyArchive(containerID, sourcePath, destPath)
}

// GetOutput returns the entirety of the container output as a string
// Used to capturing script results as an output parameter
func (k *K8sAPIExecutor) GetOutput(containerID string) (string, error) {
	log.Infof("Getting output of %s", containerID)
	return k.client.getLogs(containerID)
}

// Logs copies logs to a given path
func (k *K8sAPIExecutor) Logs(containerID, path string) error {
	log.Infof("Saving output of %s to %s", containerID, path)
	return k.client.saveLogs(containerID, path)
}

// Wait for the container to complete
func (k *K8sAPIExecutor) Wait(containerID string) error {
	log.Infof("Waiting for container %s to complete", containerID)
	return k.client.waitForTermination(containerID, 0)
}

// Kill kills a list of containerIDs first with a SIGTERM then with a SIGKILL after a grace period
func (k *K8sAPIExecutor) Kill(containerIDs []string) error {
	log.Infof("Killing containers %s", containerIDs)
	for _, containerID := range containerIDs {
		err := k.client.killGracefully(containerID)
		if err != nil {
			return err
		}
	}
	return nil
}
