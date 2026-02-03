package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

var inMemoryDataNode = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: artifact-passing-z9j6n
spec:
  activeDeadlineSeconds: 300
  entrypoint: artifact-example
  templates:
  - inputs: {}
    name: artifact-example
    steps:
    - - 
        name: generate-artifact
        template: generate-artifacts
    - - arguments:
          artifacts:
          - name: file
            s3:
              accessKeySecret:
                key: accesskey
                name: my-minio-cred
              bucket: test
              endpoint: minio:9000
              insecure: true
              key: '{{item}}'
              secretKeySecret:
                key: secretkey
                name: my-minio-cred
          parameters:
          - name: file-name
            value: '{{item}}'
        name: process-artifact
        template: process-message
        withParam: '{{steps.generate-artifact.outputs.result}}'
    - - arguments:
          parameters:
          - name: processed
            value: '{{steps.process-artifact.outputs.parameters.processed}}'
        name: collect-artifact
        template: collect-artifacts
    - - arguments:
          parameters:
          - name: file
            value: '{{steps.collect-artifact.outputs.result}}'
        name: print-artifact
        template: print-message
  - data:
      source:
        artifactPaths:
          name: ""
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: test
            endpoint: minio:9000
            insecure: true
            key: ""
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      transformation:
      - expression: filter(data, {# endsWith ".py"})
    name: generate-artifacts
  - data:
      source:
        %s
      transformation:
      - expression: map(data, {# + ".collected"})
    inputs:
      parameters:
      - name: processed
    name: collect-artifacts
  - container:
      args:
      - cat /file && echo "{{inputs.parameters.file-name}}.processed" | tee /tmp/f.txt
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
    inputs:
      artifacts:
      - name: file
        path: /file
      parameters:
      - name: file-name
    name: process-message
    outputs:
      parameters:
      - name: processed
        valueFrom:
          path: /tmp/f.txt
  - container:
      args:
      - echo {{inputs.parameters.file}}
      command:
      - sh
      - -c
      image: alpine:3.23
    inputs:
      parameters:
      - name: file
    name: print-message
  ttlStrategy:
    secondsAfterCompletion: 600
status:
  nodes:
    artifact-passing-z9j6n:
      children:
      - artifact-passing-z9j6n-3901459921
      displayName: artifact-passing-z9j6n
      finishedAt: "2021-02-22T18:01:14Z"
      id: artifact-passing-z9j6n
      name: artifact-passing-z9j6n
      outboundNodes:
      - artifact-passing-z9j6n-1913061144
      phase: Running
      startedAt: "2021-02-22T18:01:00Z"
      templateName: artifact-example
      templateScope: local/artifact-passing-z9j6n
      type: Steps
    artifact-passing-z9j6n-612855575:
      boundaryID: artifact-passing-z9j6n
      displayName: '[2]'
      finishedAt: "2021-02-22T18:01:09Z"
      id: artifact-passing-z9j6n-612855575
      name: artifact-passing-z9j6n[2]
      phase: Running
      startedAt: "2021-02-22T18:01:09Z"
      templateScope: local/artifact-passing-z9j6n
      type: StepGroup
    artifact-passing-z9j6n-613296860:
      boundaryID: artifact-passing-z9j6n
      children:
      - artifact-passing-z9j6n-4238057504
      - artifact-passing-z9j6n-762040888
      displayName: '[1]'
      finishedAt: "2021-02-22T18:01:09Z"
      id: artifact-passing-z9j6n-613296860
      name: artifact-passing-z9j6n[1]
      outputs:
        parameters:
        - name: processed
          value: '["foo/script.py.processed","script.py.processed"]'
      phase: Succeeded
      startedAt: "2021-02-22T18:01:02Z"
      templateScope: local/artifact-passing-z9j6n
      type: StepGroup
    artifact-passing-z9j6n-762040888:
      boundaryID: artifact-passing-z9j6n
      children:
      - artifact-passing-z9j6n-612855575
      displayName: process-artifact(1:script.py)
      finishedAt: "2021-02-22T18:01:08Z"
      hostNodeName: k3d-k3s-default-server-0
      id: artifact-passing-z9j6n-762040888
      name: artifact-passing-z9j6n[1].process-artifact(1:script.py)
      phase: Succeeded
      startedAt: "2021-02-22T18:01:02Z"
      templateName: process-message
      templateScope: local/artifact-passing-z9j6n
      type: Pod
    artifact-passing-z9j6n-2656044291:
      boundaryID: artifact-passing-z9j6n
      children:
      - artifact-passing-z9j6n-613296860
      displayName: generate-artifact
      finishedAt: "2021-02-22T18:01:01Z"
      id: artifact-passing-z9j6n-2656044291
      name: artifact-passing-z9j6n[0].generate-artifact
      outputs:
        result: '["foo/script.py","script.py"]'
      phase: Succeeded
      startedAt: "2021-02-22T18:01:00Z"
      templateName: generate-artifacts
      templateScope: local/artifact-passing-z9j6n
      type: Pod
    artifact-passing-z9j6n-3901312826:
      boundaryID: artifact-passing-z9j6n
      children:
      - artifact-passing-z9j6n-1913061144
      displayName: '[3]'
      finishedAt: "2021-02-22T18:01:14Z"
      id: artifact-passing-z9j6n-3901312826
      name: artifact-passing-z9j6n[3]
      phase: Succeeded
      startedAt: "2021-02-22T18:01:09Z"
      templateScope: local/artifact-passing-z9j6n
      type: StepGroup
    artifact-passing-z9j6n-3901459921:
      boundaryID: artifact-passing-z9j6n
      children:
      - artifact-passing-z9j6n-2656044291
      displayName: '[0]'
      finishedAt: "2021-02-22T18:01:02Z"
      id: artifact-passing-z9j6n-3901459921
      name: artifact-passing-z9j6n[0]
      phase: Succeeded
      startedAt: "2021-02-22T18:01:00Z"
      templateScope: local/artifact-passing-z9j6n
      type: StepGroup
    artifact-passing-z9j6n-4238057504:
      boundaryID: artifact-passing-z9j6n
      displayName: process-artifact(0:foo/script.py)
      finishedAt: "2021-02-22T18:01:08Z"
      hostNodeName: k3d-k3s-default-server-0
      id: artifact-passing-z9j6n-4238057504
      name: artifact-passing-z9j6n[1].process-artifact(0:foo/script.py)
      phase: Succeeded
      startedAt: "2021-02-22T18:01:02Z"
      templateName: process-message
      templateScope: local/artifact-passing-z9j6n
      type: Pod
  phase: Running
  startedAt: "2021-02-22T18:01:00Z"
`

// Test that a pod is created when necessary
func TestDataTemplateCreatesPod(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(fmt.Sprintf(inMemoryDataNode, `artifactPaths: {s3: {bucket: "test"}}`))
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.FindByDisplayName("collect-artifact")
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodePending, node.Phase)
}
