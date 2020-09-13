package retry

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// DefaultRetry is a default retry backoff settings when retrying API calls
// Retry   Seconds
//     1      0.01
//     2      0.03
//     3      0.07
//     4      0.15
//     5      0.31
var DefaultRetry = wait.Backoff{
	Steps:    5,
	Duration: 10 * time.Millisecond,
	Factor:   2,
}
