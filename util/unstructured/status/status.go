package status

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Infer(un *unstructured.Unstructured) (phase string, message string) {
	phase, _, _ = unstructured.NestedString(un.Object, "status", "phase")
	message, _, _ = unstructured.NestedString(un.Object, "status", "message")
	switch phase {
	case "Pending", "Running", "Succeeded", "Failed", "Error":
	default:
		phase = "Succeeded" // otherwise, we assume it is good, unless...
		items, _, _ := unstructured.NestedSlice(un.Object, "status", "conditions")
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok && m["status"].(string) == "False" {
				phase = "Running" // ...we have false conditions
			}
		}
	}
	return phase, message
}
