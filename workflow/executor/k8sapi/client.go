package k8sapi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/errors"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	waitutil "github.com/argoproj/argo-workflows/v3/util/wait"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	execcommon "github.com/argoproj/argo-workflows/v3/workflow/executor/common"
)

type k8sAPIClient struct {
	clientset kubernetes.Interface
	config    *restclient.Config
	podName   string
	namespace string
}

var _ execcommon.KubernetesClientInterface = &k8sAPIClient{}

func newK8sAPIClient(clientset kubernetes.Interface, config *restclient.Config, podName, namespace string) *k8sAPIClient {
	return &k8sAPIClient{
		clientset: clientset,
		config:    config,
		podName:   podName,
		namespace: namespace,
	}
}

func (c *k8sAPIClient) CreateArchive(ctx context.Context, containerName, sourcePath string) (*bytes.Buffer, error) {
	command := []string{"tar", "cf", "-", sourcePath}
	exec, err := common.ExecPodContainer(c.config, c.namespace, c.podName, containerName, true, false, command...)
	if err != nil {
		return nil, err
	}
	stdOut, _, err := common.GetExecutorOutput(exec)
	if err != nil {
		return nil, err
	}
	return stdOut, nil
}

func (c *k8sAPIClient) getLogsAsStream(ctx context.Context, containerName string) (io.ReadCloser, error) {
	return c.clientset.CoreV1().Pods(c.namespace).
		GetLogs(c.podName, &corev1.PodLogOptions{Container: containerName, SinceTime: &metav1.Time{}}).Stream(ctx)
}

var backoffOver30s = wait.Backoff{
	Duration: 1 * time.Second,
	Steps:    7,
	Factor:   2,
}

func (c *k8sAPIClient) getPod(ctx context.Context) (*corev1.Pod, error) {
	var pod *corev1.Pod
	err := waitutil.Backoff(backoffOver30s, func() (bool, error) {
		var err error
		pod, err = c.clientset.CoreV1().Pods(c.namespace).Get(ctx, c.podName, metav1.GetOptions{})
		return !errorsutil.IsTransientErr(err), err
	})
	return pod, err
}

func (c *k8sAPIClient) GetContainerStatus(ctx context.Context, containerName string) (*corev1.Pod, *corev1.ContainerStatus, error) {
	pod, containerStatuses, err := c.GetContainerStatuses(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, s := range containerStatuses {
		if s.Name != containerName {
			continue
		}
		return pod, &s, nil
	}
	return nil, nil, errors.New(errors.CodeNotFound, fmt.Sprintf("container %q is not found in the pod %s", containerName, c.podName))
}

func (c *k8sAPIClient) GetContainerStatuses(ctx context.Context) (*corev1.Pod, []corev1.ContainerStatus, error) {
	pod, err := c.getPod(ctx)
	if err != nil {
		return nil, nil, err
	}
	return pod, pod.Status.ContainerStatuses, nil
}

func (c *k8sAPIClient) KillContainer(pod *corev1.Pod, container *corev1.ContainerStatus, sig syscall.Signal) error {
	command := []string{"/bin/sh", "-c", fmt.Sprintf("kill -%d 1", sig)}
	exec, err := common.ExecPodContainer(c.config, c.namespace, c.podName, container.Name, false, true, command...)
	if err != nil {
		return err
	}
	_, _, err = common.GetExecutorOutput(exec)
	return err
}

func (c *k8sAPIClient) killGracefully(ctx context.Context, containerNames []string, terminationGracePeriodDuration time.Duration) error {
	return execcommon.KillGracefully(ctx, c, containerNames, terminationGracePeriodDuration)
}

func (c *k8sAPIClient) until(ctx context.Context, f func(pod *corev1.Pod) bool) error {
	podInterface := c.clientset.CoreV1().Pods(c.namespace)
	for {
		done, err := func() (bool, error) {
			w, err := podInterface.Watch(ctx, metav1.ListOptions{FieldSelector: "metadata.name=" + c.podName})
			if err != nil {
				return true, fmt.Errorf("failed to establish pod watch: %w", err)
			}
			defer w.Stop()
			for {
				select {
				case <-ctx.Done():
					return true, ctx.Err()
				case event, open := <-w.ResultChan():
					if !open {
						return false, fmt.Errorf("channel not open")
					}
					pod, ok := event.Object.(*corev1.Pod)
					if !ok {
						return true, apierrors.FromObject(event.Object)
					}
					done := f(pod)
					if done {
						return true, nil
					}
				}
			}
		}()
		if done {
			return err
		}
	}
}
