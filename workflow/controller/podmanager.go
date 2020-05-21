package controller

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
)

type PodItem struct {
	nodeName  string
	nodeID    string
	namespace string
	pod       *v1.Pod
}

type PodManager struct {
	podCreateQueue workqueue.RateLimitingInterface
	kubeClient     kubernetes.Interface
}

func NewPodManager(kubeClient kubernetes.Interface) *PodManager {
	return &PodManager{
		podCreateQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		kubeClient:     kubeClient,
	}
}

func (pm *PodManager) run(ctx context.Context, podCreateWorker int) {
	for i := 0; i < podCreateWorker; i++ {
		go wait.Until(pm.runWorker, time.Second, ctx.Done())
	}
}

func (pm *PodManager) runWorker() {
	for pm.processNextPodItem() {
	}
}

func (pm *PodManager) processNextPodItem() bool {
	pod, quit := pm.podCreateQueue.Get()
	if quit {
		return false
	}
	defer pm.podCreateQueue.Done(pod)

	podItem := pod.(*PodItem)
	_, err := pm.kubeClient.CoreV1().Pods(podItem.namespace).Create(podItem.pod)
	if err != nil {
		if apierr.IsAlreadyExists(err) {
			// workflow pod names are deterministic. We can get here if the
			// controller fails to persist the workflow after creating the pod.
			log.Infof("Skipped pod %s (%s) creation: already exists", podItem.nodeName, podItem.nodeID)
		}
		log.Infof("Failed to create pod %s (%s): %v", podItem.nodeName, podItem.nodeID, err)
	}
	return true
}
