package controller

import (
	"sync"

	"k8s.io/client-go/util/workqueue"
)

// requeues records the rate-limited requeues made through a controller's
// workflow queue, as woc.requeue makes.
type requeues struct {
	workqueue.TypedRateLimitingInterface[string]
	mu          sync.Mutex
	rateLimited int
}

// recordRequeues makes the controller's workflow queue record its requeues.
func recordRequeues(controller *WorkflowController) *requeues {
	r := &requeues{TypedRateLimitingInterface: controller.wfQueue}
	controller.wfQueue = r
	return r
}

func (r *requeues) AddRateLimited(key string) {
	r.mu.Lock()
	r.rateLimited++
	r.mu.Unlock()
	r.TypedRateLimitingInterface.AddRateLimited(key)
}

// take returns the rate-limited requeues recorded since the last take.
func (r *requeues) take() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := r.rateLimited
	r.rateLimited = 0
	return n
}
