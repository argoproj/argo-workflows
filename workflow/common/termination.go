package common

import (
	"fmt"
	"strconv"
	"syscall"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

func SignalContainer(restConfig *rest.Config, namespace string, pod string, container string, s syscall.Signal) error {
	if err := ExecPodContainerAndGetOutput(restConfig, namespace, pod, container, "/var/run/argo/argoexec", "kill", "--signal", strconv.Itoa(int(s)), "1"); err == nil {
		return nil // if successful, exit successful
	}
	return ExecPodContainerAndGetOutput(restConfig, namespace, pod, container, "/bin/sh", "-c", fmt.Sprintf("kill -%d 1", s))
}

func ExecPodContainerAndGetOutput(restConfig *rest.Config, namespace string, pod string, container string, command ...string) error {
	x, err := ExecPodContainer(restConfig, namespace, pod, container, true, true, command...)
	if err != nil {
		return err
	}
	stdout, stderr, err := GetExecutorOutput(x)
	log.WithFields(log.Fields{"stdout": stdout, "stderr": stderr}).WithError(err).Info()
	return err
}
