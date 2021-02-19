package indexes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func TestConditionsIndexFunc(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		un, _ := util.ToUnstructured(&wfv1.Workflow{})
		strings, _ := ConditionsIndexFunc(un)
		assert.Nil(t, strings)
	})
	t.Run("Some", func(t *testing.T) {
		un, _ := util.ToUnstructured(&wfv1.Workflow{Status: wfv1.WorkflowStatus{
			Conditions: wfv1.Conditions{{
				Type:    wfv1.ConditionTypePodRunning,
				Status:  metav1.ConditionTrue,
				Message: "ignored",
			}},
		}})
		strings, _ := ConditionsIndexFunc(un)
		assert.Equal(t, []string{"PodRunning/True"}, strings)
	})
}
