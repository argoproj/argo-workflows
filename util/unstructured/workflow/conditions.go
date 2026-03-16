package workflow

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// GetConditions returns the conditions, excluding the `message` field.
func GetConditions(un *unstructured.Unstructured) wfv1.Conditions {
	if un == nil {
		return nil
	}
	items, _, _ := unstructured.NestedSlice(un.Object, "status", "conditions")
	var x wfv1.Conditions
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil
		}
		_, ok = m["type"].(string)
		if !ok {
			return nil
		}
		_, ok = m["status"].(string)
		if !ok {
			return nil
		}
		x = append(x, wfv1.Condition{
			Type:   wfv1.ConditionType(m["type"].(string)),
			Status: metav1.ConditionStatus(m["status"].(string)),
		})
	}
	return x
}
