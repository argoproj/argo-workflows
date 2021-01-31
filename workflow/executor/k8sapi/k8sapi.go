package k8sapi

import (
	"context"
	"fmt"
	"io"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"github.com/argoproj/argo/v2/errors"
)

type K8sAPIExecutor struct {
	client *k8sAPIClient
}

func NewK8sAPIExecutor(clientset kubernetes.Interface, config *restclient.Config, podName, namespace string) (*K8sAPIExecutor, error) {
	log.Infof("Creating a K8sAPI executor")
	client, err := newK8sAPIClient(clientset, config, podName, namespace)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &K8sAPIExecutor{
		client: client,
	}, nil
}

func (k *K8sAPIExecutor) GetFileContents(containerName string, sourcePath string) (string, error) {
	return "", errors.Errorf(errors.CodeNotImplemented, "GetFileContents() is not implemented in the k8sapi executor.")
}

func (k *K8sAPIExecutor) CopyFile(containerName string, sourcePath string, destPath string, compressionLevel int) error {
	return errors.Errorf(errors.CodeNotImplemented, "CopyFile() is not implemented in the k8sapi executor.")
}

func (k *K8sAPIExecutor) GetOutputStream(ctx context.Context, containerName string, combinedOutput bool) (io.ReadCloser, error) {
	log.Infof("Getting output of %s", containerName)
	if !combinedOutput {
		log.Warn("non combined output unsupported")
	}
	return k.client.getLogsAsStream(ctx, containerName)
}

func (k *K8sAPIExecutor) GetExitCode(ctx context.Context, containerName string) (string, error) {
	log.Infof("Getting exit code of %s", containerName)
	_, status, err := k.client.GetContainerStatus(ctx, containerName)
	if err != nil {
		return "", errors.InternalWrapError(err, "Could not get container status")
	}
	if status != nil && status.State.Terminated != nil {
		return fmt.Sprint(status.State.Terminated.ExitCode), nil
	}
	return "", nil
}

// Wait for the container to complete
func (k *K8sAPIExecutor) Wait(ctx context.Context) error {
	return k.Until(ctx, func(pod *corev1.Pod) bool {
		for _, s := range pod.Status.ContainerStatuses {
			if s.Name == "main" && s.State.Terminated != nil {
				return true
			}
		}
		return false
	})
}

func (k *K8sAPIExecutor) Until(ctx context.Context, f func(pod *corev1.Pod) bool) error {
	return k.client.until(ctx, f)
}

// Kill kills a list of containers first with a SIGTERM then with a SIGKILL after a grace period
func (k *K8sAPIExecutor) Kill(ctx context.Context, containerNames []string) error {
	log.Infof("Killing containers %s", containerNames)
	for _, containerName := range containerNames {
		err := k.client.killGracefully(ctx, containerName)
		if err != nil {
			return err
		}
	}
	return nil
}
