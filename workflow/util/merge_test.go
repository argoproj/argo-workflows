package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var origWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  arguments:
    parameters:
    - name: message
      value: original
  entrypoint: start
  onExit: end
  serviceAccountName: argo
  workflowTemplateRef:
    name: workflow-template-submittable
`
var patchWF = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: workflow-template-hello-world-
spec:
  arguments:
    parameters:
    - name: message
      value: patch
  serviceAccountName: argo1
  podGC:
    strategy: OnPodSuccess
`

func TestMergeWorkflows(t *testing.T) {
	patchWf := unmarshalWF(origWF)
	targetWf := unmarshalWF(patchWF)

	err := MergeTo(patchWf, targetWf)
	assert.NoError(t, err)
	assert.Equal(t, "start", targetWf.Spec.Entrypoint)
	assert.Equal(t, "argo1", targetWf.Spec.ServiceAccountName)
	assert.Equal(t, "message", targetWf.Spec.Arguments.Parameters[0].Name)
	assert.Equal(t, "patch", targetWf.Spec.Arguments.Parameters[0].Value.String())
}
