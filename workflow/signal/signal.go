package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func Container(ctx context.Context, restConfig *rest.Config, pod *corev1.Pod, container string, s syscall.Signal) error {
	command, err := killCommand(pod, container, s)
	if err != nil {
		return err
	}
	return ExecPodContainerAndGetOutput(ctx, restConfig, pod.Namespace, pod.Name, container, command...)
}

// killCommand builds the command that delivers signal s to the container's
// entrypoint (PID 1).
//
// It prefers `argoexec kill` (which works even on distroless/scratch images that
// lack /bin/sh). argoexec reaches the container by one of two routes depending
// on the pod layout:
//   - init-less: delivered through the argoexec-bin image volume at
//     common.ArgoExecBinPath (nothing populates VarRunArgoPath/argoexec here);
//   - legacy: copied by the init container into common.LegacyArgoExecBinPath.
//
// The image-volume mount is checked first so init-less main-level containers
// don't fall through to the (non-existent) legacy path. A kill-command
// annotation, when present, overrides everything.
func killCommand(pod *corev1.Pod, container string, s syscall.Signal) ([]string, error) {
	command := []string{"/bin/sh", "-c", "kill -%d 1"}

	if common.IsArgoAuxilliary(container) {
		// Argoexec is on the path for our own sidecars
		command = []string{"argoexec", "kill", "%d", "1"}
	} else {
		for _, c := range pod.Spec.Containers {
			if c.Name != container {
				continue
			}
			var hasVarRunArgo, hasArgoExecBin bool
			for _, m := range c.VolumeMounts {
				switch m.MountPath {
				case common.VarRunArgoPath:
					hasVarRunArgo = true
				case common.ArgoExecBinMountPath:
					hasArgoExecBin = true
				}
			}
			switch {
			case hasArgoExecBin:
				command = []string{common.ArgoExecBinPath, "kill", "%d", "1"}
			case hasVarRunArgo:
				command = []string{common.LegacyArgoExecBinPath, "kill", "%d", "1"}
			}
		}
	}
	if v, ok := pod.Annotations[common.AnnotationKeyKillCmd(container)]; ok {
		if err := json.Unmarshal([]byte(v), &command); err != nil {
			return nil, fmt.Errorf("failed to unmarshall kill command annotation %q: %w", v, err)
		}
	}

	for i, v := range command {
		if strings.Contains(v, "%d") {
			command[i] = fmt.Sprintf(v, s)
		}
	}
	return command, nil
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
