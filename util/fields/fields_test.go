package fields

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var sampleWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-qgpxz
spec:
  entrypoint: whalesay
  templates:
  - 
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
status:
  artifactRepositoryRef:
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "True"
    type: Completed
  finishedAt: "2020-12-01T17:30:51Z"
  nodes:
    hello-world-qgpxz:
      displayName: hello-world-qgpxz
  phase: Succeeded
  progress: 1/1
  resourcesDuration:
    cpu: 3
    memory: 1
  startedAt: "2020-12-01T17:30:46Z"
`

func TestCleanFields(t *testing.T) {
	var wf v1alpha1.Workflow
	v1alpha1.MustUnmarshal([]byte(sampleWorkflow), &wf)

	jsonWf, err := json.Marshal(wf)
	assert.NoError(t, err)

	cleanJsonWf, err := CleanFields("status.phase,metadata.name,spec.entrypoint", jsonWf)
	assert.NoError(t, err)

	var cleanWf v1alpha1.Workflow
	v1alpha1.MustUnmarshal(cleanJsonWf, &cleanWf)
	assert.NoError(t, err)

	assert.Equal(t, v1alpha1.WorkflowSucceeded, cleanWf.Status.Phase)
	assert.Equal(t, "whalesay", cleanWf.Spec.Entrypoint)
	assert.Equal(t, "hello-world-qgpxz", cleanWf.Name)

	assert.Nil(t, cleanWf.Status.Nodes)
}

func TestCleanFieldsExclude(t *testing.T) {
	var wf v1alpha1.Workflow
	v1alpha1.MustUnmarshal([]byte(sampleWorkflow), &wf)

	jsonWf, err := json.Marshal(wf)
	assert.NoError(t, err)

	cleanJsonWf, err := CleanFields("-status.phase,metadata.name,spec.entrypoint", jsonWf)
	assert.NoError(t, err)

	var cleanWf v1alpha1.Workflow
	v1alpha1.MustUnmarshal(cleanJsonWf, &cleanWf)

	assert.Empty(t, cleanWf.Status.Phase)
	assert.Empty(t, cleanWf.Spec.Entrypoint)
	assert.Empty(t, cleanWf.Name)

	assert.NotNil(t, cleanWf.Status.Nodes)
}
