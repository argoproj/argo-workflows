package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTaskConfig_MainEnv(t *testing.T) {
	cfg := TaskConfig{
		WorkflowName:        "my-wf",
		NodeID:              "my-wf-1",
		IncludeScriptOutput: true,
		Deadline:            time.Date(2026, 10, 27, 12, 0, 0, 0, time.FixedZone("BST", 3600)),
		TraceContext:        map[string]string{"tracestate": "k=v", "traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"},
	}
	assert.Equal(t, []string{
		"ARGO_NODE_ID=my-wf-1",
		"ARGO_INCLUDE_SCRIPT_OUTPUT=true",
		"ARGO_DEADLINE=2026-10-27T11:00:00Z",
		"ARGO_WORKFLOW_NAME=my-wf",
		"TRACEPARENT=00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
		"TRACESTATE=k=v",
	}, cfg.MainEnv())

	t.Run("Progress", func(t *testing.T) {
		cfg := TaskConfig{ProgressFile: "/var/run/argo/progress", ProgressPatchTickDuration: time.Minute, ProgressFileTickDuration: 3 * time.Second}
		assert.Equal(t, []string{
			"ARGO_PROGRESS_FILE=/var/run/argo/progress",
			"ARGO_PROGRESS_PATCH_TICK_DURATION=1m0s",
			"ARGO_PROGRESS_FILE_TICK_DURATION=3s",
		}, cfg.MainEnv()[4:])
		cfg.ProgressFileTickDuration = 0
		assert.Len(t, cfg.MainEnv(), 4, "progress is monitored only when both ticks are set")
	})
	t.Run("NoDeadline", func(t *testing.T) {
		assert.Contains(t, TaskConfig{}.MainEnv(), "ARGO_DEADLINE=0001-01-01T00:00:00Z", "as a pod's")
	})
}
