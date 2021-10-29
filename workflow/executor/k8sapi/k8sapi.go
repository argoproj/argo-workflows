package k8sapi

import (
	"context"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	execcommon "github.com/argoproj/argo-workflows/v3/workflow/executor/common"
)

type K8sAPIExecutor struct {
	client *k8sAPIClient
}

func NewK8sAPIExecutor(clientset kubernetes.Interface, config *restclient.Config, podName, namespace string) *K8sAPIExecutor {
	client := newK8sAPIClient(clientset, config, podName, namespace)
	return &K8sAPIExecutor{
		client: client,
	}
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

// Wait for the container to complete
func (k *K8sAPIExecutor) Wait(ctx context.Context, containerNames []string) error {
	return k.Until(ctx, func(pod *corev1.Pod) bool {
		return execcommon.AllTerminated(pod.Status.ContainerStatuses, containerNames) || pod.Status.Reason == common.ErrDeadlineExceeded
	})
}

func (k *K8sAPIExecutor) Until(ctx context.Context, f func(pod *corev1.Pod) bool) error {
	return k.client.until(ctx, f)
}

// Kill kills a list of containers first with a SIGTERM then with a SIGKILL after a grace period
func (k *K8sAPIExecutor) Kill(ctx context.Context, containerNames []string, terminationGracePeriodDuration time.Duration) error {
	log.Infof("Killing containers %v", containerNames)
	return k.client.killGracefully(ctx, containerNames, terminationGracePeriodDuration)
}

func (k *K8sAPIExecutor) ListContainerNames(ctx context.Context) ([]string, error) {
	pod, err := k.client.getPod(ctx)
	if err != nil {
		return nil, err
	}
	var containerNames []string
	for _, c := range pod.Status.ContainerStatuses {
		containerNames = append(containerNames, c.Name)
	}
	return containerNames, nil
}
