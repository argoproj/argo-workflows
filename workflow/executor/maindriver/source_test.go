package maindriver

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

func templateJSON(t *testing.T, name string) []byte {
	t.Helper()
	body, err := json.Marshal(wfv1.Template{Name: name})
	require.NoError(t, err)
	return body
}

func readTemplateName(t *testing.T, s *PodSource) string {
	t.Helper()
	data, err := s.readTemplate()
	require.NoError(t, err)
	var got wfv1.Template
	require.NoError(t, json.Unmarshal(data, &got))
	return got.Name
}

func TestReadTemplate_PrefersFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "template"), templateJSON(t, "from-file"), 0o644))
	// ARGO_TEMPLATE set to something else — must be ignored when file exists.
	s := &PodSource{VarRunArgo: dir, TemplateEnv: `{"name":"from-env"}`, OffloadDir: dir}
	assert.Equal(t, "from-file", readTemplateName(t, s))
}

func TestReadTemplate_FallsBackToEnv(t *testing.T) {
	dir := t.TempDir() // template file intentionally not created
	s := &PodSource{VarRunArgo: dir, TemplateEnv: string(templateJSON(t, "from-env")), OffloadDir: dir}
	assert.Equal(t, "from-env", readTemplateName(t, s))
}

func TestReadTemplate_HandlesOffloaded(t *testing.T) {
	dir := t.TempDir() // template file intentionally not created
	offloadDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(offloadDir, common.EnvVarTemplate), templateJSON(t, "from-offload"), 0o644))
	s := &PodSource{VarRunArgo: dir, TemplateEnv: common.EnvVarTemplateOffloaded, OffloadDir: offloadDir}
	assert.Equal(t, "from-offload", readTemplateName(t, s))
}

func TestReadTemplate_NeitherAvailable(t *testing.T) {
	dir := t.TempDir()
	_, err := (&PodSource{VarRunArgo: dir, OffloadDir: dir}).readTemplate()
	require.Error(t, err)
}
