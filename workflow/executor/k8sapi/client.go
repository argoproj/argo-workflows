package k8sapi

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/argoproj/argo/errors"
	"github.com/argoproj/argo/workflow/common"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	readWSResponseTimeout = time.Minute * 1
	containerShimPrefix   = "://"
)

type k8sAPIClient struct {
	clientset *kubernetes.Clientset
	config    *restclient.Config
	podName   string
	namespace string
}

func newK8sAPIClient() (*k8sAPIClient, error) {
	kubeconfigPath := os.Getenv(common.EnvVarK8sAPIConfigPath)
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, errors.InternalWrapError(err)
	}

	namespace := os.Getenv(common.EnvVarK8sAPITargetNamespace)
	if namespace == "" {
		return nil, errors.New(errors.CodeBadRequest, fmt.Sprintf("namespace must be specified"))
	}

	podName := os.Getenv(common.EnvVarK8sAPITargetPodName)
	if podName == "" {
		return nil, errors.New(errors.CodeBadRequest, fmt.Sprintf("pod name must be specified"))
	}

	return &k8sAPIClient{
		clientset: clientset,
		config:    kubeConfig,
		podName:   podName,
		namespace: namespace,
	}, nil
}

func (c *k8sAPIClient) getFileContents(containerID, sourcePath string) (string, error) {
	containerStatus, err := c.getContainerStatus(containerID)
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

func (c *k8sAPIClient) createArchive(containerID, sourcePath string) (*bytes.Buffer, error) {
	containerStatus, err := c.getContainerStatus(containerID)
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
	containerStatus, err := c.getContainerStatus(containerID)
	if err != nil {
		return nil, err
	}
	return c.clientset.CoreV1().Pods(c.namespace).
		GetLogs(c.podName, &v1.PodLogOptions{Container: containerStatus.Name, SinceTime: nil}).Stream()
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
	defer outFile.Close()
	_, err = io.Copy(outFile, reader)
	if err != nil {
		return errors.InternalWrapError(err)
	}
	return nil
}

func (c *k8sAPIClient) terminatePodWithContainerID(containerID string, sig syscall.Signal) error {
	containerStatus, err := c.getContainerStatus(containerID)
	if err != nil {
		return err
	}
	if containerStatus.State.Terminated != nil {
		log.Infof("Container %s is already terminated: %v", containerID, containerStatus.State.Terminated.String())
		return nil
	}
	pod, err := c.getPod()
	if err != nil {
		return err
	}
	if pod.Spec.HostPID {
		return fmt.Errorf("cannot terminate a hostPID Pod %s", pod.Name)
	}
	if pod.Spec.RestartPolicy != v1.RestartPolicyNever {
		return fmt.Errorf("cannot terminate pod with a %q restart policy", pod.Spec.RestartPolicy)
	}
	command := []string{"/bin/sh", "-c", fmt.Sprintf("kill -%d 1", sig)}
	exec, err := common.ExecPodContainer(c.config, c.namespace, c.podName, containerStatus.Name, false, false, command...)
	if err != nil {
		return err
	}
	_, _, err = common.GetExecutorOutput(exec)
	return err
}

func getContainerID(container *v1.ContainerStatus) string {
	i := strings.Index(container.ContainerID, containerShimPrefix)
	if i == -1 {
		return ""
	}
	return container.ContainerID[i+len(containerShimPrefix):]
}

func (c *k8sAPIClient) getPod() (*v1.Pod, error) {
	return c.clientset.CoreV1().Pods(c.namespace).Get(c.podName, metav1.GetOptions{})
}

func (c *k8sAPIClient) getContainerStatus(containerID string) (*v1.ContainerStatus, error) {
	pod, err := c.getPod()
	if err != nil {
		return nil, err
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if getContainerID(&containerStatus) != containerID {
			continue
		}
		return &containerStatus, nil
	}
	return nil, errors.New(errors.CodeNotFound, fmt.Sprintf("containerID %q is not found in the pod %s", containerID, c.podName))
}

func (c *k8sAPIClient) waitForTermination(containerID string, timeout time.Duration) error {
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
			containerStatus, err := c.getContainerStatus(containerID)
			if err != nil {
				return err
			}
			if containerStatus.State.Terminated == nil {
				continue
			}
			log.Infof("ContainerID %q is terminated: %v", containerID, containerStatus.String())
			return nil
		case <-timer.C:
			return errors.New(errors.CodeTimeout, fmt.Sprintf("timeout after %s", timeout.String()))
		}
	}
}
