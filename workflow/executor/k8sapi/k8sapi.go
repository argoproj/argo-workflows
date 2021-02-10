package k8sapi

import (
	"context"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/common/wait"
)

type K8sAPIExecutor struct {
	client *k8sAPIClient
}

func NewK8sAPIExecutor(clientset *kubernetes.Clientset, config *restclient.Config, podName, namespace string) (*K8sAPIExecutor, error) {
	log.Infof("Creating a K8sAPI executor")
	client, err := newK8sAPIClient(clientset, config, podName, namespace)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	return &K8sAPIExecutor{
		client: client,
	}, nil
}

func (k *K8sAPIExecutor) GetFileContents(containerID string, sourcePath string) (string, error) {
	return "", errors.Errorf(errors.CodeNotImplemented, "GetFileContents() is not implemented in the k8sapi executor.")
}

func (k *K8sAPIExecutor) CopyFile(containerID string, sourcePath string, destPath string, compressionLevel int) error {
	return errors.Errorf(errors.CodeNotImplemented, "CopyFile() is not implemented in the k8sapi executor.")
}

func (k *K8sAPIExecutor) GetOutputStream(ctx context.Context, containerID string, combinedOutput bool) (io.ReadCloser, error) {
	log.Infof("Getting output of %s", containerID)
	if !combinedOutput {
		log.Warn("non combined output unsupported")
	}
	return k.client.getLogsAsStream(ctx, containerID)
}

func (k *K8sAPIExecutor) GetExitCode(ctx context.Context, containerID string) (string, error) {
	log.Infof("Getting exit code of %s", containerID)
	_, status, err := k.client.GetContainerStatus(ctx, containerID)
	if err != nil {
		return "", errors.InternalWrapError(err, "Could not get container status")
	}
	if status != nil && status.State.Terminated != nil {
		return fmt.Sprint(status.State.Terminated.ExitCode), nil
	}
	return "", nil
}

func (k *K8sAPIExecutor) WaitInit() error {
	return nil
}

// Wait for the container to complete
func (k *K8sAPIExecutor) Wait(ctx context.Context, containerID string) error {
	return wait.UntilTerminated(ctx, k.client.clientset, k.client.namespace, k.client.podName, containerID)
}

// Kill kills a list of containerIDs first with a SIGTERM then with a SIGKILL after a grace period
func (k *K8sAPIExecutor) Kill(ctx context.Context, containerIDs []string, terminationGracePeriodDuration time.Duration) error {
	log.Infof("Killing containers %s", containerIDs)
	for _, containerID := range containerIDs {
		err := k.client.killGracefully(ctx, containerID, terminationGracePeriodDuration)
		if err != nil {
			return err
		}
	}
	return nil
}
