package k8sapi

import (
	"compress/gzip"
	"os"
	"syscall"
	"time"

	"github.com/argoproj/argo/errors"
	log "github.com/sirupsen/logrus"
)

// killGracePeriod is the time in seconds after sending SIGTERM before
// forcefully killing the sidecar with SIGKILL (value matches k8s)
const killGracePeriod = 10

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
	log.Infof("Archiving %s:%s to %s", containerID, sourcePath, destPath)
	b, err := k.client.createArchive(containerID, sourcePath)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	w := gzip.NewWriter(f)
	_, err = w.Write(b.Bytes())
	if err != nil {
		return err
	}
	err = w.Flush()
	if err != nil {
		return err
	}
	err = w.Close()
	if err != nil {
		return err
	}
	return nil
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
		log.Infof("SIGTERM containerID %q ...", containerID, syscall.SIGTERM.String())
		err := k.client.terminatePodWithContainerID(containerID, syscall.SIGTERM)
		if err != nil {
			return err
		}
		err = k.client.waitForTermination(containerID, time.Second*killGracePeriod)
		if err == nil {
			log.Infof("ContainerID %q successfully killed", containerID)
			continue
		}
		log.Infof("SIGKILL containerID %q ...", containerID, syscall.SIGKILL.String())
		err = k.client.terminatePodWithContainerID(containerID, syscall.SIGKILL)
		if err != nil {
			return err
		}
		err = k.client.waitForTermination(containerID, time.Second*killGracePeriod)
		if err != nil {
			return err
		}
		log.Infof("ContainerID %q successfully killed", containerID)
	}
	return nil
}
