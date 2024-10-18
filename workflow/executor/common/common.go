package common

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	envutil "github.com/argoproj/argo-workflows/v3/util/env"
)

const (
	containerShimPrefix = "://"
)

// GetContainerID returns container ID of a ContainerStatus resource
func GetContainerID(container string) string {
	i := strings.Index(container, containerShimPrefix)
	if i == -1 {
		return container
	}
	return container[i+len(containerShimPrefix):]
}

// KubernetesClientInterface is the interface to implement getContainerStatus method
type KubernetesClientInterface interface {
	GetContainerStatuses(ctx context.Context) (*v1.Pod, []v1.ContainerStatus, error)
	KillContainer(pod *v1.Pod, container *v1.ContainerStatus, sig syscall.Signal) error
	CreateArchive(ctx context.Context, containerName, sourcePath string) (*bytes.Buffer, error)
}

// WaitForTermination of the given containerName, set the timeout to 0 to discard it
func WaitForTermination(ctx context.Context, c KubernetesClientInterface, containerNames []string, timeout time.Duration) error {
	ticker := time.NewTicker(envutil.LookupEnvDurationOr("WAIT_CONTAINER_STATUS_CHECK_INTERVAL", time.Second*5))
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

// TerminatePodWithContainerNames invoke the given SIG against the PID1 of the container.
// No-op if the container is on the hostPID
func TerminatePodWithContainerNames(ctx context.Context, c KubernetesClientInterface, containerNames []string, sig syscall.Signal) error {
	pod, containerStatuses, err := c.GetContainerStatuses(ctx)
	if err != nil {
		return err
	}
	for _, s := range containerStatuses {
		if !slices.Contains(containerNames, s.Name) {
			continue
		}
		if s.State.Terminated != nil {
			log.Infof("Container %s is already terminated: %v", s.Name, s.State.Terminated.String())
			continue
		}
		if pod.Spec.ShareProcessNamespace != nil && *pod.Spec.ShareProcessNamespace {
			return fmt.Errorf("cannot terminate a process-namespace-shared Pod %s", pod.Name)
		}
		if pod.Spec.HostPID {
			return fmt.Errorf("cannot terminate a hostPID Pod %s", pod.Name)
		}
		if pod.Spec.RestartPolicy != "Never" {
			return fmt.Errorf("cannot terminate pod with a %q restart policy", pod.Spec.RestartPolicy)
		}
		err := c.KillContainer(pod, &s, sig)
		if err != nil {
			return err
		}
	}
	return nil
}

// KillGracefully kills a container gracefully.
func KillGracefully(ctx context.Context, c KubernetesClientInterface, containerNames []string, terminationGracePeriodDuration time.Duration) error {
	log.Infof("SIGTERM containers %s: %s", strings.Join(containerNames, ","), syscall.SIGTERM.String())
	err := TerminatePodWithContainerNames(ctx, c, containerNames, syscall.SIGTERM)
	if err != nil {
		return err
	}
	err = WaitForTermination(ctx, c, containerNames, terminationGracePeriodDuration)
	if err == nil {
		log.Infof("Containers %s successfully killed", strings.Join(containerNames, ","))
		return nil
	}
	log.Infof("SIGKILL containers %s: %s", strings.Join(containerNames, ","), syscall.SIGKILL.String())
	err = TerminatePodWithContainerNames(ctx, c, containerNames, syscall.SIGKILL)
	if err != nil {
		return err
	}
	err = WaitForTermination(ctx, c, containerNames, terminationGracePeriodDuration)
	if err != nil {
		return err
	}
	log.Infof("Containers %s successfully killed", strings.Join(containerNames, ","))
	return nil
}

// CopyArchive downloads files and directories as a tarball and saves it to a specified path.
func CopyArchive(ctx context.Context, c KubernetesClientInterface, containerName, sourcePath, destPath string) error {
	log.Infof("Archiving %s:%s to %s", containerName, sourcePath, destPath)
	b, err := c.CreateArchive(ctx, containerName, sourcePath)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Clean(destPath), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
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
