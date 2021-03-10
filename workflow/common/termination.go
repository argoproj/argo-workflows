package common

import (
	"fmt"
	"syscall"

	"k8s.io/client-go/rest"
)

func GetKillCommand(s syscall.Signal) []string {
	return []string{"/bin/sh", "-c", fmt.Sprintf("kill -%d 1", s)}
}

func SignalContainer(restConfig *rest.Config, namespace string, pod string, container string, s syscall.Signal) error {
	x, err := ExecPodContainer(restConfig, namespace, pod, container, false, true, GetKillCommand(s)...)
	if err != nil {
		return err
	}
	_, _, err = GetExecutorOutput(x)
	return err
}
