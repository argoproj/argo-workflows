package retry

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/argoproj/argo-workflows/v3/util/env"
)

// ExecutorRetry is a retry backoff settings for WorkflowExecutor
// Run	Seconds
// 0	0.000
// 1	1.000
// 2	2.600
// 3	5.160
// 4	9.256
var (
	Steps         = env.LookupEnvIntOr("EXECUTOR_RETRY_BACKOFF_STEPS", 5)
	Duration      = env.LookupEnvDurationOr("EXECUTOR_RETRY_BACKOFF_DURATION", 1*time.Second)
	Factor        = env.LookupEnvFloatOr("EXECUTOR_RETRY_BACKOFF_FACTOR", 1.6)
	Jitter        = env.LookupEnvFloatOr("EXECUTOR_RETRY_BACKOFF_JITTER", 0.5)
	ExecutorRetry = wait.Backoff{Steps: Steps, Duration: Duration, Factor: Factor, Jitter: Jitter}
)
