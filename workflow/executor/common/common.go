package common

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

// KubernetesClientInterface is the interface to implement getContainerStatus method
type KubernetesClientInterface interface {
	GetContainerStatus(ctx context.Context, containerName string) (*v1.Pod, *v1.ContainerStatus, error)
	GetContainerStatuses(ctx context.Context) (*v1.Pod, []v1.ContainerStatus, error)
	CreateArchive(ctx context.Context, containerName, sourcePath string) (*bytes.Buffer, error)
}

// WaitForTermination of the given containerName, set the timeout to 0 to discard it
func WaitForTermination(ctx context.Context, c KubernetesClientInterface, containerNames []string, timeout time.Duration) error {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	timer := time.NewTimer(timeout)
	if timeout == 0 {
		if !timer.Stop() {
			<-timer.C
		}
	} else {
		defer timer.Stop()
	}
	log.Infof("Starting to wait completion of containers %s...", strings.Join(containerNames, ","))
	for {
		select {
		case <-ticker.C:
			done, err := isTerminated(ctx, c, containerNames)
			if err != nil {
				return err
			} else if done {
				return nil
			}
		case <-timer.C:
			return fmt.Errorf("timeout after %s", timeout.String())
		}
	}
}

func isTerminated(ctx context.Context, c KubernetesClientInterface, containerNames []string) (bool, error) {
	_, containerStatus, err := c.GetContainerStatuses(ctx)
	if err != nil {
		return false, err
	}
	return AllTerminated(containerStatus, containerNames), nil
}

func AllTerminated(containerStatuses []v1.ContainerStatus, containerNames []string) bool {
	terminated := make(map[string]bool)
	// I've seen a few cases where containers are missing from container status just after a pod started.
	for _, c := range containerStatuses {
		terminated[c.Name] = c.State.Terminated != nil
	}
	for _, n := range containerNames {
		if !terminated[n] {
			return false
		}
	}
	return true
}

// CopyArchive downloads files and directories as a tarball and saves it to a specified path.
func CopyArchive(ctx context.Context, c KubernetesClientInterface, containerName, sourcePath, destPath string) error {
	log.Infof("Archiving %s:%s to %s", containerName, sourcePath, destPath)
	b, err := c.CreateArchive(ctx, containerName, sourcePath)
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
	return err
}
