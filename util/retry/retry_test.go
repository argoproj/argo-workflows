package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestExecutorTaskResultRetryTimeout(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	t.Run("Default", func(t *testing.T) {
		t.Setenv("EXECUTOR_TASK_RESULT_RETRY_TIMEOUT_SECONDS", "")
		assert.Equal(t, 10*time.Minute, ExecutorTaskResultRetryTimeout(ctx))
	})
	t.Run("FromEnv", func(t *testing.T) {
		t.Setenv("EXECUTOR_TASK_RESULT_RETRY_TIMEOUT_SECONDS", "90")
		assert.Equal(t, 90*time.Second, ExecutorTaskResultRetryTimeout(ctx))
	})
	t.Run("NegativeUsesDefault", func(t *testing.T) {
		t.Setenv("EXECUTOR_TASK_RESULT_RETRY_TIMEOUT_SECONDS", "-1")
		assert.Equal(t, 10*time.Minute, ExecutorTaskResultRetryTimeout(ctx))
	})
	t.Run("ZeroUsesDefault", func(t *testing.T) {
		t.Setenv("EXECUTOR_TASK_RESULT_RETRY_TIMEOUT_SECONDS", "0")
		assert.Equal(t, 10*time.Minute, ExecutorTaskResultRetryTimeout(ctx))
	})
}
