package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func SignalContainer(ctx context.Context, restConfig *rest.Config, pod *corev1.Pod, container string, s syscall.Signal) error {
	command := []string{"/bin/sh", "-c", "kill -%d 1"}

	// If the container has the /var/run/argo volume mounted, this it will have access to `argoexec`.
	for _, c := range pod.Spec.Containers {
		if c.Name == container {
			for _, m := range c.VolumeMounts {
				if m.MountPath == common.VarRunArgoPath {
					command = []string{filepath.Join(common.VarRunArgoPath, "argoexec"), "kill", "%d", "1"}
				}
			}
		}
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

	return ExecPodContainerAndGetOutput(ctx, restConfig, pod.Namespace, pod.Name, container, command...)
}

func ExecPodContainerAndGetOutput(ctx context.Context, restConfig *rest.Config, namespace string, pod string, container string, command ...string) error {
	x, err := common.ExecPodContainer(ctx, restConfig, namespace, pod, container, true, true, command...)
	if err != nil {
		return err
	}
	// workaround for when exec does not properly return: https://github.com/kubernetes/kubernetes/pull/103177
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	stdout, stderr, err := common.GetExecutorOutput(ctx, x)
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(logging.Fields{
		"namespace": namespace,
		"pod":       pod,
		"container": container,
		"stdout":    stdout,
		"stderr":    stderr,
	}).WithError(err).Info(ctx, "signaled container")
	return err
}
