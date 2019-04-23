package k8sapi

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	"github.com/argoproj/argo/util"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	execcommon "github.com/argoproj/argo/workflow/executor/common"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

type k8sAPIClient struct {
	clientset *kubernetes.Clientset
	config    *restclient.Config
	podName   string
	namespace string
}

var _ execcommon.KubernetesClientInterface = &k8sAPIClient{}

func newK8sAPIClient(clientset *kubernetes.Clientset, config *restclient.Config, podName, namespace string) (*k8sAPIClient, error) {
	return &k8sAPIClient{
		clientset: clientset,
		config:    config,
		podName:   podName,
		namespace: namespace,
	}, nil
}

func (c *k8sAPIClient) getFileContents(containerID, sourcePath string) (string, error) {
	_, containerStatus, err := c.GetContainerStatus(containerID)
	if err != nil {
		return "", err
	}
	command := []string{"cat", sourcePath}
	exec, err := common.ExecPodContainer(c.config, c.namespace, c.podName, containerStatus.Name, true, false, command...)
	if err != nil {
		return "", err
	}
	stdOut, _, err := common.GetExecutorOutput(exec)
	if err != nil {
		return "", err
	}
	return stdOut.String(), nil
}

func (c *k8sAPIClient) CreateArchive(containerID, sourcePath string) (*bytes.Buffer, error) {
	_, containerStatus, err := c.GetContainerStatus(containerID)
	if err != nil {
		return nil, err
	}
	command := []string{"tar", "cf", "-", sourcePath}
	exec, err := common.ExecPodContainer(c.config, c.namespace, c.podName, containerStatus.Name, true, false, command...)
	if err != nil {
		return nil, err
	}
	stdOut, _, err := common.GetExecutorOutput(exec)
	if err != nil {
		return nil, err
	}
	return stdOut, nil
}

func (c *k8sAPIClient) getLogsAsStream(containerID string) (io.ReadCloser, error) {
	_, containerStatus, err := c.GetContainerStatus(containerID)
	if err != nil {
		return nil, err
	}
	return c.clientset.CoreV1().Pods(c.namespace).
		GetLogs(c.podName, &v1.PodLogOptions{Container: containerStatus.Name, SinceTime: &metav1.Time{}}).Stream()
}

func (c *k8sAPIClient) getLogs(containerID string) (string, error) {
	reader, err := c.getLogsAsStream(containerID)
	if err != nil {
		return "", err
	}
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", errors.InternalWrapError(err)
	}
	return string(bytes), nil
}

func (c *k8sAPIClient) saveLogs(containerID, path string) error {
	reader, err := c.getLogsAsStream(containerID)
	if err != nil {
		return err
	}
	outFile, err := os.Create(path)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	defer util.Close(outFile)
	_, err = io.Copy(outFile, reader)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	return nil
}

func (c *k8sAPIClient) getPod() (*v1.Pod, error) {
	return c.clientset.CoreV1().Pods(c.namespace).Get(c.podName, metav1.GetOptions{})
}

func (c *k8sAPIClient) GetContainerStatus(containerID string) (*v1.Pod, *v1.ContainerStatus, error) {
	pod, err := c.getPod()
	if err != nil {
		return nil, nil, err
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if execcommon.GetContainerID(&containerStatus) != containerID {
			continue
		}
		return pod, &containerStatus, nil
	}
	return nil, nil, errors.New(errors.CodeNotFound, fmt.Sprintf("containerID %q is not found in the pod %s", containerID, c.podName))
}

func (c *k8sAPIClient) waitForTermination(containerID string, timeout time.Duration) error {
	return execcommon.WaitForTermination(c, containerID, timeout)
}

func (c *k8sAPIClient) KillContainer(pod *v1.Pod, container *v1.ContainerStatus, sig syscall.Signal) error {
	command := []string{"/bin/sh", "-c", fmt.Sprintf("kill -%d 1", sig)}
	exec, err := common.ExecPodContainer(c.config, c.namespace, c.podName, container.Name, false, false, command...)
	if err != nil {
		return err
	}
	_, _, err = common.GetExecutorOutput(exec)
	return err
}

func (c *k8sAPIClient) killGracefully(containerID string) error {
	return execcommon.KillGracefully(c, containerID)
}

func (c *k8sAPIClient) copyArchive(containerID, sourcePath, destPath string) error {
	return execcommon.CopyArchive(c, containerID, sourcePath, destPath)
}
