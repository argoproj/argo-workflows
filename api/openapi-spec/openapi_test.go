package openapi_spec //nolint:staticcheck

import (
	"os"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/require"
)

func TestOpenAPIV3Spec(t *testing.T) {
	data, err := os.ReadFile("openapi.yaml")
	require.NoError(t, err, "openapi.yaml must exist - run 'make swagger'")

	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(data)
	require.NoError(t, err, "openapi.yaml must be parseable as OpenAPI v3")

	err = doc.Validate(loader.Context)
	require.NoError(t, err, "openapi.yaml must be valid OpenAPI v3")

	// Verify key Argo types exist in components/schemas
	t.Run("io.argoproj.workflow.v1alpha1.Workflow", func(t *testing.T) {
		_, ok := doc.Components.Schemas["io.argoproj.workflow.v1alpha1.Workflow"]
		require.True(t, ok, "expected Workflow schema in components/schemas")
	})

	t.Run("io.argoproj.workflow.v1alpha1.CronWorkflow", func(t *testing.T) {
		_, ok := doc.Components.Schemas["io.argoproj.workflow.v1alpha1.CronWorkflow"]
		require.True(t, ok, "expected CronWorkflow schema in components/schemas")
	})

	t.Run("io.argoproj.workflow.v1alpha1.WorkflowTemplate", func(t *testing.T) {
		_, ok := doc.Components.Schemas["io.argoproj.workflow.v1alpha1.WorkflowTemplate"]
		require.True(t, ok, "expected WorkflowTemplate schema in components/schemas")
	})

	// Verify key API paths exist
	t.Run("workflow list path", func(t *testing.T) {
		require.NotNil(t, doc.Paths.Find("/api/v1/workflows/{namespace}"),
			"expected workflow list path to exist")
	})

	t.Run("openapi version", func(t *testing.T) {
		require.Equal(t, "3.0.3", doc.OpenAPI, "expected OpenAPI version 3.0.0")
	})
}
