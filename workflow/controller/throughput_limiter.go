package controller

import (
	"time"

	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo/util/env"
)

// https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/podautoscaler/rate_limiters.go
type throughputLimiter struct{}

func (r *throughputLimiter) When(interface{}) time.Duration {
	return env.LookupEnvDurationOr("DEFAULT_REQUEUE_TIME", 2*time.Second)
}

func (r *throughputLimiter) Forget(interface{}) {}

func (r *throughputLimiter) NumRequeues(interface{}) int {
	return 1
}

var _ workqueue.RateLimiter = &throughputLimiter{}
