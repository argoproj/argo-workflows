package controller

import (
	"context"
	"testing"
	"time"

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

func TestResourceDurationMetricDefaultMetricScope(t *testing.T) {
	wf := wfv1.Workflow{Status: wfv1.WorkflowStatus{StartedAt: metav1.NewTime(time.Now())}}
	woc := wfOperationCtx{
		globalParams: make(common.Parameters),
		wf:           &wf,
	}

	localScope, realTimeScope := woc.prepareDefaultMetricScope()

	assert.Equal(t, "0", localScope["resourcesDuration.cpu"])
	assert.Equal(t, "0", localScope["resourcesDuration.memory"])
	assert.Equal(t, "0", localScope["duration"])
	assert.Equal(t, "Pending", localScope["status"])
	assert.Less(t, realTimeScope["workflow.duration"](), 1.0)
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
