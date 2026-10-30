package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/executor"
)

func TestPodSource_YieldsOnce(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	t.Setenv(common.EnvVarTemplate, `{"name":"t"}`)
	t.Setenv(common.EnvVarNodeID, "node-1")
	t.Setenv(common.EnvVarDeadline, time.Now().Format(time.RFC3339))
	dir := t.TempDir()
	s := &PodSource{VarRunArgo: dir}

	task, ok, err := s.Next(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "t", task.Config.Template.Name)
	assert.Equal(t, "node-1", task.Config.NodeID)
	assert.Equal(t, PodPaths(dir), task.Paths)

	_, ok, err = s.Next(ctx)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestTaskConfigFromLookup(t *testing.T) {
	env := map[string]string{
		common.EnvVarTemplate:                  `{"name":"t"}`,
		common.EnvVarWorkflowName:              "my-wf",
		common.EnvVarWorkflowUID:               "wf-uid",
		common.EnvVarNodeID:                    "node-1",
		common.EnvVarIncludeScriptOutput:       "true",
		common.EnvVarDeadline:                  "2026-10-27T12:00:00Z",
		common.EnvVarProgressFile:              "/var/run/argo/progress",
		common.EnvVarProgressPatchTickDuration: "1m0s",
		common.EnvVarProgressFileTickDuration:  "3s",
		common.EnvVarInputArtifactPluginNames:  "a,b",
		"TRACEPARENT":                          "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
		"TRACESTATE":                           "k=v",
	}
	lookup := func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	}
	cfg, err := TaskConfigFromLookup(lookup, t.TempDir())
	require.NoError(t, err)
	assert.Equal(t, executor.TaskConfig{
		WorkflowName:              "my-wf",
		WorkflowUID:               "wf-uid",
		NodeID:                    "node-1",
		Template:                  wfv1.Template{Name: "t"},
		IncludeScriptOutput:       true,
		Deadline:                  time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC),
		ProgressFile:              "/var/run/argo/progress",
		ProgressPatchTickDuration: time.Minute,
		ProgressFileTickDuration:  3 * time.Second,
		TraceContext: map[string]string{
			"traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
			"tracestate":  "k=v",
		},
		InputArtifactPluginNames: []wfv1.ArtifactPluginName{"a", "b"},
	}, cfg)

	delete(env, "TRACEPARENT")
	cfg, err = TaskConfigFromLookup(lookup, t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, cfg.TraceContext, "a trace state alone is no trace context")

	delete(env, common.EnvVarTemplate)
	_, err = TaskConfigFromLookup(lookup, t.TempDir())
	require.Error(t, err, "without the env, the template is read from the shared directory")
}
