package signal

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall"

	corev1 "k8s.io/api/core/v1"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func SignalContainer(restConfig *rest.Config, pod *corev1.Pod, container string, s syscall.Signal) error {
	command := []string{"/bin/sh", "-c", "kill -%d 1"}
	if container == common.WaitContainerName {
		command = []string{"/bin/sh", "-c", "kill -%d $(pidof argoexec)"}
	}

	if v, ok := pod.Annotations[common.AnnotationKeyKillCmd(container)]; ok {
		if err := json.Unmarshal([]byte(v), &command); err != nil {
			return fmt.Errorf("failed to unmarshall kill command annotation %q: %w", v, err)
		}
	}

	for i, v := range command {
		if strings.Contains(v, "%d") {
			command[i] = fmt.Sprintf(v, s)
		}
	}

	return ExecPodContainerAndGetOutput(restConfig, pod.Namespace, pod.Name, container, command...)
}

func ExecPodContainerAndGetOutput(restConfig *rest.Config, namespace string, pod string, container string, command ...string) error {
	x, err := common.ExecPodContainer(restConfig, namespace, pod, container, true, true, command...)
	if err != nil {
		return err
	}
	stdout, stderr, err := common.GetExecutorOutput(x)
	log.
		WithField("namespace", namespace).
		WithField("pod", pod).
		WithField("container", container).
		WithField("stdout", stdout).
		WithField("stderr", stderr).
		WithError(err).
		Info("signaled container")
	return err
}
