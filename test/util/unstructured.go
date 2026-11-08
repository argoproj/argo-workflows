package util

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// MustUnmarshalUnstructured unmarshals an unstructured object from either a file
// (if text starts with "@"), JSON (if it starts with "{") or a YAML string.
func MustUnmarshalUnstructured(text string) *unstructured.Unstructured {
	v := &unstructured.Unstructured{}
	wfv1.MustUnmarshal(text, v)
	return v
}
