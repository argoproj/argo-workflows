package common

import (
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watchpkg "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/watch"
)

func Wait(kubernetesInterface kubernetes.Interface, namespace, podName, containerID string) error {
	log.Infof("Waiting for container %s to complete", containerID)
	watcher, err := watch.NewRetryWatcher("", &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watchpkg.Interface, error) {
			options.FieldSelector = "metadata.name=" + podName
			return kubernetesInterface.CoreV1().Pods(namespace).Watch(options)
		},
	})
	if err != nil {
		return err
	}
	defer watcher.Stop()
	for event := range watcher.ResultChan() {
		pod, ok := event.Object.(*corev1.Pod)
		if !ok {
			return apierrors.FromObject(event.Object)
		}
		for _, s := range pod.Status.ContainerStatuses {
			if GetContainerID(&s) == containerID && s.State.Terminated != nil {
				return nil
			}
		}
	}
	return nil
}
