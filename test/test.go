package test

import (
	"io/ioutil"
	"path/filepath"
	"runtime"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/ghodss/yaml"
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
	return unmarshalWF(yamlBytes)
}

// LoadTestWorkflow returns a workflow relative to the test file
func LoadTestWorkflow(path string) *wfv1.Workflow {
	yamlBytes, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return unmarshalWF(yamlBytes)
}

func unmarshalWF(yamlBytes []byte) *wfv1.Workflow {
	var wf wfv1.Workflow
	err := yaml.Unmarshal(yamlBytes, &wf)
	if err != nil {
		panic(err)
	}
	return &wf

}
