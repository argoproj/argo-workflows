package wait

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/argoproj/argo/workflow/executor/common"
)

func UntilTerminated(kubernetesInterface kubernetes.Interface, namespace, podName, containerID string) error {
	log.Infof("Waiting for container %s to be terminated", containerID)
	podInterface := kubernetesInterface.CoreV1().Pods(namespace)
	listOptions := metav1.ListOptions{FieldSelector: "metadata.name=" + podName}
	for {
		done, err := untilTerminatedAux(podInterface, containerID, listOptions)
		if done {
			return err
		}
	}
}

func untilTerminatedAux(podInterface v1.PodInterface, containerID string, listOptions metav1.ListOptions) (bool, error) {
	w, err := podInterface.Watch(listOptions)
	if err != nil {
		return true, fmt.Errorf("could not watch pod: %w", err)
	}
	defer w.Stop()
	for event := range w.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			return false, apierrors.FromObject(event.Object)
		}
		for _, s := range pod.Status.ContainerStatuses {
			if common.GetContainerID(&s) == containerID && s.State.Terminated != nil {
				return true, nil
			}
		}
		listOptions.ResourceVersion = pod.ResourceVersion
	}
	return true, nil
}
