package workflow

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func GetPhase(un *unstructured.Unstructured) wfv1.NodePhase {
	if un == nil {
		return ""
	}
	phase, _, _ := unstructured.NestedString(un.Object, "status", "phase")
	return wfv1.NodePhase(phase)
}
