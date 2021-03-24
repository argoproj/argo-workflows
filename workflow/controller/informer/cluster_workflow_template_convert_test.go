package informer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func Test_objectToClusterWorkflowTemplate(t *testing.T) {
	t.Run("NotUnstructured", func(t *testing.T) {
		v, err := objectToClusterWorkflowTemplate(&corev1.Status{})
		assert.EqualError(t, err, "malformed cluster workflow template: expected \"*unstructured.Unstructured\", got \"*v1.Status\"")
		assert.NotNil(t, v)
	})
	t.Run("MalformedClusterWorkflowTemplate", func(t *testing.T) {
		v, err := objectToClusterWorkflowTemplate(&unstructured.Unstructured{Object: map[string]interface{}{
			"metadata": map[string]interface{}{"name": "my-name"},
			"spec":     "ops",
		}})
		assert.EqualError(t, err, "malformed cluster workflow template \"my-name\": cannot restore struct from: string")
		if assert.NotNil(t, v) {
			assert.Equal(t, "my-name", v.Name)
		}
	})
	t.Run("ClusterWorkflowTemplate", func(t *testing.T) {
		v, err := objectToClusterWorkflowTemplate(&unstructured.Unstructured{})
		assert.NoError(t, err)
		assert.Equal(t, &wfv1.ClusterWorkflowTemplate{}, v)
	})
}

func Test_objectsToClusterWorkflowTemplates(t *testing.T) {
	assert.Len(t, objectsToClusterWorkflowTemplates([]runtime.Object{&corev1.Status{}, &unstructured.Unstructured{}}), 2)
}
