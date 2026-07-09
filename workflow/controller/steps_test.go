package controller

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/variables"
	"github.com/argoproj/argo-workflows/v4/workflow/common/dag"
)

// TestStepsFailedRetries ensures a steps template will recognize exhausted retries
func TestStepsFailedRetries(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/steps-failed-retries.yaml")
	woc := newWoc(ctx, *wf)
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
      image: alpine:3.23
      command: [sh, -c]
      args: ["cat /tmp/message"]

`

// Tests ability to reference workflow parameters from within top level spec fields (e.g. spec.volumes)
func TestArtifactResolutionWhenSkipped(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(artifactResolutionWhenSkipped)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// The engine cascades instant completions within a single Execute call,
	// so skipped groups are processed without extra cycles. We loop here as
	// a safety net in case future changes alter the batching behavior.
	for range 10 {
		woc.operate(ctx)
		if woc.wf.Status.Phase == wfv1.WorkflowSucceeded {
			break
		}
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	}
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
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(stepsWithParamAndGlobalParam)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var stepsWithParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-with-params-
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: use-with-param
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "{{item}}"
        withParam: "[1234, \"foo\\tbar\", true, []]"
`

func TestExpandStepGroupWithParam(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(stepsWithParam)
	woc := newWoc(ctx, *wf)
	tmpl := wf.Spec.Templates[0]
	step := tmpl.Steps[0].Steps[0]
	dagTask := &wfv1.DAGTask{
		Name:      step.Name,
		Template:  step.Template,
		Arguments: step.Arguments,
		WithParam: step.WithParam,
	}

	evaluator := newDAGEvaluator(wf, &tmpl, "test", "test")
	expanded, err := evaluator.ExpandTask(ctx, *dagTask, make(map[string]string), woc)
	require.NoError(t, err)
	require.Len(t, expanded, 4)

	expectedExpandedTasks := []struct {
		Name      string
		Parameter string
	}{
		{
			Name:      "use-with-param(0:1234)",
			Parameter: "1234",
		},
		{
			Name:      "use-with-param(1:foo\tbar)",
			Parameter: "foo\tbar",
		},
		{
			Name:      "use-with-param(2:true)",
			Parameter: "true",
		},
		{
			Name:      "use-with-param(3:[])",
			Parameter: "[]",
		},
	}

	for i, expected := range expectedExpandedTasks {
		assert.Equal(t, expected.Name, expanded[i].Name)
		require.Len(t, expanded[i].Arguments.Parameters, 1)
		assert.Equal(t, expected.Parameter, expanded[i].Arguments.Parameters[0].Value.String())
	}
}

var stepsWithItems = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-with-items-
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: use-with-items
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "{{item}}"
        withItems:
          - Hello"Argo
`

func TestExpandStepGroupWithItems(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(stepsWithItems)
	woc := newWoc(ctx, *wf)
	tmpl := wf.Spec.Templates[0]
	step := tmpl.Steps[0].Steps[0]
	dagTask := &wfv1.DAGTask{
		Name:      step.Name,
		Template:  step.Template,
		Arguments: step.Arguments,
		WithItems: step.WithItems,
	}

	evaluator := newDAGEvaluator(wf, &tmpl, "test", "test")
	expanded, err := evaluator.ExpandTask(ctx, *dagTask, make(map[string]string), woc)
	require.NoError(t, err)
	require.Len(t, expanded, 1)

	assert.Equal(t, "Hello\"Argo", expanded[0].Arguments.Parameters[0].Value.String())
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

	woc := wfOperationCtx{scope: variables.NewScope()}
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
		scope: variables.NewScope(),
		wf:    &wf,
	}

	localScope, realTimeScope := woc.prepareDefaultMetricScope()

	assert.Equal(t, "0", localScope["resourcesDuration.cpu"])
	assert.Equal(t, "0", localScope["resourcesDuration.memory"])
	assert.Equal(t, "0", localScope["duration"])
	assert.Equal(t, "Pending", localScope["status"])
	assert.Less(t, realTimeScope["workflow.duration"](), 1.0)
}

// Regression: when realtime metrics are evaluated before Status.StartedAt has
// been populated (the first operate cycle of a brand-new workflow), the
// workflow.duration closure must return 0 rather than time.Since(zero-time)
// saturating to MaxInt64 nanoseconds (~9.22e9 seconds).
func TestRealTimeWorkflowDurationBeforeStartedAt(t *testing.T) {
	wf := wfv1.Workflow{}
	woc := wfOperationCtx{
		scope: variables.NewScope(),
		wf:    &wf,
	}

	_, realTimeScope := woc.prepareDefaultMetricScope()

	assert.InDelta(t, 0.0, realTimeScope["workflow.duration"](), 0)
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
      image: alpine:3.23
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
      image: alpine:3.23
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
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(optionalArgumentAndParameter)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var artifactResolutionWhenOptionalAndSubpath = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: artifact-passing-subpath-rx7f4
spec:
  entrypoint: artifact-example
  templates:
  - name: artifact-example
    steps:
    - - name: hello-world-to-file
        template: hello-world-to-file
    - - name: hello-world-to-file2
        template: hello-world-to-file2
        arguments:
          artifacts:
          - name: bar
            from: "{{steps.hello-world-to-file.outputs.artifacts.foo}}"
            optional: true
            subpath: bar.txt
        withParam: "[0, 1]"

  - name: hello-world-to-file
    container:
      image: busybox:latest
      imagePullPolicy: IfNotPresent
      command: [sh, -c]
      args: ["sleep 1; echo hello world"]
    outputs:
      artifacts:
      - name: foo
        path: /tmp/foo
        optional: true
        archive:
          none: {}

  - name: hello-world-to-file2
    inputs:
      artifacts:
      - name: bar
        path: /tmp/bar.txt
        optional: true
        archive:
          none: {}
    container:
      image: busybox:latest
      imagePullPolicy: IfNotPresent
      command: [sh, -c]
      args: ["sleep 1; echo hello world"]
status:
  nodes:
    artifact-passing-subpath-rx7f4:
      children:
      - artifact-passing-subpath-rx7f4-1763046061
      displayName: artifact-passing-subpath-rx7f4
      id: artifact-passing-subpath-rx7f4
      name: artifact-passing-subpath-rx7f4
      phase: Running
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 5
      startedAt: "2024-09-06T04:53:32Z"
      templateName: artifact-example
      templateScope: local/artifact-passing-subpath-rx7f4
      type: Steps
    artifact-passing-subpath-rx7f4-511855021:
      boundaryID: artifact-passing-subpath-rx7f4
      children:
      - artifact-passing-subpath-rx7f4-1696082680
      displayName: hello-world-to-file
      finishedAt: "2024-09-06T04:53:39Z"
      id: artifact-passing-subpath-rx7f4-511855021
      name: artifact-passing-subpath-rx7f4[0].hello-world-to-file
      outputs:
        artifacts:
        - archive:
            none: {}
          name: foo
          optional: true
          path: /tmp/foo
        - name: main-logs
          s3:
            key: artifact-passing-subpath-rx7f4/artifact-passing-subpath-rx7f4-hello-world-to-file-511855021/main.log
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 5
      startedAt: "2024-09-06T04:53:32Z"
      templateName: hello-world-to-file
      templateScope: local/artifact-passing-subpath-rx7f4
      type: Pod
    artifact-passing-subpath-rx7f4-1763046061:
      boundaryID: artifact-passing-subpath-rx7f4
      children:
      - artifact-passing-subpath-rx7f4-511855021
      displayName: '[0]'
      finishedAt: "2024-09-06T04:53:41Z"
      id: artifact-passing-subpath-rx7f4-1763046061
      name: artifact-passing-subpath-rx7f4[0]
      nodeFlag: {}
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 0
        memory: 5
      startedAt: "2024-09-06T04:53:32Z"
      templateScope: local/artifact-passing-subpath-rx7f4
      type: StepGroup
  phase: Running
  taskResultsCompletionStatus:
    artifact-passing-subpath-rx7f4-511855021: true`

func TestOptionalArgumentUseSubPathInLoop(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(artifactResolutionWhenOptionalAndSubpath)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

// Regression test: referencing {{steps.<expanded_step>.id}} where the step was expanded
// via withItems. Before the fix, buildLocalScope was not called for the StepGroup node
// when a step had withItem/withParam expansion, so steps.<step>.id was unavailable
// and caused a requeue.
var stepsStepGroupIDRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-stepgroup-id-ref
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: fanout
        template: echo
        arguments:
          parameters:
          - name: msg
            value: '{{item}}'
        withItems: [0, 1]
    - - name: use-id
        template: echo
        arguments:
          parameters:
          - name: msg
            value: '{{steps.fanout.id}}'
  - name: echo
    inputs:
      parameters:
      - name: msg
    container:
      image: alpine:3.23
      command: [echo, '{{inputs.parameters.msg}}']
status:
  nodes:
    steps-stepgroup-id-ref:
      id: steps-stepgroup-id-ref
      name: steps-stepgroup-id-ref
      displayName: steps-stepgroup-id-ref
      type: Steps
      templateName: main
      templateScope: local/steps-stepgroup-id-ref
      phase: Running
      startedAt: "2020-04-20T16:39:00Z"
      children:
      - steps-stepgroup-id-ref-3297018276
    steps-stepgroup-id-ref-3297018276:
      id: steps-stepgroup-id-ref-3297018276
      name: steps-stepgroup-id-ref[0]
      displayName: '[0]'
      type: StepGroup
      templateName: main
      templateScope: local/steps-stepgroup-id-ref
      boundaryID: steps-stepgroup-id-ref
      phase: Succeeded
      startedAt: "2020-04-20T16:39:00Z"
      finishedAt: "2020-04-20T16:39:09Z"
      children:
      - steps-stepgroup-id-ref-2590864174
      - steps-stepgroup-id-ref-2140877386
    steps-stepgroup-id-ref-2590864174:
      id: steps-stepgroup-id-ref-2590864174
      name: steps-stepgroup-id-ref[0].fanout(0:0)
      displayName: fanout(0:0)
      type: Pod
      templateName: echo
      templateScope: local/steps-stepgroup-id-ref
      boundaryID: steps-stepgroup-id-ref
      phase: Succeeded
      startedAt: "2020-04-20T16:39:00Z"
      finishedAt: "2020-04-20T16:39:06Z"
      inputs:
        parameters:
        - name: msg
          value: "0"
      outputs:
        exitCode: "0"
    steps-stepgroup-id-ref-2140877386:
      id: steps-stepgroup-id-ref-2140877386
      name: steps-stepgroup-id-ref[0].fanout(1:1)
      displayName: fanout(1:1)
      type: Pod
      templateName: echo
      templateScope: local/steps-stepgroup-id-ref
      boundaryID: steps-stepgroup-id-ref
      phase: Succeeded
      startedAt: "2020-04-20T16:39:00Z"
      finishedAt: "2020-04-20T16:39:07Z"
      inputs:
        parameters:
        - name: msg
          value: "1"
      outputs:
        exitCode: "0"
  phase: Running
  startedAt: "2020-04-20T16:39:00Z"
`

func TestStepsStepGroupIDReference(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(stepsStepGroupIDRef)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	// Verify the use-id step was created (not stuck in requeue due to missing variable)
	useIDNode := woc.wf.Status.Nodes.FindByDisplayName("use-id")
	require.NotNil(t, useIDNode, "use-id node should be created when steps.fanout.id is resolvable")

	// Verify the resolved value of steps.fanout.id matches the StepGroup node's ID
	require.NotNil(t, useIDNode.Inputs)
	require.Len(t, useIDNode.Inputs.Parameters, 1)
	assert.Equal(t, "steps-stepgroup-id-ref-3978234417", useIDNode.Inputs.Parameters[0].Value.String())
}

var stepsWhenSkipNoRequeue = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-when-skip-no-requeue-
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: A
        template: script-echo
        when: "false"
    - - name: B
        when: "{{steps.A.status}} == Succeeded"
        template: echo-with-param
        arguments:
          parameters:
          - name: msg
            value: "{{steps.A.outputs.result}}"
  - name: script-echo
    script:
      image: alpine:3.23
      command: [sh]
      source: |
        echo hello
  - name: echo-with-param
    inputs:
      parameters:
      - name: msg
    container:
      image: alpine:3.23
      command: [echo, "{{inputs.parameters.msg}}"]
`

// TestStepsWhenSkipNoRequeue verifies that a step with a "when" clause that evaluates to false
// does not cause a requeue even when other fields in the step reference outputs that don't exist.
// Scenario: A is skipped (when: "false"), so A's outputs don't exist. B has
// when: "{{steps.A.status}} == Succeeded" which evaluates to false ("Skipped == Succeeded").
// B also references {{steps.A.outputs.result}} which is unresolvable since A was skipped.
// Without the fix, the full ReplaceStrict would fail on the missing output and requeue.
// With the fix, the when clause is resolved first, evaluates to false, and B is skipped early.
func TestStepsWhenSkipNoRequeue(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(stepsWhenSkipNoRequeue)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	// Workflow should succeed: A was skipped, B's when evaluated false so B was also skipped
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)

	nodeB := woc.wf.Status.Nodes.FindByDisplayName("B")
	require.NotNil(t, nodeB)
	assert.Equal(t, wfv1.NodeSkipped, nodeB.Phase)
}

var stepsWhenExprWithParamFilter = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-when-expr-filter-
spec:
  entrypoint: main
  templates:
    - name: main
      inputs:
        parameters:
          - name: test
            value: 'true'
          - name: list
            value: "{{= concat(['always'], inputs.parameters.test == 'true' ? ['test'] : []) | toJSON() }}"
      steps:
        - - name: fst
            template: run
            when: |
              "{{= get(item, 'type') ?? 'always' }}"
              in
              ("{{= inputs.parameters.list | fromJSON() | join('","') }}","")
            withParam: |
              [
                { "name": "first", "type": "" },
                { "name": "second", "type": "always" },
                { "name": "third", "type": "test" },
                { "name": "fourth" }
              ]
            arguments:
              parameters:
                - name: name
                  value: "{{ item.name }}{{ inputs.parameters.list }}"
    - name: run
      inputs:
        parameters:
          - name: name
      container:
        image: alpine:3.23
        command: [echo]
        args: ["{{inputs.parameters.name}}"]
`

// TestStepsWhenExprWithParamFilter verifies that expression templates work correctly
// in a steps workflow with withParam expansion and a when clause that filters items
// using expression functions (concat, get, ??, toJSON, fromJSON, join).
// This mirrors a real-world pattern where a dynamic list parameter controls which
// withParam items execute.
func TestStepsWhenExprWithParamFilter(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(stepsWhenExprWithParamFilter)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	// Workflow is Running because pods haven't completed, but we can verify:
	// 1. No error occurred during expression evaluation
	// 2. All 4 items were expanded and scheduled (none were incorrectly skipped/errored)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// "first" has type "" which matches empty string in the filter list
	node0 := woc.wf.Status.Nodes.FindByDisplayName("fst(0:name:first,type:)")
	require.NotNil(t, node0)
	assert.Equal(t, wfv1.NodePending, node0.Phase)

	// "second" has type "always" which is in the filter list
	node1 := woc.wf.Status.Nodes.FindByDisplayName("fst(1:name:second,type:always)")
	require.NotNil(t, node1)
	assert.Equal(t, wfv1.NodePending, node1.Phase)

	// "third" has type "test" which is in the filter list (test=true)
	node2 := woc.wf.Status.Nodes.FindByDisplayName("fst(2:name:third,type:test)")
	require.NotNil(t, node2)
	assert.Equal(t, wfv1.NodePending, node2.Phase)

	// "fourth" has no type, defaults to "always" via ?? operator
	node3 := woc.wf.Status.Nodes.FindByDisplayName("fst(3:name:fourth)")
	require.NotNil(t, node3)
	assert.Equal(t, wfv1.NodePending, node3.Phase)
}

// TestThreeStepCrossGroupArtifactResolution tests that step [2] can reference
// artifacts from step [0] (not a direct dependency) via the preceding groups scope fix.
// This matches the artifact-passing-subpath.yaml example workflow.
func TestThreeStepCrossGroupArtifactResolution(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Build a steps template with 3 groups where [2] references [0]'s artifact
	tmpl := &wfv1.Template{
		Steps: []wfv1.ParallelSteps{
			{Steps: []wfv1.WorkflowStep{{Name: "generate-artifact", Template: "gen"}}},
			{Steps: []wfv1.WorkflowStep{{Name: "list-artifact", Template: "list", Arguments: wfv1.Arguments{
				Artifacts: []wfv1.Artifact{{Name: "message", From: "{{steps.generate-artifact.outputs.artifacts.hello-art}}"}},
			}}}},
			{Steps: []wfv1.WorkflowStep{{Name: "consume-artifact", Template: "consume", Arguments: wfv1.Arguments{
				Artifacts: []wfv1.Artifact{{Name: "message", From: "{{steps.generate-artifact.outputs.artifacts.hello-art}}", SubPath: "hello_world.txt"}},
			}}}},
		},
	}

	wfName := "test-three-step-artifact"
	wf := &wfv1.Workflow{}
	wf.Name = wfName
	wf.Status.Nodes = make(wfv1.Nodes)

	// Create the [0].generate-artifact node (completed with artifact outputs)
	genNodeName := wfName + "[0].generate-artifact"
	genNodeID := wf.NodeID(genNodeName)
	wf.Status.Nodes[genNodeID] = wfv1.NodeStatus{
		ID:    genNodeID,
		Name:  genNodeName,
		Phase: wfv1.NodeSucceeded,
		Type:  wfv1.NodeTypePod,
		Outputs: &wfv1.Outputs{
			Artifacts: []wfv1.Artifact{
				{
					Name: "hello-art",
					ArtifactLocation: wfv1.ArtifactLocation{
						S3: &wfv1.S3Artifact{
							S3Bucket: wfv1.S3Bucket{Bucket: "my-bucket"},
							Key:      "outputs/hello-art",
						},
					},
				},
			},
		},
	}

	// Create the [1].list-artifact node (completed, no outputs)
	listNodeName := wfName + "[1].list-artifact"
	listNodeID := wf.NodeID(listNodeName)
	wf.Status.Nodes[listNodeID] = wfv1.NodeStatus{
		ID:    listNodeID,
		Name:  listNodeName,
		Phase: wfv1.NodeSucceeded,
		Type:  wfv1.NodeTypePod,
	}

	// Build the step adapters (same way executeSteps does)
	var tasks []dag.Task
	var prevStepNames []string
	for i, stepGroup := range tmpl.Steps {
		var currentStepNames []string
		for _, step := range stepGroup.Steps {
			s := step // capture
			task := &StepAdapter{
				step:         &s,
				dependencies: prevStepNames,
				groupIndex:   i,
			}
			tasks = append(tasks, task)
			currentStepNames = append(currentStepNames, task.GetName())
		}
		prevStepNames = currentStepNames
	}

	// The third task is [2].consume-artifact
	consumeTask := tasks[2]
	require.Equal(t, "[2].consume-artifact", consumeTask.GetName())
	require.Equal(t, []string{"[1].list-artifact"}, consumeTask.GetDependencies())

	// Create a minimal woc and Engine
	woc := &wfOperationCtx{
		wf:    wf,
		scope: variables.NewScope(),
		log:   logging.RequireLoggerFromContext(ctx),
	}
	engine := &Engine{
		woc:      woc,
		nodeName: wfName,
		tmpl:     tmpl,
		log:      woc.log,
	}
	engine.evaluator = dag.NewDAGEvaluatorFromTasks(wf, tasks, tmpl, "", wfName)

	// Build scope for [2].consume-artifact
	scope, err := engine.buildLocalScopeFromTask(ctx, consumeTask)
	require.NoError(t, err, "buildLocalScopeFromTask should not error")

	// Verify the scope contains the artifact from step [0]
	artKey := "steps.generate-artifact.outputs.artifacts.hello-art"
	_, val, resolveErr := scope.resolveVar("{{" + artKey + "}}")
	require.NoError(t, resolveErr, "scope should resolve %s", artKey)
	require.NotNil(t, val, "resolved artifact should not be nil")

	// Verify it's an actual artifact
	art, ok := val.(wfv1.Artifact)
	require.True(t, ok, "resolved value should be a wfv1.Artifact, got %T", val)
	assert.Equal(t, "hello-art", art.Name)
	assert.Equal(t, "outputs/hello-art", art.S3.Key)
}

var stepsInstantCompletionCascade = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-cascade-test
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: step-a
        template: echo
        when: "false"
    - - name: step-b
        template: echo
        when: "false"
    - - name: step-c
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestStepsInstantCompletionCascades proves that the engine cascades instant
// completions (when-skipped groups) within a single operate() call.
// If the old N-cycle regression claim were true, step-c would NOT exist after
// one operate() because groups [0] and [1] would each consume a separate cycle.
func TestStepsInstantCompletionCascades(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(stepsInstantCompletionCascade)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Single operate() call — should cascade through all skipped groups.
	woc.operate(ctx)

	// Groups [0] and [1] should be skipped.
	nodeA := woc.wf.Status.Nodes.FindByDisplayName("step-a")
	require.NotNil(t, nodeA, "step-a node should exist")
	assert.Equal(t, wfv1.NodeSkipped, nodeA.Phase)

	nodeB := woc.wf.Status.Nodes.FindByDisplayName("step-b")
	require.NotNil(t, nodeB, "step-b node should exist")
	assert.Equal(t, wfv1.NodeSkipped, nodeB.Phase)

	// The key assertion: step-c must exist after a single operate().
	// This proves the engine cascaded through the skipped groups in one call.
	nodeC := woc.wf.Status.Nodes.FindByDisplayName("step-c")
	require.NotNil(t, nodeC, "step-c should be reached in a single operate() call, proving cascading works")
}

var stepsSequentialGroups = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-sequential-groups
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: step-a
        template: echo
    - - name: step-b
        template: echo
  - name: echo
    container:
      image: alpine:3.23
      command: [echo, hello]
`

// TestStepsGroupsChainedNotSiblings verifies that StepGroup nodes are chained
// sequentially in the node graph rather than attached as siblings to the Steps
// root. Legacy behavior (and the expected behavior) is:
//
//	root          → [0]
//	[0]           → step-a
//	step-a (pod)  → [1]        (outbound linkage)
//	[1]           → step-b
//
// A regression during the DAG refactor attached every StepGroup directly under
// the Steps root, producing a graph where [0] and [1] were siblings — breaking
// the sequential chain required for UI rendering and outbound-node traversal.
//
// The linking must not run while [i-1] is in-flight: injecting [i] into the
// descendant chain of a running pod would poison childrenFulfilled() and break
// retry finalization / sync-lock release for the prior group. So [1] only
// appears in the graph once [0] has fulfilled.
func TestStepsGroupsChainedNotSiblings(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(stepsSequentialGroups)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Cycle 1: step-a starts but is not fulfilled. [1] is initialized (the
	// engine needs it to schedule step-b once dependencies resolve) but must
	// NOT yet be wired into the graph — doing so would break retry/sync
	// semantics for [0].
	woc.operate(ctx)

	rootID := woc.wf.NodeID(wf.Name)
	sg0ID := woc.wf.NodeID(wf.Name + "[0]")
	sg1ID := woc.wf.NodeID(wf.Name + "[1]")

	root, err := woc.wf.Status.Nodes.Get(rootID)
	require.NoError(t, err, "root node should exist")
	assert.Contains(t, root.Children, sg0ID, "[0] must be a child of the Steps root")
	assert.NotContains(t, root.Children, sg1ID,
		"[1] must NOT be a sibling of [0] under the Steps root (regression check)")

	stepA := woc.wf.Status.Nodes.FindByDisplayName("step-a")
	require.NotNil(t, stepA, "step-a pod node should exist")
	assert.NotContains(t, stepA.Children, sg1ID,
		"[1] must NOT be wired under step-a until step-a fulfills (timing invariant)")

	// Cycle 2: step-a completes. linkStepGroups now runs its wiring; step-b
	// gets scheduled under [1]; the full chain root → [0] → step-a → [1] →
	// step-b must be coherent.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	root, err = woc.wf.Status.Nodes.Get(rootID)
	require.NoError(t, err)
	assert.Equal(t, []string{sg0ID}, root.Children,
		"the Steps root must have exactly [0] as its only child")

	sg0, err := woc.wf.Status.Nodes.Get(sg0ID)
	require.NoError(t, err)
	stepA = woc.wf.Status.Nodes.FindByDisplayName("step-a")
	require.NotNil(t, stepA)
	assert.Contains(t, sg0.Children, stepA.ID, "[0] must contain step-a as a child")
	assert.Contains(t, stepA.Children, sg1ID,
		"[1] must be wired as a child of step-a (the outbound of [0]) once [0] fulfills")

	// linkStepGroups must be idempotent: [1] appears exactly once under step-a
	// even after repeated operate cycles.
	count := 0
	for _, c := range stepA.Children {
		if c == sg1ID {
			count++
		}
	}
	assert.Equal(t, 1, count, "[1] should appear exactly once in step-a.Children")

	stepB := woc.wf.Status.Nodes.FindByDisplayName("step-b")
	require.NotNil(t, stepB, "step-b should be scheduled after step-a succeeds")
	sg1, err := woc.wf.Status.Nodes.Get(sg1ID)
	require.NoError(t, err)
	assert.True(t, slices.Contains(sg1.Children, stepB.ID),
		"[1] must contain step-b as a child")
}
