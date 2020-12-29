package controller

import (
	"time"

	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo/util/env"
)

type fixedItemIntervalRateLimiter struct{}

func (r *fixedItemIntervalRateLimiter) When(interface{}) time.Duration {
	return env.LookupEnvDurationOr("DEFAULT_REQUEUE_TIME", 10*time.Second)
}

func (r *fixedItemIntervalRateLimiter) Forget(interface{}) {}

func (r *fixedItemIntervalRateLimiter) NumRequeues(interface{}) int {
	return 1
}

var _ workqueue.RateLimiter = &fixedItemIntervalRateLimiter{}
