package util

import wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

// Deprecated: use MustUnmarshall
func MustUnmarshallYAML(text string, v interface{}) {
	wfv1.MustUnmarshal(text, v)
}
