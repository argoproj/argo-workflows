package commands

import (
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestSubmitComplexOne(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("/Users/niklashansson/Documents/go/src/github.com/argoproj/argo/cmd/argo/commands/testComplex.yaml")
	output, err := replaceGlobalParameters(replaceGlobalParameter)
	assert.NoError(t, err)
	workflows := unmarshalWorkflows(output[0], true)
	var ans int32 = 2
	assert.Equal(t, *workflows[0].Spec.Templates[0].RetryStrategy.Limit, ans)
}

func TestSubmitComplexTwo(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("/Users/niklashansson/Documents/go/src/github.com/argoproj/argo/cmd/argo/commands/testComplexTwo.yaml")
	output, err := replaceGlobalParameters(replaceGlobalParameter)
	assert.NoError(t, err)
	var wfSpec wfv1.Workflow
	yaml.Unmarshal(output[0], &wfSpec)
	assert.Equal(t, *wfSpec.Spec.Templates[0].Inputs.Parameters[1].Value, "hello world")
}
