package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
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
