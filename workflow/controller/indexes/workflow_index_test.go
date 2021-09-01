package indexes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func TestWorkflowIndexFunc(t *testing.T) {
	obj := &unstructured.Unstructured{}
	wfv1.MustUnmarshal(`
apiVersion: v1
kind: Pod
metadata:
  namespace: my-ns
  labels:
    workflows.argoproj.io/workflow: my-wf
`, obj)
	v, err := MetaWorkflowIndexFunc(obj)
	if assert.NoError(t, err) {
		assert.Equal(t, []string{"my-ns/my-wf"}, v)
	}
}

func TestWorkflowIndexValue(t *testing.T) {
	assert.Equal(t, "my-ns/my-wf", WorkflowIndexValue("my-ns", "my-wf"))
}

func TestWorkflowSemaphoreKeysIndexFunc(t *testing.T) {
	t.Run("Incomplete", func(t *testing.T) {
		un, _ := util.ToUnstructured(&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					common.LabelKeyCompleted: "false",
				},
			},
			Spec: wfv1.WorkflowSpec{
				Synchronization: &wfv1.Synchronization{
					Semaphore: &wfv1.SemaphoreRef{
						ConfigMapKeyRef: &apiv1.ConfigMapKeySelector{},
					},
				},
			},
		})
		result, err := WorkflowSemaphoreKeysIndexFunc()(un)
		assert.NoError(t, err)
		assert.Len(t, result, 1)
	})
	t.Run("Complete", func(t *testing.T) {
		un, _ := util.ToUnstructured(&wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					common.LabelKeyCompleted: "true",
				},
			},
		})
		result, err := WorkflowSemaphoreKeysIndexFunc()(un)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}
