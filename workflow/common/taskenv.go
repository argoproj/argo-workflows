package common

import (
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/propagators/envcar"
	"go.opentelemetry.io/otel/propagation"
	apiv1 "k8s.io/api/core/v1"
)

// TaskEnv is the per-task env a task's main gets on top of its own: the
// controller sets it on a pod's containers, and a hot worker pool pod's
// worker gives it to main with each task, so both build it here.
type TaskEnv struct {
	NodeID              string
	IncludeScriptOutput bool
	// Deadline is formatted as given.
	Deadline     time.Time
	WorkflowName string
	// TraceContext is the task's W3C trace context, keyed as
	// propagation.TraceContext names its fields.
	TraceContext map[string]string
	// ProgressFile, ProgressPatchTickDuration and ProgressFileTickDuration
	// are set only when progress is monitored (ProgressMonitored).
	ProgressFile              string
	ProgressPatchTickDuration time.Duration
	ProgressFileTickDuration  time.Duration
}

// ProgressMonitored reports whether a task's progress is monitored, which
// it is only when both tick durations are set.
func ProgressMonitored(patchTick, fileTick time.Duration) bool {
	return patchTick != 0 && fileTick != 0
}

// EnvVars returns the env, the trace context named as the OpenTelemetry
// environment carrier names it.
func (e TaskEnv) EnvVars() []apiv1.EnvVar {
	env := []apiv1.EnvVar{
		{Name: EnvVarNodeID, Value: e.NodeID},
		{Name: EnvVarIncludeScriptOutput, Value: strconv.FormatBool(e.IncludeScriptOutput)},
		{Name: EnvVarDeadline, Value: e.Deadline.Format(time.RFC3339)},
		{Name: EnvVarWorkflowName, Value: e.WorkflowName},
	}
	carrier := envcar.Carrier{SetEnvFunc: func(name, value string) {
		env = append(env, apiv1.EnvVar{Name: name, Value: value})
	}}
	for _, key := range (propagation.TraceContext{}).Fields() {
		if v := e.TraceContext[key]; v != "" {
			carrier.Set(key, v)
		}
	}
	if ProgressMonitored(e.ProgressPatchTickDuration, e.ProgressFileTickDuration) {
		env = append(env,
			apiv1.EnvVar{Name: EnvVarProgressFile, Value: e.ProgressFile},
			apiv1.EnvVar{Name: EnvVarProgressPatchTickDuration, Value: e.ProgressPatchTickDuration.String()},
			apiv1.EnvVar{Name: EnvVarProgressFileTickDuration, Value: e.ProgressFileTickDuration.String()},
		)
	}
	return env
}
