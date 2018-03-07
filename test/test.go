package test

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
	"github.com/gobuffalo/packr"
)

var (
	// Manifests is a packr box to the test manifests
	Manifests = packr.NewBox("e2e")
)

// GetWorkflow returns a test workflow by it's path
func GetWorkflow(path string) *wfv1.Workflow {
	var wf wfv1.Workflow
	err := yaml.Unmarshal(Manifests.Bytes(path), &wf)
	if err != nil {
		panic(err)
	}
	// Set the workflow name explicitly since generateName doesn't work in unit tests
	if wf.Name == "" {
		wf.Name = wf.GenerateName
	}
	return &wf
}
