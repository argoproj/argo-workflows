package retry

import (
	"context"
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
func ExecutorRetry(ctx context.Context) wait.Backoff {
	steps := env.LookupEnvIntOr(ctx, "EXECUTOR_RETRY_BACKOFF_STEPS", 5)
	duration := env.LookupEnvDurationOr(ctx, "EXECUTOR_RETRY_BACKOFF_DURATION", 1*time.Second)
	factor := env.LookupEnvFloatOr(ctx, "EXECUTOR_RETRY_BACKOFF_FACTOR", 1.6)
	jitter := env.LookupEnvFloatOr(ctx, "EXECUTOR_RETRY_BACKOFF_JITTER", 0.5)
	return wait.Backoff{Steps: steps, Duration: duration, Factor: factor, Jitter: jitter}
}
