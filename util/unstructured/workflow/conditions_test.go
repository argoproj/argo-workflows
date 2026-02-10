package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func TestGetConditions(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		un := &unstructured.Unstructured{Object: map[string]any{}}

		assert.Nil(t, GetConditions(un))
	})
	t.Run("Some", func(t *testing.T) {
		un, _ := util.ToUnstructured(&wfv1.Workflow{
			Status: wfv1.WorkflowStatus{
				Conditions: wfv1.Conditions{{Type: wfv1.ConditionTypeCompleted, Status: corev1.ConditionTrue}},
			},
		})

		assert.Equal(t, wfv1.Conditions{{Type: wfv1.ConditionTypeCompleted, Status: corev1.ConditionTrue}}, GetConditions(un))
	})
}
