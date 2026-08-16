package controller

import (
	"time"

	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v4/util/env"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

var envRequeueTime = env.LookupEnvDurationOr(logging.InitLoggerInContext(), common.EnvVarDefaultRequeueTime, 10*time.Second)

func GetRequeueTime() time.Duration {
	// We need to rate limit a minimum 1s, otherwise informers are unlikely to be upto date
	// and we'll operate on an out of date version of a workflow.
	// Under high load, the informer can get many seconds behind. Increasing this to 30s
	// would be sensible for some users.
	// Higher values mean that workflows with many short running (<20s) nodes do not progress as quickly.
	// So some users may wish to have this as low as 2s.
	// The default of 10s provides a balance more most users.
	return envRequeueTime
}

type fixedItemIntervalRateLimiter struct{}

func (r *fixedItemIntervalRateLimiter) When(_ string) time.Duration {
	return GetRequeueTime()
}

func (r *fixedItemIntervalRateLimiter) Forget(string) {}

func (r *fixedItemIntervalRateLimiter) NumRequeues(string) int {
	return 1
}

var _ workqueue.TypedRateLimiter[string] = &fixedItemIntervalRateLimiter{}
