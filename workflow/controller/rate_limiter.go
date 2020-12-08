package controller

import (
	"time"

	"k8s.io/client-go/util/workqueue"
)

// https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/podautoscaler/rate_limiters.go
type throughputLimiter struct{}

func (r *throughputLimiter) When(interface{}) time.Duration {
	return defaultRequeueTime
}

func (r *throughputLimiter) Forget(interface{}) {}

func (r *throughputLimiter) NumRequeues(interface{}) int {
	return 1
}

var _ workqueue.RateLimiter = &throughputLimiter{}
