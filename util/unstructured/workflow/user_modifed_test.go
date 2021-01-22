package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func Test_UserModified(t *testing.T) {
	t.Run("StatusOnly", func(t *testing.T) {
		a, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&wfv1.Workflow{})
		assert.NoError(t, err)
		b, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&wfv1.Workflow{Status: wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning}})
		assert.NoError(t, err)

		assert.False(t, UserModified(&unstructured.Unstructured{Object: a}, &unstructured.Unstructured{Object: b}))
	})
	t.Run("Spec", func(t *testing.T) {
		a, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&wfv1.Workflow{})
		assert.NoError(t, err)
		b, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&wfv1.Workflow{Spec: wfv1.WorkflowSpec{ServiceAccountName: "my-sa"}})
		assert.NoError(t, err)

		assert.True(t, UserModified(&unstructured.Unstructured{Object: a}, &unstructured.Unstructured{Object: b}))
	})
	t.Run("Suspend", func(t *testing.T) {
		b, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&wfv1.Workflow{
			Spec: wfv1.WorkflowSpec{
				Templates: []wfv1.Template{
					{Suspend: &wfv1.SuspendTemplate{}},
				},
			},
		})
		assert.NoError(t, err)

		assert.True(t, UserModified(&unstructured.Unstructured{Object: b}, &unstructured.Unstructured{Object: b}))
	})
}
