package util

import (
	"sigs.k8s.io/yaml"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func MustUnmarshallWorkflow(text string) *wfv1.Workflow {
	v := &wfv1.Workflow{}
	err := yaml.UnmarshalStrict([]byte(text), v)
	if err != nil {
		panic(err)
	}
	return v
}
