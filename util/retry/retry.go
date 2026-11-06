package retry

import (
	"context"
	"math"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	envutil "github.com/argoproj/argo-workflows/v4/util/env"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// DefaultRetry is a default retry backoff settings when retrying API calls
// Retry   Seconds
//
//	1      0.01
//	2      0.03
//	3      0.07
//	4      0.15
//	5      0.31
func DefaultRetry(ctx context.Context) wait.Backoff {
	return wait.Backoff{
		Steps:    envutil.LookupEnvIntOr(ctx, "RETRY_BACKOFF_STEPS", 5),
		Duration: envutil.LookupEnvDurationOr(ctx, "RETRY_BACKOFF_DURATION", 10*time.Millisecond),
		Factor:   envutil.LookupEnvFloatOr(ctx, "RETRY_BACKOFF_FACTOR", 2.),
	}
}

// ExecutorRetry is a retry backoff settings for WorkflowExecutor
// Run	Seconds
// 0	0.000
// 1	1.000
// 2	2.600
// 3	5.160
// 4	9.256
func ExecutorRetry(ctx context.Context) wait.Backoff {
	steps := envutil.LookupEnvIntOr(ctx, "EXECUTOR_RETRY_BACKOFF_STEPS", 5)
	duration := envutil.LookupEnvDurationOr(ctx, "EXECUTOR_RETRY_BACKOFF_DURATION", 1*time.Second)
	factor := envutil.LookupEnvFloatOr(ctx, "EXECUTOR_RETRY_BACKOFF_FACTOR", 1.6)
	jitter := envutil.LookupEnvFloatOr(ctx, "EXECUTOR_RETRY_BACKOFF_JITTER", 0.5)
	return wait.Backoff{Steps: steps, Duration: duration, Factor: factor, Jitter: jitter}
}

// DefaultExecutorTaskResultRetryTimeout is the default for
// ExecutorTaskResultRetryTimeout.
const DefaultExecutorTaskResultRetryTimeout = 10 * time.Minute

// ExecutorTaskResultRetryTimeout is how long the WorkflowExecutor keeps
// retrying transient errors when writing its task result. Values that are
// not positive, or too large for a time.Duration, use the default.
func ExecutorTaskResultRetryTimeout(ctx context.Context) time.Duration {
	const maxSeconds = math.MaxInt64 / int64(time.Second)
	seconds := envutil.LookupEnvIntOr(ctx, "EXECUTOR_TASK_RESULT_RETRY_TIMEOUT_SECONDS", int(DefaultExecutorTaskResultRetryTimeout/time.Second))
	if seconds <= 0 || int64(seconds) > maxSeconds {
		logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{
			"EXECUTOR_TASK_RESULT_RETRY_TIMEOUT_SECONDS": seconds,
			"max":     maxSeconds,
			"default": DefaultExecutorTaskResultRetryTimeout,
		}).Warn(ctx, "EXECUTOR_TASK_RESULT_RETRY_TIMEOUT_SECONDS is out of range (1 to max), using the default")
		return DefaultExecutorTaskResultRetryTimeout
	}
	return time.Duration(seconds) * time.Second
}
