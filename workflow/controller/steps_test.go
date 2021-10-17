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

var testContinueOnExitCode = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2021-10-17T16:06:58Z"
  generateName: steps-
  generation: 13
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Failed
  name: steps-5jrq5
  namespace: argo
  resourceVersion: "10518341"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/workflows/steps-5jrq5
  uid: 2e10dcec-5eb6-4aa6-a2dd-faa5ad66d28f
spec:
  activeDeadlineSeconds: 300
  arguments: {}
  entrypoint: hello-hello-hello
  podSpecPatch: |
    terminationGracePeriodSeconds: 3
  templates:
  - inputs: {}
    metadata: {}
    name: hello-hello-hello
    outputs: {}
    steps:
    - - arguments:
          parameters:
          - name: message
            value: hello1
        name: hello1
        template: whalesay
    - - arguments:
          parameters:
          - name: message
            value: hello2a
        name: hello2a
        template: whalesay
      - arguments:
          parameters:
          - name: message
            value: hello2b
        continueOn: {}
        name: hello2b
        template: intentional-fail
    - - arguments:
          parameters:
          - name: message
            value: hello3
        name: hello3
        template: whalesay
  - container:
      args:
      - '{{inputs.parameters.message}}'
      command:
      - cowsay
      image: docker/whalesay
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: message
    metadata: {}
    name: whalesay
    outputs: {}
  - container:
      args:
      - echo intentional failure; exit 9
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: intentional-fail
    outputs: {}
  ttlStrategy:
    secondsAfterCompletion: 600
status:
  artifactRepositoryRef:
    artifactRepository:
      archiveLogs: true
      s3:
        accessKeySecret:
          key: accesskey
          name: my-minio-cred
        bucket: my-bucket
        endpoint: minio:9000
        insecure: true
        secretKeySecret:
          key: secretkey
          name: my-minio-cred
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  conditions:
  - status: "False"
    type: PodRunning
  - status: "True"
    type: Completed
  finishedAt: "2021-10-17T16:07:21Z"
  message: child 'steps-5jrq5-99117897' failed
  nodes:
    steps-5jrq5:
      children:
      - steps-5jrq5-3023201322
      displayName: steps-5jrq5
      finishedAt: "2021-10-17T16:07:21Z"
      id: steps-5jrq5
      message: child 'steps-5jrq5-99117897' failed
      name: steps-5jrq5
      outboundNodes:
      - steps-5jrq5-48785040
      - steps-5jrq5-99117897
      phase: Failed
      progress: 3/3
      resourcesDuration:
        cpu: 19
        memory: 10
      startedAt: "2021-10-17T16:06:58Z"
      templateName: hello-hello-hello
      templateScope: local/steps-5jrq5
      type: Steps
    steps-5jrq5-48785040:
      boundaryID: steps-5jrq5
      displayName: hello2a
      finishedAt: "2021-10-17T16:07:20Z"
      hostNodeName: docker-desktop
      id: steps-5jrq5-48785040
      inputs:
        parameters:
        - name: message
          value: hello2a
      name: steps-5jrq5[1].hello2a
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: steps-5jrq5/steps-5jrq5-whalesay-48785040/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 9
        memory: 5
      startedAt: "2021-10-17T16:07:08Z"
      templateName: whalesay
      templateScope: local/steps-5jrq5
      type: Pod
    steps-5jrq5-99117897:
      boundaryID: steps-5jrq5
      displayName: hello2b
      finishedAt: "2021-10-17T16:07:15Z"
      hostNodeName: docker-desktop
      id: steps-5jrq5-99117897
      message: Error (exit code 9)
      name: steps-5jrq5[1].hello2b
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: steps-5jrq5/steps-5jrq5-intentional-fail-99117897/main.log
        exitCode: "9"
      phase: Failed
      progress: 1/1
      resourcesDuration:
        cpu: 4
        memory: 2
      startedAt: "2021-10-17T16:07:08Z"
      templateName: intentional-fail
      templateScope: local/steps-5jrq5
      type: Pod
    steps-5jrq5-2955943751:
      boundaryID: steps-5jrq5
      children:
      - steps-5jrq5-48785040
      - steps-5jrq5-99117897
      displayName: '[1]'
      finishedAt: "2021-10-17T16:07:21Z"
      id: steps-5jrq5-2955943751
      message: child 'steps-5jrq5-99117897' failed
      name: steps-5jrq5[1]
      phase: Failed
      progress: 2/2
      resourcesDuration:
        cpu: 13
        memory: 7
      startedAt: "2021-10-17T16:07:08Z"
      templateScope: local/steps-5jrq5
      type: StepGroup
    steps-5jrq5-3023201322:
      boundaryID: steps-5jrq5
      children:
      - steps-5jrq5-3383135015
      displayName: '[0]'
      finishedAt: "2021-10-17T16:07:08Z"
      id: steps-5jrq5-3023201322
      name: steps-5jrq5[0]
      phase: Succeeded
      progress: 3/3
      resourcesDuration:
        cpu: 19
        memory: 10
      startedAt: "2021-10-17T16:06:58Z"
      templateScope: local/steps-5jrq5
      type: StepGroup
    steps-5jrq5-3383135015:
      boundaryID: steps-5jrq5
      children:
      - steps-5jrq5-2955943751
      displayName: hello1
      finishedAt: "2021-10-17T16:07:07Z"
      hostNodeName: docker-desktop
      id: steps-5jrq5-3383135015
      inputs:
        parameters:
        - name: message
          value: hello1
      name: steps-5jrq5[0].hello1
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: steps-5jrq5/steps-5jrq5-whalesay-3383135015/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 6
        memory: 3
      startedAt: "2021-10-17T16:06:58Z"
      templateName: whalesay
      templateScope: local/steps-5jrq5
      type: Pod
  phase: Failed
  progress: 3/3
  resourcesDuration:
    cpu: 19
    memory: 10
  startedAt: "2021-10-17T16:06:58Z"
`

func TestContinueOnExitCode(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testContinueOnExitCode)
	cancel, controller := newController(wf)
	defer cancel()

	woc := newWorkflowOperationCtx(wf, controller)

	ctx := context.Background()
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}
