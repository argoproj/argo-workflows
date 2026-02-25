package retry

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	envutil "github.com/argoproj/argo-workflows/v4/util/env"
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
