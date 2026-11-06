package executor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"go.opentelemetry.io/otel/propagation"
	"k8s.io/apimachinery/pkg/types"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/tracing"
)

// MainEnv is the per-task env a pod's main container gets on top of its
// own, as NAME=value: the task's identity, deadline, trace context and,
// when progress is monitored, where to write it. A process that hands main
// its tasks one at a time gives it this env with each. A pooled task's
// deadline is its own, so it can be earlier than a pod's ARGO_DEADLINE,
// which leaves the template's timeout and activeDeadlineSeconds to the
// pod's own activeDeadlineSeconds (pooled.TaskConfig).
func (c TaskConfig) MainEnv() []string {
	vars := common.TaskEnv{
		NodeID:                    c.NodeID,
		IncludeScriptOutput:       c.IncludeScriptOutput,
		Deadline:                  c.Deadline.UTC(),
		WorkflowName:              c.WorkflowName,
		TraceContext:              c.TraceContext,
		ProgressFile:              c.ProgressFile,
		ProgressPatchTickDuration: c.ProgressPatchTickDuration,
		ProgressFileTickDuration:  c.ProgressFileTickDuration,
	}.EnvVars()
	env := make([]string, 0, len(vars))
	for _, v := range vars {
		env = append(env, v.Name+"="+v.Value)
	}
	return env
}

// withTraceContext returns ctx continuing the task's trace context, or ctx
// unchanged when the task has none.
func (c TaskConfig) withTraceContext(ctx context.Context) context.Context {
	if len(c.TraceContext) == 0 {
		return ctx
	}
	return propagation.TraceContext{}.Extract(ctx, propagation.MapCarrier(c.TraceContext))
}

// TaskConfigFromLookup reads the per-task values the controller bakes into
// a pod for the single task a classic argoexec pod runs, looking each env
// var up with lookupEnv: the composition root (cmd/argoexec) passes
// os.LookupEnv. Without the template's env var, the template is read from
// the file the init container wrote to varRunArgo.
func TaskConfigFromLookup(lookupEnv func(string) (string, bool), varRunArgo string) (TaskConfig, error) {
	getenv := func(key string) string {
		v, _ := lookupEnv(key)
		return v
	}
	tmpl := &wfv1.Template{}
	envVarTemplateValue, ok := lookupEnv(common.EnvVarTemplate)
	var templateBytes []byte
	var err error
	if !ok {
		// Wait container reads template from the file written by init container,
		// not from the env var.
		templateBytes, err = os.ReadFile(filepath.Join(varRunArgo, "template"))
	} else {
		// Offload-sentinel resolution is shared with the emissary via
		// common.ResolveTemplateEnvValue so the offload protocol stays in one place.
		templateBytes, err = common.ResolveTemplateEnvValue(envVarTemplateValue, common.EnvConfigMountPath)
	}
	if err != nil {
		return TaskConfig{}, err
	}
	if err = json.Unmarshal(templateBytes, tmpl); err != nil {
		return TaskConfig{}, err
	}

	deadline, err := time.Parse(time.RFC3339, getenv(common.EnvVarDeadline))
	if err != nil {
		return TaskConfig{}, err
	}

	var inputPlugins []wfv1.ArtifactPluginName
	for _, p := range common.SplitPluginNames(getenv(common.EnvVarInputArtifactPluginNames)) {
		inputPlugins = append(inputPlugins, wfv1.ArtifactPluginName(p))
	}

	// errors ignored because values are set by the controller and checked there.
	progressPatchTickDuration, _ := time.ParseDuration(getenv(common.EnvVarProgressPatchTickDuration))
	progressFileTickDuration, _ := time.ParseDuration(getenv(common.EnvVarProgressFileTickDuration))

	return TaskConfig{
		WorkflowName:              getenv(common.EnvVarWorkflowName),
		WorkflowUID:               types.UID(getenv(common.EnvVarWorkflowUID)),
		NodeID:                    getenv(common.EnvVarNodeID),
		Template:                  *tmpl,
		IncludeScriptOutput:       getenv(common.EnvVarIncludeScriptOutput) == "true",
		Deadline:                  deadline,
		ProgressFile:              getenv(common.EnvVarProgressFile),
		ProgressPatchTickDuration: progressPatchTickDuration,
		ProgressFileTickDuration:  progressFileTickDuration,
		TraceContext:              tracing.TraceContextFromEnv(lookupEnv),
		InputArtifactPluginNames:  inputPlugins,
	}, nil
}
