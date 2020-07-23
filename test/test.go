package test

import (
	"io/ioutil"
	"path/filepath"
	"runtime"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/util"
)

var (
	testDir string
)

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("could not determine test directory")
	}
	testDir = filepath.Dir(filename)
}

// LoadE2EWorkflow returns a test workflow by it's path
func LoadE2EWorkflow(path string) *wfv1.Workflow {
	yamlBytes, err := ioutil.ReadFile(filepath.Join(testDir, "e2e", path))
	if err != nil {
		panic(err)
	}
	return LoadWorkflowFromBytes(yamlBytes)
}

// LoadTestWorkflow returns a workflow relative to the test file
func LoadTestWorkflow(path string) *wfv1.Workflow {
	yamlBytes, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return LoadWorkflowFromBytes(yamlBytes)
}

// LoadWorkflowFromBytes returns a workflow unmarshalled from an yaml byte array
func LoadWorkflowFromBytes(yamlBytes []byte) *wfv1.Workflow {
	v := &wfv1.Workflow{}
	util.MustUnmarshallYAML(string(yamlBytes), v)
	return v
}

// LoadTestWorkflow returns a workflow relative to the test file
func LoadTestWorkflowTemplate(path string) *wfv1.WorkflowTemplate {
	yamlBytes, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return LoadWorkflowTemplateFromBytes(yamlBytes)
}

// LoadWorkflowFromBytes returns a workflow unmarshalled from an yaml byte array
func LoadWorkflowTemplateFromBytes(yamlBytes []byte) *wfv1.WorkflowTemplate {
	v := &wfv1.WorkflowTemplate{}
	util.MustUnmarshallYAML(string(yamlBytes), v)
	return v
}

func LoadClusterWorkflowTemplateFromBytes(yamlBytes []byte) *wfv1.ClusterWorkflowTemplate {
	v := &wfv1.ClusterWorkflowTemplate{}
	util.MustUnmarshallYAML(string(yamlBytes), v)
	return v
}
