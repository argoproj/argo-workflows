package controller

import (
	"slices"
	"sync"
	"time"

	"k8s.io/client-go/util/workqueue"
)

// requeues records the requeues made through a controller's workflow queue:
// rate-limited ones, as woc.requeue makes, and delayed ones, as
// woc.requeueAfter makes.
type requeues struct {
	workqueue.TypedRateLimitingInterface[string]
	mu          sync.Mutex
	rateLimited int
	after       []time.Duration
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

func (r *requeues) AddAfter(key string, d time.Duration) {
	r.mu.Lock()
	r.after = append(r.after, d)
	r.mu.Unlock()
	r.TypedRateLimitingInterface.AddAfter(key, d)
}

// take returns the rate-limited requeues and the delays of the delayed ones
// recorded since the last take.
func (r *requeues) take() (int, []time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, after := r.rateLimited, r.after
	r.rateLimited, r.after = 0, nil
	return n, after
}

// requeuedAfter says whether one of after is a requeue for when remaining,
// measured a moment before, ends.
func requeuedAfter(after []time.Duration, remaining time.Duration) bool {
	return slices.ContainsFunc(after, func(d time.Duration) bool { return d <= remaining && d > remaining-5*time.Second })
}
