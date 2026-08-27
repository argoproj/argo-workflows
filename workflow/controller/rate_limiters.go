package controller

import (
	"time"

	"k8s.io/client-go/util/workqueue"
)

type fixedItemIntervalRateLimiter struct {
	// requeueTime rate limits how often a workflow is reconciled, minimum 1s,
	// otherwise informers are unlikely to be up to date and we'll operate on an
	// out of date version of a workflow. Under high load, the informer can get
	// many seconds behind. Increasing this to 30s would be sensible for some
	// users. Higher values mean that workflows with many short running (<20s)
	// nodes do not progress as quickly. So some users may wish to have this as
	// low as 2s. The default of 10s provides a balance for most users.
	// (DEFAULT_REQUEUE_TIME)
	requeueTime time.Duration
}

func (r *fixedItemIntervalRateLimiter) When(_ string) time.Duration {
	return r.requeueTime
}

func (r *fixedItemIntervalRateLimiter) Forget(string) {}

func (r *fixedItemIntervalRateLimiter) NumRequeues(string) int {
	return 1
}

var _ workqueue.TypedRateLimiter[string] = &fixedItemIntervalRateLimiter{}
