package signal

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func SignalContainer(restConfig *rest.Config, namespace string, pod string, container string, s os.Signal) error {
	command := fmt.Sprintf("kill -s%d -- -1", s)
	if container == "wait" {
		command = fmt.Sprintf("kill -s %d $(pidof argoexec)", s)
	}
	return ExecPodContainerAndGetOutput(restConfig, namespace, pod, container, "sh", "-c", command)
}

func ExecPodContainerAndGetOutput(restConfig *rest.Config, namespace string, pod string, container string, command ...string) error {
	x, err := common.ExecPodContainer(restConfig, namespace, pod, container, true, true, command...)
	if err != nil {
		return err
	}
	stdout, stderr, err := common.GetExecutorOutput(x)
	log.WithFields(log.Fields{"stdout": stdout, "stderr": stderr}).WithError(err).Debug()
	return err
}
