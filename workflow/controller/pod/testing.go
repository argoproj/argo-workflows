package pod

import (
	"context"

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

// TestingNewController returns a controller that only serves pod lookups from
// podInformer, with no cleanup queue or background goroutines.
func TestingNewController(podInformer cache.SharedIndexInformer) *Controller {
	return &Controller{podInformer: podInformer}
}
