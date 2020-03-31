package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo/workflow/util"
)

func TestSubmitSimple(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("../../../test/e2e/functional/retry-paramter.yaml")
	assert.NoError(t, err)
	submitOpts := util.SubmitOpts{}
	output, err := replaceGlobalParameters(replaceGlobalParameter, &submitOpts)
	assert.NoError(t, err)
	workflows := unmarshalWorkflows(output[0], true)
	var ans int32 = 2
	assert.Equal(t, *workflows[0].Spec.Templates[0].RetryStrategy.Limit, ans)
}

func TestSubmitRetryParamterCommandlineParameter(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("../../../test/e2e/functional/retry-paramter.yaml")
	assert.NoError(t, err)
	parameters := []string{"retry-count=1"}
	submitOpts := util.SubmitOpts{Parameters: parameters}
	output, err := replaceGlobalParameters(replaceGlobalParameter, &submitOpts)
	assert.NoError(t, err)
	workflows := unmarshalWorkflows(output[0], true)
	var ans int32 = 1
	assert.Equal(t, *workflows[0].Spec.Templates[0].RetryStrategy.Limit, ans)
}

func TestSubmitRetryParamterCommandlineParameterFile(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("../../../test/e2e/functional/retry-paramter.yaml")
	assert.NoError(t, err)
	parameterfile := "../../../test/e2e/functional/parameter-file.yaml"
	submitOpts := util.SubmitOpts{ParameterFile: parameterfile}
	output, err := replaceGlobalParameters(replaceGlobalParameter, &submitOpts)
	assert.NoError(t, err)
	workflows := unmarshalWorkflows(output[0], true)
	var ans int32 = 7
	assert.Equal(t, *workflows[0].Spec.Templates[0].RetryStrategy.Limit, ans)
}

func TestSubmitRetryParamterCommandlineParameterFileParameters(t *testing.T) {
	replaceGlobalParameter, err := util.ReadManifest("../../../test/e2e/functional/retry-paramter.yaml")
	assert.NoError(t, err)
	parameterfile := "../../../test/e2e/functional/parameter-file.yaml"
	parameters := []string{"retry-count=1"}
	submitOpts := util.SubmitOpts{ParameterFile: parameterfile, Parameters: parameters}
	output, err := replaceGlobalParameters(replaceGlobalParameter, &submitOpts)
	assert.NoError(t, err)
	workflows := unmarshalWorkflows(output[0], true)
	var ans int32 = 1
	assert.Equal(t, *workflows[0].Spec.Templates[0].RetryStrategy.Limit, ans)
}
