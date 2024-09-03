package informer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func Test_objectToWorkflowTemplate(t *testing.T) {
	t.Run("NotUnstructured", func(t *testing.T) {
		v, err := objectToWorkflowTemplate(&corev1.Status{})
		require.EqualError(t, err, "malformed workflow template: expected \"*unstructured.Unstructured\", got \"*v1.Status\"")
		assert.NotNil(t, v)
	})
	t.Run("MalformedWorkflowTemplate", func(t *testing.T) {
		v, err := objectToWorkflowTemplate(&unstructured.Unstructured{Object: map[string]interface{}{
			"metadata": map[string]interface{}{"namespace": "my-ns", "name": "my-name"},
			"spec":     "ops",
		}})
		require.EqualError(t, err, "malformed workflow template \"my-ns/my-name\": cannot restore struct from: string")
		require.NotNil(t, v)
		assert.Equal(t, "my-ns", v.Namespace)
		assert.Equal(t, "my-name", v.Name)
	})
	t.Run("WorkflowTemplate", func(t *testing.T) {
		v, err := objectToWorkflowTemplate(&unstructured.Unstructured{})
		require.NoError(t, err)
		assert.Equal(t, &wfv1.WorkflowTemplate{}, v)
	})
}

func Test_objectsToWorkflowTemplates(t *testing.T) {
	assert.Len(t, objectsToWorkflowTemplates([]runtime.Object{&corev1.Status{}, &unstructured.Unstructured{}}), 2)
}
