package common

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Wait(kubernetesInterface kubernetes.Interface, namespace, podName, containerID string) error {
	log.Infof("Waiting for container %s to complete", containerID)
	listOptions := metav1.ListOptions{FieldSelector: "metadata.name=" + podName}
	for {
		err, done := waitAux(kubernetesInterface, namespace, containerID, listOptions)
		if done {
			return err
		}
	}
}

func waitAux(kubernetesInterface kubernetes.Interface, namespace, containerID string, listOptions metav1.ListOptions) (error, bool) {
	w, err := kubernetesInterface.CoreV1().Pods(namespace).Watch(listOptions)
	if err != nil {
		return fmt.Errorf("could not watch pod: %w", err), true
	}
	defer w.Stop()
	for event := range w.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			return apierrors.FromObject(event.Object), false
		}
		for _, s := range pod.Status.ContainerStatuses {
			if GetContainerID(&s) == containerID && s.State.Terminated != nil {
				return nil, true
			}
		}
		listOptions.ResourceVersion = pod.ResourceVersion
	}
	return nil, false
}
