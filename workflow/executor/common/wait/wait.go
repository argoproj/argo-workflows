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

const (
	pod_finished = 0
	pod_running = 1
	pod_unknown = 2
)

func UntilTerminated(kubernetesInterface kubernetes.Interface, namespace, podName, containerID string) error {
	log.Infof("Waiting for container %s to be terminated", containerID)
	podInterface := kubernetesInterface.CoreV1().Pods(namespace)
	listOptions := metav1.ListOptions{FieldSelector: "metadata.name=" + podName}
	for {
		pod_status, err := untilTerminatedAux(podInterface, containerID, listOptions)
		if err != nil {
			return err
		}
		switch pod_status {
			case pod_finished:
				return nil
			case pod_unknown:
				log.Info("Pod watch timed out, checking pod status")
				pod, err := podInterface.Get(podName, metav1.GetOptions{})
				if err != nil {
					return err
				}
				if hasPodTerminated(pod, containerID) {
					return nil
				}
		}
	}
}

func untilTerminatedAux(podInterface v1.PodInterface, containerID string, listOptions metav1.ListOptions) (int, error) {
	w, err := podInterface.Watch(listOptions)
	if err != nil {
		return pod_unknown, fmt.Errorf("could not watch pod: %w", err)
	}
	defer w.Stop()
	for event := range w.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			return pod_unknown, apierrors.FromObject(event.Object)
		}
		if hasPodTerminated(pod, containerID) {
			return pod_finished, nil
		}
		listOptions.ResourceVersion = pod.ResourceVersion
	}
	return pod_unknown, nil
}

func hasPodTerminated(pod *corev1.Pod, containerID string) bool {
	for _, s := range pod.Status.ContainerStatuses {
		if common.GetContainerID(&s) == containerID && s.State.Terminated != nil {
			return true
		}
	}
	return false
}
