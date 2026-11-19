package executor

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
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
	assert.Equal(t, TaskConfig{
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
	varRunArgo := t.TempDir()
	_, err = TaskConfigFromLookup(lookup, varRunArgo)
	require.ErrorIs(t, err, fs.ErrNotExist, "without the env, the template is read from the shared directory")
	require.NoError(t, os.WriteFile(filepath.Join(varRunArgo, "template"), []byte(`{"name":"from-file"}`), 0o600))
	cfg, err = TaskConfigFromLookup(lookup, varRunArgo)
	require.NoError(t, err)
	assert.Equal(t, "from-file", cfg.Template.Name)
}
