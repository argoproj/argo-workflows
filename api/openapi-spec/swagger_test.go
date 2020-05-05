package openapi_spec

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

type obj = map[string]interface{}

func Test(t *testing.T) {
	swagger := obj{}
	data, err := ioutil.ReadFile("swagger.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(data, &swagger)
	if err != nil {
		panic(err)
	}
	definitions := swagger["definitions"].(obj)
	// one definition from each API
	t.Run("io.argoproj.workflow.v1alpha1.CreateCronWorkflowRequest", func(t *testing.T) {
		assert.Contains(t, definitions, "io.argoproj.workflow.v1alpha1.CreateCronWorkflowRequest")
	})
	t.Run("io.argoproj.workflow.v1alpha1.WorkflowCreateRequest", func(t *testing.T) {
		assert.Contains(t, definitions, "io.argoproj.workflow.v1alpha1.WorkflowCreateRequest")
	})
	t.Run("io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateCreateRequest", func(t *testing.T) {
		assert.Contains(t, definitions, "io.argoproj.workflow.v1alpha1.ClusterWorkflowTemplateCreateRequest")
	})
	t.Run("io.argoproj.workflow.v1alpha1.WorkflowTemplateCreateRequest", func(t *testing.T) {
		assert.Contains(t, definitions, "io.argoproj.workflow.v1alpha1.WorkflowTemplateCreateRequest")
	})
	t.Run("io.argoproj.workflow.v1alpha1.WorkflowTemplateCreateRequest", func(t *testing.T) {
		assert.Contains(t, definitions, "io.argoproj.workflow.v1alpha1.WorkflowTemplateCreateRequest")
	})
	t.Run("io.argoproj.workflow.v1alpha1.InfoResponse", func(t *testing.T) {
		assert.Contains(t, definitions, "io.argoproj.workflow.v1alpha1.InfoResponse")
	})
	// this test makes sure we deal with `inline`
	t.Run("io.argoproj.workflow.v1alpha1.UserContainer", func(t *testing.T) {
		definition := definitions["io.argoproj.workflow.v1alpha1.UserContainer"].(obj)
		properties := definition["properties"]
		assert.Contains(t, properties, "image")
	})
	// this test makes sure we can deal with an instance where we are wrong vs Kuberenetes
	t.Run("io.k8s.api.core.v1.SecretKeySelector", func(t *testing.T) {
		definition := definitions["io.k8s.api.core.v1.SecretKeySelector"].(obj)
		properties := definition["properties"]
		assert.Contains(t, properties, "name")
	})
	// this test makes sure we can deal with an instance where we are wrong vs Kuberenetes
	t.Run("io.k8s.api.core.v1.Volume", func(t *testing.T) {
		definition := definitions["io.k8s.api.core.v1.Volume"].(obj)
		properties := definition["properties"]
		assert.Contains(t, properties, "name")
		assert.NotContains(t, properties, "volumeSource")
	})
}
