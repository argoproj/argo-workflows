package commands

import (
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestSubmitSimple(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("../../../test/e2e/functional/retry-paramter.yaml")
	submitOpts := util.SubmitOpts{}
	cliOpts := cliSubmitOpts{SubstituteParams: true}
	output, err := replaceGlobalParameters(replaceGlobalParameter, &submitOpts, &cliOpts)
	assert.NoError(t, err)
	workflows := unmarshalWorkflows(output[0], true)
	var ans int32 = 2
	assert.Equal(t, *workflows[0].Spec.Templates[0].RetryStrategy.Limit, ans)
}

func TestSubmitGlobalParametersComplex(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("../../../test/e2e/functional/global-parameters-complex.yaml")
	parameters := []string{`message1=goodbye world`}
	cliOpts := cliSubmitOpts{SubstituteParams: true}
	submitOpts := util.SubmitOpts{Parameters: parameters}
	output, err := replaceGlobalParameters(replaceGlobalParameter, &submitOpts, &cliOpts)
	assert.NoError(t, err)
	var wfSpec wfv1.Workflow
	yaml.Unmarshal(output[0], &wfSpec)
	assert.Equal(t, *wfSpec.Spec.Templates[0].Inputs.Parameters[1].Value, "goodbye world")
}

func TestSubmitRetryParamterCommandlineParameter(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("../../../test/e2e/functional/retry-paramter.yaml")
	parameters := []string{"retry-count=1"}
	cliOpts := cliSubmitOpts{SubstituteParams: true}
	submitOpts := util.SubmitOpts{Parameters: parameters}
	output, err := replaceGlobalParameters(replaceGlobalParameter, &submitOpts, &cliOpts)
	assert.NoError(t, err)
	workflows := unmarshalWorkflows(output[0], true)
	var ans int32 = 1
	assert.Equal(t, *workflows[0].Spec.Templates[0].RetryStrategy.Limit, ans)
}

func TestSubmitRetryParamterCommandlineParameterFile(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("../../../test/e2e/functional/retry-paramter.yaml")
	parameterfile := "../../../test/e2e/functional/parameter-file.yaml"
	cliOpts := cliSubmitOpts{SubstituteParams: true}
	submitOpts := util.SubmitOpts{ParameterFile: parameterfile}
	output, err := replaceGlobalParameters(replaceGlobalParameter, &submitOpts, &cliOpts)
	assert.NoError(t, err)
	workflows := unmarshalWorkflows(output[0], true)
	var ans int32 = 7
	assert.Equal(t, *workflows[0].Spec.Templates[0].RetryStrategy.Limit, ans)
}

func TestSubmitRetryParamterCommandlineParameterFileParameters(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("../../../test/e2e/functional/retry-paramter.yaml")
	parameterfile := "../../../test/e2e/functional/parameter-file.yaml"
	parameters := []string{"retry-count=1"}
	cliOpts := cliSubmitOpts{SubstituteParams: true}
	submitOpts := util.SubmitOpts{ParameterFile: parameterfile, Parameters: parameters}
	output, err := replaceGlobalParameters(replaceGlobalParameter, &submitOpts, &cliOpts)
	assert.NoError(t, err)
	workflows := unmarshalWorkflows(output[0], true)
	var ans int32 = 1
	assert.Equal(t, *workflows[0].Spec.Templates[0].RetryStrategy.Limit, ans)
}
