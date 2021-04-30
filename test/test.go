package test

import (
	"path/filepath"
	"runtime"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var testDir string

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("could not determine test directory")
	}
	testDir = filepath.Dir(filename)
}

// LoadE2EWorkflow returns a test workflow by it's path
// Deprecated
func LoadE2EWorkflow(path string) *wfv1.Workflow {
	return wfv1.MustUnmarshalWorkflow("@" + filepath.Join(testDir, "e2e", path))
}

// LoadTestWorkflow returns a workflow relative to the test file
// Deprecated
func LoadTestWorkflow(path string) *wfv1.Workflow {
	return wfv1.MustUnmarshalWorkflow("@" + path)
}

// LoadWorkflowFromBytes returns a workflow unmarshalled from an yaml byte array
// Deprecated
func LoadWorkflowFromBytes(yamlBytes []byte) *wfv1.Workflow {
	return wfv1.MustUnmarshalWorkflow(yamlBytes)
}

// LoadTestWorkflow returns a workflow relative to the test file
// Deprecated
func LoadTestWorkflowTemplate(path string) *wfv1.WorkflowTemplate {
	return wfv1.MustUnmarshalWorkflowTemplate("@" + path)
}

// LoadWorkflowFromBytes returns a workflow unmarshalled from an yaml byte array
// Deprecated
func LoadWorkflowTemplateFromBytes(yamlBytes []byte) *wfv1.WorkflowTemplate {
	return wfv1.MustUnmarshalWorkflowTemplate(yamlBytes)
}

// Deprecated
func LoadClusterWorkflowTemplateFromBytes(yamlBytes []byte) *wfv1.ClusterWorkflowTemplate {
	return wfv1.MustUnmarshalClusterWorkflow(yamlBytes)
}
