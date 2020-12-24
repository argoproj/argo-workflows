package workflow

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func GetConditions(un *unstructured.Unstructured) []wfv1.Condition {
	if un == nil {
		return nil
	}
	items, _, _ := unstructured.NestedSlice(un.Object, "status", "conditions")
	var y []wfv1.Condition
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil
		}
		y = append(y, wfv1.Condition{
			Type:    wfv1.ConditionType(m["type"].(string)),
			Status:  metav1.ConditionStatus(m["status"].(string)),
			Message: m["message"].(string),
		})
	}
	return y
}
