package clustertemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const cwfts = `
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-workflow-template-whalesay-template
spec:
  templates:
  - name: whalesay-template
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
---
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-workflow-template-whalesay-template
spec:
  templates:
  - name: whalesay-template
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

func TestUnmarshalCWFT(t *testing.T) {
	clusterwfts, err := unmarshalClusterWorkflowTemplates([]byte(cwfts), false)
	if assert.NoError(t, err) {
		assert.Equal(t, 2, len(clusterwfts))
	}
}
