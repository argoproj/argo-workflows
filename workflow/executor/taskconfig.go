package executor

import (
	"context"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/propagators/envcar"
	"go.opentelemetry.io/otel/propagation"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// MainEnv is the per-task env a pod's main container gets on top of its
// own, as NAME=value: the task's identity, deadline, trace context and,
// when progress is monitored, where to write it. A process that hands main
// its tasks one at a time gives it this env with each.
func (c TaskConfig) MainEnv() []string {
	env := []string{
		common.EnvVarNodeID + "=" + c.NodeID,
		common.EnvVarIncludeScriptOutput + "=" + strconv.FormatBool(c.IncludeScriptOutput),
		common.EnvVarDeadline + "=" + c.Deadline.UTC().Format(time.RFC3339),
		common.EnvVarWorkflowName + "=" + c.WorkflowName,
	}
	// Named as the OpenTelemetry environment carrier names them, as the
	// controller's are for a pod.
	carrier := envcar.Carrier{SetEnvFunc: func(name, value string) { env = append(env, name+"="+value) }}
	for _, key := range (propagation.TraceContext{}).Fields() {
		if v := c.TraceContext[key]; v != "" {
			carrier.Set(key, v)
		}
	}
	if c.progressMonitored() {
		env = append(env,
			common.EnvVarProgressFile+"="+c.ProgressFile,
			common.EnvVarProgressPatchTickDuration+"="+c.ProgressPatchTickDuration.String(),
			common.EnvVarProgressFileTickDuration+"="+c.ProgressFileTickDuration.String(),
		)
	}
	return env
}

func (c TaskConfig) progressMonitored() bool {
	return c.ProgressPatchTickDuration != 0 && c.ProgressFileTickDuration != 0
}

// withTraceContext returns ctx continuing the task's trace context, or ctx
// unchanged when the task has none.
func (c TaskConfig) withTraceContext(ctx context.Context) context.Context {
	if len(c.TraceContext) == 0 {
		return ctx
	}
	return propagation.TraceContext{}.Extract(ctx, propagation.MapCarrier(c.TraceContext))
}
