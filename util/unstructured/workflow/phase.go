package workflow

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func GetPhase(un *unstructured.Unstructured) wfv1.WorkflowPhase {
	p, _, _ := unstructured.NestedString(un.Object, "status", "phase")
	return wfv1.WorkflowPhase(p)
}
