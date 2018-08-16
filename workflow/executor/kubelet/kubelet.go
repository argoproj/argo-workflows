package kubelet

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
const killGracePeriod = 30

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
	log.Infof("Archiving %s:%s to %s", containerID, sourcePath, destPath)
	b, err := k.cli.GetFileContents(containerID, sourcePath)
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
	return f.Close()
}

// GetOutput returns the entirety of the container output as a string
// Used to capturing script results as an output parameter
func (k *KubeletExecutor) GetOutput(containerID string) (string, error) {
	return k.cli.GetContainerLogs(containerID)
}

// Wait for the container to complete
func (k *KubeletExecutor) Wait(containerID string) error {
	return k.cli.WaitForTermination(containerID, 0)
}

// killContainers kills a list of containerIDs first with a SIGTERM then with a SIGKILL after a grace period
func (k *KubeletExecutor) Kill(containerIDs []string) error {
	for _, containerID := range containerIDs {
		log.Infof("SIGTERM containerID %q ...", containerID, syscall.SIGTERM.String())
		err := k.cli.TerminatePodWithContainerID(containerID, syscall.SIGTERM)
		if err != nil {
			return err
		}
		err = k.cli.WaitForTermination(containerID, time.Second*killGracePeriod)
		if err == nil {
			log.Infof("ContainerID %q successfully killed", containerID)
			continue
		}
		log.Infof("SIGKILL containerID %q ...", containerID, syscall.SIGKILL.String())
		err = k.cli.TerminatePodWithContainerID(containerID, syscall.SIGKILL)
		if err != nil {
			return err
		}
		err = k.cli.WaitForTermination(containerID, time.Second*killGracePeriod)
		if err != nil {
			return err
		}
		log.Infof("ContainerID %q successfully killed", containerID)
	}
	return nil
}
