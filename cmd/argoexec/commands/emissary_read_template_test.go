package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestReadTemplate_PrefersFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "template")
	tmpl := wfv1.Template{Name: "from-file"}
	body, err := json.Marshal(tmpl)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filePath, body, 0o644))
	// Env var set to something else — must be ignored when file exists.
	t.Setenv(common.EnvVarTemplate, `{"name":"from-env"}`)

	data, err := readTemplateAt(filePath, dir)
	require.NoError(t, err)
	var got wfv1.Template
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "from-file", got.Name)
}

func TestReadTemplate_FallsBackToEnv(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "template") // intentionally not created
	tmpl := wfv1.Template{Name: "from-env"}
	body, err := json.Marshal(tmpl)
	require.NoError(t, err)
	t.Setenv(common.EnvVarTemplate, string(body))

	data, err := readTemplateAt(filePath, dir)
	require.NoError(t, err)
	var got wfv1.Template
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "from-env", got.Name)
}

func TestReadTemplate_HandlesOffloaded(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "template") // intentionally not created
	offloadDir := t.TempDir()
	tmpl := wfv1.Template{Name: "from-offload"}
	body, err := json.Marshal(tmpl)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(offloadDir, common.EnvVarTemplate), body, 0o644))
	t.Setenv(common.EnvVarTemplate, common.EnvVarTemplateOffloaded)

	data, err := readTemplateAt(filePath, offloadDir)
	require.NoError(t, err)
	var got wfv1.Template
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "from-offload", got.Name)
}

func TestReadTemplate_NeitherAvailable(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "template")
	os.Unsetenv(common.EnvVarTemplate)

	_, err := readTemplateAt(filePath, dir)
	require.Error(t, err)
}
