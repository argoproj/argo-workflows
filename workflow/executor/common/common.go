package common

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

const (
	containerShimPrefix = "://"
)

// killGracePeriod is the time in seconds after sending SIGTERM before
// forcefully killing the sidecar with SIGKILL (value matches k8s)
const KillGracePeriod = 10

// GetContainerID returns container ID of a ContainerStatus resource
func GetContainerID(container *v1.ContainerStatus) string {
	i := strings.Index(container.ContainerID, containerShimPrefix)
	if i == -1 {
		return ""
	}
	return container.ContainerID[i+len(containerShimPrefix):]
}

// KubernetesClientInterface is the interface to implement getContainerStatus method
type KubernetesClientInterface interface {
	GetContainerStatus(containerID string) (*v1.Pod, *v1.ContainerStatus, error)
	KillContainer(pod *v1.Pod, container *v1.ContainerStatus, sig syscall.Signal) error
	CreateArchive(containerID, sourcePath string) (*bytes.Buffer, error)
}

// WaitForTermination of the given containerID, set the timeout to 0 to discard it
func WaitForTermination(c KubernetesClientInterface, containerID string, timeout time.Duration) error {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	if timeout == 0 {
		timer.Stop()
	} else {
		defer timer.Stop()
	}

	log.Infof("Starting to wait completion of containerID %s ...", containerID)
	for {
		select {
		case <-ticker.C:
			_, containerStatus, err := c.GetContainerStatus(containerID)
			if err != nil {
				return err
			}
			if containerStatus.State.Terminated == nil {
				continue
			}
			log.Infof("ContainerID %q is terminated: %v", containerID, containerStatus.String())
			return nil
		case <-timer.C:
			return fmt.Errorf("timeout after %s", timeout.String())
		}
	}
}

// TerminatePodWithContainerID invoke the given SIG against the PID1 of the container.
// No-op if the container is on the hostPID
func TerminatePodWithContainerID(c KubernetesClientInterface, containerID string, sig syscall.Signal) error {
	pod, container, err := c.GetContainerStatus(containerID)
	if err != nil {
		return err
	}
	if container.State.Terminated != nil {
		log.Infof("Container %s is already terminated: %v", container.ContainerID, container.State.Terminated.String())
		return nil
	}
	if pod.Spec.HostPID {
		return fmt.Errorf("cannot terminate a hostPID Pod %s", pod.Name)
	}
	if pod.Spec.RestartPolicy != "Never" {
		return fmt.Errorf("cannot terminate pod with a %q restart policy", pod.Spec.RestartPolicy)
	}
	return c.KillContainer(pod, container, sig)
}

// KillGracefully kills a container gracefully.
func KillGracefully(c KubernetesClientInterface, containerID string) error {
	log.Infof("SIGTERM containerID %q: %s", containerID, syscall.SIGTERM.String())
	err := TerminatePodWithContainerID(c, containerID, syscall.SIGTERM)
	if err != nil {
		return err
	}
	err = WaitForTermination(c, containerID, time.Second*KillGracePeriod)
	if err == nil {
		log.Infof("ContainerID %q successfully killed", containerID)
		return nil
	}
	log.Infof("SIGKILL containerID %q: %s", containerID, syscall.SIGKILL.String())
	err = TerminatePodWithContainerID(c, containerID, syscall.SIGKILL)
	if err != nil {
		return err
	}
	err = WaitForTermination(c, containerID, time.Second*KillGracePeriod)
	if err != nil {
		return err
	}
	log.Infof("ContainerID %q successfully killed", containerID)
	return nil
}

// CopyArchive downloads files and directories as a tarball and saves it to a specified path.
func CopyArchive(c KubernetesClientInterface, containerID, sourcePath, destPath string) error {
	log.Infof("Archiving %s:%s to %s", containerID, sourcePath, destPath)
	b, err := c.CreateArchive(containerID, sourcePath)
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
