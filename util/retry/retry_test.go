package retry

import (
	"os"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/stretchr/testify/assert"
)

func TestRetryBackoffSettings(t *testing.T) {
	defer func() {
		_ = os.Unsetenv("RETRY_BACKOFF_STEPS")
		_ = os.Unsetenv("RETRY_BACKOFF_DURATION")
		_ = os.Unsetenv("RETRY_BACKOFF_FACTOR")
	}()
	defaultBackoff := wait.Backoff{Steps: 5, Duration: 10 * time.Millisecond, Factor: 2}
	assert.Equal(t, defaultBackoff, BackoffSettings(), "default settings")

	_ = os.Setenv("RETRY_BACKOFF_STEPS", "3")
	_ = os.Setenv("RETRY_BACKOFF_DURATION", "3ms")
	_ = os.Setenv("RETRY_BACKOFF_FACTOR", "3")
	assert.Equal(t, wait.Backoff{Steps: 3, Duration: 3 * time.Millisecond, Factor: 3}, BackoffSettings(), "settings from environment variables")

	_ = os.Setenv("RETRY_BACKOFF_DURATION", "bad-duration")
	assert.Panics(t, func() { BackoffSettings() }, "bad duration from environment variables")

	_ = os.Setenv("RETRY_BACKOFF_DURATION", "3ms")
	_ = os.Setenv("RETRY_BACKOFF_STEPS", "bad-steps")
	assert.Equal(t, wait.Backoff{Steps: defaultBackoff.Steps, Duration: 3 * time.Millisecond, Factor: 3}, BackoffSettings(), "bad steps from environment variables")
}
