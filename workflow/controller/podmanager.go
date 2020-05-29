package controller

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type PodItem struct {
	wf        *wfv1.Workflow
	nodeName  string
	nodeID    string
	namespace string
	pod       *v1.Pod
}

type PodManager struct {
	podCreateQueue workqueue.RateLimitingInterface
	kubeClient     kubernetes.Interface
	controller     WorkflowController
	nodeError      map[string]error
}

func NewPodManager(kubeClient kubernetes.Interface, controller WorkflowController) *PodManager {
	return &PodManager{
		podCreateQueue: workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		kubeClient:     kubeClient,
		controller:     controller,
		nodeError:      make(map[string]error),
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
		pm.nodeError[podItem.nodeID] = err
		key, _ := cache.MetaNamespaceKeyFunc(podItem.wf)
		pm.controller.wfQueue.Add(key)
	}
	return true
}
