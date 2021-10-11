package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// TestStepsFailedRetries ensures a steps template will recognize exhausted retries
func TestStepsFailedRetries(t *testing.T) {
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps-failed-retries.yaml")
	woc := newWoc(*wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

var artifactResolutionWhenSkipped = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: conditional-artifact-passing-
spec:
  entrypoint: artifact-example
  templates:
  - name: artifact-example
    steps:
    - - name: generate-artifact
        template: whalesay
        when: "false"
    - - name: consume-artifact
        template: print-message
        when: "false"
        arguments:
          artifacts:
          - name: message
            from: "{{steps.generate-artifact.outputs.artifacts.hello-art}}"

  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["sleep 1; cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      artifacts:
      - name: hello-art
        path: /tmp/hello_world.txt

  - name: print-message
    inputs:
      artifacts:
      - name: message
        path: /tmp/message
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["cat /tmp/message"]

`

// Tests ability to reference workflow parameters from within top level spec fields (e.g. spec.volumes)
func TestArtifactResolutionWhenSkipped(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(artifactResolutionWhenSkipped)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

var stepsWithParamAndGlobalParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-with-param-and-global-param-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: workspace
      value: /argo_workspace/{{workflow.uid}}
  templates:
  - name: main
    steps:
    - - name: use-with-param
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "hello {{workflow.parameters.workspace}} {{item}}"
        withParam: "[0, 1, 2]"
  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

func TestStepsWithParamAndGlobalParam(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(stepsWithParamAndGlobalParam)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestResourceDurationMetric(t *testing.T) {
	nodeStatus := `
      boundaryID: many-items-z26lj
      displayName: sleep(4:four)
      finishedAt: "2020-06-02T16:04:50Z"
      hostNodeName: minikube
      id: many-items-z26lj-3491220632
      name: many-items-z26lj[0].sleep(4:four)
      outputs:
        parameters:
        - name: pipeline_tid
        artifacts:
        - archiveLogs: true
          name: main-logs
          s3:
            accessKeySecret:
              key: accesskey
              name: my-minio-cred
            bucket: my-bucket
            endpoint: minio:9000
            insecure: true
            key: many-items-z26lj/many-items-z26lj-3491220632/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 33
        memory: 24
      startedAt: "2020-06-02T16:04:21Z"
      templateName: sleep
      templateScope: local/many-items-z26lj
      type: Pod
`

	woc := wfOperationCtx{globalParams: make(common.Parameters)}
	var node wfv1.NodeStatus
	wfv1.MustUnmarshal([]byte(nodeStatus), &node)
	localScope, _ := woc.prepareMetricScope(&node)
	assert.Equal(t, "33", localScope["resourcesDuration.cpu"])
	assert.Equal(t, "24", localScope["resourcesDuration.memory"])
	assert.Equal(t, "0", localScope["exitCode"])
}

var optionalArgumentAndParameter = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: optional-input-artifact-ctc82
spec:
  
  entrypoint: plan
  templates:
  - 
    inputs: {}
    metadata: {}
    name: plan
    outputs: {}
    steps:
    - - 
        name: create-artifact
        template: artifact-creation
        when: "false"
    - - arguments:
          artifacts:
          - from: '{{steps.create-artifact.outputs.artifacts.hello}}'
            name: artifact
            optional: true
        name: print-artifact
        template: artifact-printing
  - 
    container:
      args:
      - echo 'hello' > /tmp/hello.txt
      command:
      - sh
      - -c
      image: alpine:3.11
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: artifact-creation
    outputs:
      artifacts:
      - name: hello
        path: /tmp/hello.txt
  - 
    container:
      args:
      - echo 'goodbye'
      command:
      - sh
      - -c
      image: alpine:3.11
      name: ""
      resources: {}
    inputs:
      artifacts:
      - name: artifact
        optional: true
        path: /tmp/file
    metadata: {}
    name: artifact-printing
    outputs: {}
status:
  nodes:
    optional-input-artifact-ctc82:
      children:
      - optional-input-artifact-ctc82-4087665160
      displayName: optional-input-artifact-ctc82
      finishedAt: "2020-12-08T18:40:26Z"
      id: optional-input-artifact-ctc82
      name: optional-input-artifact-ctc82
      outboundNodes:
      - optional-input-artifact-ctc82-1701987189
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2020-12-08T18:40:21Z"
      templateName: plan
      templateScope: local/optional-input-artifact-ctc82
      type: Steps
    optional-input-artifact-ctc82-3164000327:
      boundaryID: optional-input-artifact-ctc82
      children:
      - optional-input-artifact-ctc82-933325693
      displayName: create-artifact
      finishedAt: "2020-12-08T18:40:21Z"
      id: optional-input-artifact-ctc82-3164000327
      message: when 'false' evaluated false
      name: optional-input-artifact-ctc82[0].create-artifact
      phase: Skipped
      progress: 1/1
      startedAt: "2020-12-08T18:40:21Z"
      templateName: artifact-creation
      templateScope: local/optional-input-artifact-ctc82
      type: Skipped
    optional-input-artifact-ctc82-4087665160:
      boundaryID: optional-input-artifact-ctc82
      children:
      - optional-input-artifact-ctc82-3164000327
      displayName: '[0]'
      finishedAt: "2020-12-08T18:40:21Z"
      id: optional-input-artifact-ctc82-4087665160
      name: optional-input-artifact-ctc82[0]
      phase: Running
      progress: 1/1
      startedAt: "2020-12-08T18:40:21Z"
      templateName: plan
      templateScope: local/optional-input-artifact-ctc82
      type: StepGroup
  phase: Running
`

func TestOptionalArgumentAndParameter(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(optionalArgumentAndParameter)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var stepsParamAggregationWithRetryStrategy = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: parameter-aggregation-retry-steps-h8b82-
  name: parameter-aggregation-retry-steps-h8b82-dpfh7
  namespace: argon
spec:
  arguments: {}
  entrypoint: parameter-aggregation-retry
  serviceAccountName: argon-argo-workflow
  templates:
  - inputs: {}
    metadata: {}
    name: parameter-aggregation-retry
    outputs: {}
    steps:
    - - arguments:
          parameters:
          - name: num
            value: '{{item}}'
        name: odd-or-even
        template: odd-or-even
        withItems:
        - 1
        - 2
    - - arguments:
          parameters:
          - name: message
            value: '{{steps.odd-or-even.outputs.parameters.num}}'
        name: print-nums
        template: whalesay
      - arguments:
          parameters:
          - name: message
            value: '{{steps.odd-or-even.outputs.parameters.evenness}}'
        name: print-evenness
        template: whalesay
  - container:
      args:
      - |
        sleep 1 &&
        echo {{inputs.parameters.num}} > /tmp/num &&
        if [ $(({{inputs.parameters.num}}%2)) -eq 0 ]; then
          echo "even" > /tmp/even;
        else
          echo "odd" > /tmp/even;
        fi
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: num
    metadata: {}
    name: odd-or-even
    outputs:
      parameters:
      - name: num
        valueFrom:
          path: /tmp/num
      - name: evenness
        valueFrom:
          path: /tmp/even
    retryStrategy:
      limit: 10
  - container:
      args:
      - '{{inputs.parameters.message}}'
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: message
    metadata: {}
    name: whalesay
    outputs: {}
status:
  nodes:
    parameter-aggregation-retry-steps-h8b82-dpfh7:
      children:
      - parameter-aggregation-retry-steps-h8b82-dpfh7-3071098285
      displayName: parameter-aggregation-retry-steps-h8b82-dpfh7
      finishedAt: "2021-08-27T05:14:05Z"
      id: parameter-aggregation-retry-steps-h8b82-dpfh7
      name: parameter-aggregation-retry-steps-h8b82-dpfh7
      startedAt: "2021-08-27T05:12:39Z"
      templateName: parameter-aggregation-retry
      templateScope: local/parameter-aggregation-retry-steps-h8b82-dpfh7
      type: Steps
    parameter-aggregation-retry-steps-h8b82-dpfh7-400240683:
      boundaryID: parameter-aggregation-retry-steps-h8b82-dpfh7
      children:
      - parameter-aggregation-retry-steps-h8b82-dpfh7-3540252070
      displayName: odd-or-even(1:2)
      finishedAt: "2021-08-27T05:13:19Z"
      id: parameter-aggregation-retry-steps-h8b82-dpfh7-400240683
      inputs:
        parameters:
        - name: num
          value: "2"
      name: parameter-aggregation-retry-steps-h8b82-dpfh7[0].odd-or-even(1:2)
      outputs:
        exitCode: "0"
        parameters:
        - name: num
          value: "2"
          valueFrom:
            path: /tmp/num
        - name: evenness
          value: even
          valueFrom:
            path: /tmp/even
      phase: Succeeded
      startedAt: "2021-08-27T05:12:39Z"
      templateName: odd-or-even
      templateScope: local/parameter-aggregation-retry-steps-h8b82-dpfh7
      type: Retry
    parameter-aggregation-retry-steps-h8b82-dpfh7-1758157232:
      boundaryID: parameter-aggregation-retry-steps-h8b82-dpfh7
      children:
      - parameter-aggregation-retry-steps-h8b82-dpfh7-3004134904
      displayName: odd-or-even(0:1)(0)
      finishedAt: "2021-08-27T05:13:10Z"
      hostNodeName: docker-desktop
      id: parameter-aggregation-retry-steps-h8b82-dpfh7-1758157232
      inputs:
        parameters:
        - name: num
          value: "1"
      name: parameter-aggregation-retry-steps-h8b82-dpfh7[0].odd-or-even(0:1)(0)
      outputs:
        exitCode: "0"
        parameters:
        - name: num
          value: "1"
          valueFrom:
            path: /tmp/num
        - name: evenness
          value: odd
          valueFrom:
            path: /tmp/even
      phase: Succeeded
      startedAt: "2021-08-27T05:12:39Z"
      templateName: odd-or-even
      templateScope: local/parameter-aggregation-retry-steps-h8b82-dpfh7
      type: Pod
    parameter-aggregation-retry-steps-h8b82-dpfh7-2783464393:
      boundaryID: parameter-aggregation-retry-steps-h8b82-dpfh7
      children:
      - parameter-aggregation-retry-steps-h8b82-dpfh7-1758157232
      displayName: odd-or-even(0:1)
      finishedAt: "2021-08-27T05:13:19Z"
      id: parameter-aggregation-retry-steps-h8b82-dpfh7-2783464393
      inputs:
        parameters:
        - name: num
          value: "1"
      name: parameter-aggregation-retry-steps-h8b82-dpfh7[0].odd-or-even(0:1)
      outputs:
        exitCode: "0"
        parameters:
        - name: num
          value: "1"
          valueFrom:
            path: /tmp/num
        - name: evenness
          value: odd
          valueFrom:
            path: /tmp/even
      phase: Succeeded
      startedAt: "2021-08-27T05:12:39Z"
      templateName: odd-or-even
      templateScope: local/parameter-aggregation-retry-steps-h8b82-dpfh7
      type: Retry
    parameter-aggregation-retry-steps-h8b82-dpfh7-3071098285:
      boundaryID: parameter-aggregation-retry-steps-h8b82-dpfh7
      children:
      - parameter-aggregation-retry-steps-h8b82-dpfh7-2783464393
      - parameter-aggregation-retry-steps-h8b82-dpfh7-400240683
      displayName: '[0]'
      finishedAt: "2021-08-27T05:13:19Z"
      id: parameter-aggregation-retry-steps-h8b82-dpfh7-3071098285
      name: parameter-aggregation-retry-steps-h8b82-dpfh7[0]
      phase: Succeeded
      startedAt: "2021-08-27T05:12:39Z"
      templateScope: local/parameter-aggregation-retry-steps-h8b82-dpfh7
      type: StepGroup
    parameter-aggregation-retry-steps-h8b82-dpfh7-3540252070:
      boundaryID: parameter-aggregation-retry-steps-h8b82-dpfh7
      children:
      - parameter-aggregation-retry-steps-h8b82-dpfh7-3004134904
      displayName: odd-or-even(1:2)(0)
      finishedAt: "2021-08-27T05:13:14Z"
      hostNodeName: docker-desktop
      id: parameter-aggregation-retry-steps-h8b82-dpfh7-3540252070
      inputs:
        parameters:
        - name: num
          value: "2"
      name: parameter-aggregation-retry-steps-h8b82-dpfh7[0].odd-or-even(1:2)(0)
      outputs:
        exitCode: "0"
        parameters:
        - name: num
          value: "2"
          valueFrom:
            path: /tmp/num
        - name: evenness
          value: even
          valueFrom:
            path: /tmp/even
      phase: Succeeded
      startedAt: "2021-08-27T05:12:39Z"
      templateName: odd-or-even
      templateScope: local/parameter-aggregation-retry-steps-h8b82-dpfh7
      type: Pod
  phase: Succeeded
  startedAt: "2021-08-27T05:12:39Z"
`

func TestStepsParamAggregationWithRetryStrategy(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(stepsParamAggregationWithRetryStrategy)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	evenNode := woc.wf.Status.Nodes.FindByDisplayName("print-evenness")
	if assert.NotNil(t, evenNode) {
		if assert.Len(t, evenNode.Inputs.Parameters, 1) {
			assert.Equal(t, `["odd","even"]`, evenNode.Inputs.Parameters[0].Value.String())
		}
	}

	numNode := woc.wf.Status.Nodes.FindByDisplayName("print-nums")
	if assert.NotNil(t, numNode) {
		if assert.Len(t, numNode.Inputs.Parameters, 1) {
			assert.Equal(t, `["1","2"]`, numNode.Inputs.Parameters[0].Value.String())
		}
	}
}
