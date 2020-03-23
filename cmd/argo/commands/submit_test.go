package commands

import (
	"fmt"
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestSubmitComplex(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("/Users/niklashansson/Documents/go/src/github.com/argoproj/argo/cmd/argo/commands/testComplex.yaml")
	output, err := replaceGlobalParameters(replaceGlobalParameter)
	fmt.Println(string(output[0]))
	workflowRaw := make(map[interface{}]interface{})
	err = yaml.Unmarshal(output[0], &workflowRaw)
	fmt.Println(workflowRaw)
	assert.NoError(t, err)
	assert.Equal(t, true, true)
}

func TestSubmitComplexTwo(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("/Users/niklashansson/Documents/go/src/github.com/argoproj/argo/cmd/argo/commands/testComplexTwo.yaml")
	output, err := replaceGlobalParameters(replaceGlobalParameter)
	assert.NoError(t, err)
	fmt.Println("NOW NOW START")
	fmt.Println(string(output[0]))
	var wfSpec wfv1.Workflow
	yaml.Unmarshal(output[0], &wfSpec)
	assert.NoError(t, err)
	assert.Equal(t, true, true)
}
