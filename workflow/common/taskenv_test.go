package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
)

// The progress env is set only when both ticks are, as progress is only
// monitored then.
func TestTaskEnv_EnvVars(t *testing.T) {
	env := TaskEnv{
		NodeID:                    "wf-1",
		Deadline:                  time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC),
		WorkflowName:              "wf",
		TraceContext:              map[string]string{"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"},
		ProgressFile:              "/var/run/argo/progress",
		ProgressPatchTickDuration: time.Minute,
	}
	want := []apiv1.EnvVar{
		{Name: EnvVarNodeID, Value: "wf-1"},
		{Name: EnvVarIncludeScriptOutput, Value: "false"},
		{Name: EnvVarDeadline, Value: "2026-10-27T12:00:00Z"},
		{Name: EnvVarWorkflowName, Value: "wf"},
		{Name: "TRACEPARENT", Value: "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"},
	}
	assert.Equal(t, want, env.EnvVars())

	env.ProgressFileTickDuration = 3 * time.Second
	assert.Equal(t, append(want,
		apiv1.EnvVar{Name: EnvVarProgressFile, Value: "/var/run/argo/progress"},
		apiv1.EnvVar{Name: EnvVarProgressPatchTickDuration, Value: "1m0s"},
		apiv1.EnvVar{Name: EnvVarProgressFileTickDuration, Value: "3s"},
	), env.EnvVars())
}
