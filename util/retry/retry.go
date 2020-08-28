package retry

import (
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// DefaultRetry is a default retry backoff settings when retrying API calls
var DefaultRetry = wait.Backoff{
	Steps:    5,
	Duration: 10 * time.Millisecond,
	Factor:   1.0,
}
