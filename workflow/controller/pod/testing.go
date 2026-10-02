package pod

import (
	"context"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"

	"k8s.io/client-go/tools/cache"
)

// Accessors for the unit tests in /workflow/controller

func (c *Controller) TestingPodInformer() cache.SharedIndexInformer {
	return c.podInformer
}

func (c *Controller) TestingProcessNextItem(ctx context.Context) bool {
	return c.processNextPodCleanupItem(ctx)
}

func (c *Controller) TestingQueueNumRequeues(key string) int {
	return c.workqueue.NumRequeues(key)
}

func (c *Controller) TestingQueueLen() int {
	return c.workqueue.Len()
}

// TestingPodEvent uses the same Add or significant Update path as the informer.
func (c *Controller) TestingPodEvent(ctx context.Context, pod *apiv1.Pod, update bool) {
	if update {
		c.updatePodEvent(ctx, pod, pod)
	} else {
		c.addPodEvent(ctx, pod)
	}
}

// TestingSetQueue permits a controlled scheduling clock in integration tests.
func (c *Controller) TestingSetQueue(queue workqueue.TypedRateLimitingInterface[string]) {
	c.workqueue.ShutDown()
	c.workqueue = queue
}
