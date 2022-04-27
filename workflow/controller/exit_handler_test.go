package controller

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var stepsOnExitTmpl = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-on-exit
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: leafA
        hooks:
          exit: 
            template: exitContainer
            arguments:
              parameters:
              - name: input
                value: '{{steps.leafA.outputs.parameters.result}}'
        template: whalesay
    - - name: leafB
        hooks:
          exit: 
            template: exitContainer
            arguments:
              parameters:
              - name: input
                value: '{{steps.leafB.outputs.parameters.result}}'
        template: whalesay
  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
    outputs:
      parameters:
      - name: result
        valueFrom:
          default: "welcome"
          path: /tmp/hello_world.txt
  - name: exitContainer

    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["goodbye world"]
`

func TestStepsOnExitTmpl(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(stepsOnExitTmpl)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	onExitNodeIsPresent := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

var dagOnExitTmpl = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-on-exit
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    dag:
      tasks:
      - name: leafA
        hooks:
          exit: 
            template: exitContainer
            arguments:
              parameters:
              - name: input
                value: '{{tasks.leafA.outputs.parameters.result}}'
        template: whalesay
      - name: leafB
        dependencies: [leafA]
        hooks:
          exit: 
            template: exitContainer
            arguments:
              parameters:
              - name: input
                value: '{{tasks.leafB.outputs.parameters.result}}'
        template: whalesay
  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
    outputs:
      parameters:
      - name: result
        valueFrom:
          default: "welcome"
          path: /tmp/hello_world.txt
  - name: exitContainer
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["goodbye world"]
`

func TestDAGOnExitTmpl(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(dagOnExitTmpl)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	onExitNodeIsPresent := false
	for _, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

var stepsOnExitTmplWithArt = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-on-exit
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: leafA
        hooks:
          exit: 
            template: exitContainer
            arguments:
              artifacts:
              - name: input
                from: '{{steps.leafA.outputs.artifacts.result}}'
        template: whalesay
  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
    outputs:
      artifacts:
      - name: result
        path: /tmp/hello_world.txt
  - name: exitContainer
    inputs:
      artifacts:
      - name: input
        path: /my-artifact
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["goodbye world"]
`

func TestStepsOnExitTmplWithArt(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(stepsOnExitTmplWithArt)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	for idx, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, ".leafA") {
			node.Outputs = &wfv1.Outputs{
				Artifacts: wfv1.Artifacts{
					{
						Name: "result",
						ArtifactLocation: wfv1.ArtifactLocation{
							S3: &wfv1.S3Artifact{Key: "test"},
						},
					},
				},
			}
			woc.wf.Status.Nodes[idx] = node
		}
	}
	woc1 := newWorkflowOperationCtx(woc.wf, controller)
	woc1.operate(ctx)
	onExitNodeIsPresent := false
	for _, node := range woc1.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

var dagOnExitTmplWithArt = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-on-exit
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: leafA
        hooks:
          exit: 
            template: exitContainer
            arguments:
              artifacts:
              - name: input
                from: '{{tasks.leafA.outputs.artifacts.result}}'
        template: whalesay
  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
    outputs:
      artifacts:
      - name: result
        path: /tmp/hello_world.txt
  - name: exitContainer
    inputs:
      artifacts:
      - name: input
        path: /my-artifact
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["goodbye world"]`

func TestDAGOnExitTmplWithArt(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(dagOnExitTmplWithArt)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	for idx, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, ".leafA") {
			node.Outputs = &wfv1.Outputs{
				Artifacts: wfv1.Artifacts{
					{
						Name: "result",
						ArtifactLocation: wfv1.ArtifactLocation{
							S3: &wfv1.S3Artifact{Key: "test"},
						},
					},
				},
			}
			woc.wf.Status.Nodes[idx] = node
		}
	}
	woc1 := newWorkflowOperationCtx(woc.wf, controller)
	woc1.operate(ctx)
	onExitNodeIsPresent := false
	for _, node := range woc1.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

var stepsTmplOnExit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-on-exit
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: leafA
        onExit: exitContainer1
        template: whalesay
    - - name: leafB
        hooks:
          exit: 
            template: exitContainer
            arguments:
              parameters:
              - name: input
                value: '{{steps.leafB.outputs.parameters.result}}'
        template: whalesay
  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
    outputs:
      parameters:
      - name: result
        valueFrom:
          default: "welcome"
          path: /tmp/hello_world.txt
  - name: exitContainer
    inputs:
      parameters:
      - name: input
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["goodbye world"]
  - name: exitContainer1
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["goodbye world"]
`

func TestStepsTmplOnExit(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(stepsTmplOnExit)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded, withOutputs(wfv1.Outputs{Result: pointer.StringPtr("ok"), Parameters: []wfv1.Parameter{{}}}))
	woc1 := newWorkflowOperationCtx(woc.wf, controller)
	woc1.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc1.wf.Status.Phase)
	onExitNodeIsPresent := false
	for _, node := range woc1.wf.Status.Nodes {
		if node.Phase == wfv1.NodePending && strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}

	assert.True(t, onExitNodeIsPresent)
	makePodsPhase(ctx, woc1, apiv1.PodSucceeded)
	woc2 := newWorkflowOperationCtx(woc1.wf, controller)
	woc2.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc2.wf.Status.Phase)
	makePodsPhase(ctx, woc2, apiv1.PodSucceeded)
	for idx, node := range woc2.wf.Status.Nodes {
		if strings.Contains(node.Name, ".leafB") {
			node.Outputs = &wfv1.Outputs{
				Parameters: []wfv1.Parameter{
					{
						Name:  "result",
						Value: wfv1.AnyStringPtr("Welcome"),
					},
				},
			}
			woc2.wf.Status.Nodes[idx] = node
		}
	}

	woc3 := newWorkflowOperationCtx(woc2.wf, controller)
	woc3.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc3.wf.Status.Phase)
	onExitNodeIsPresent = false
	for _, node := range woc3.wf.Status.Nodes {
		if node.Phase == wfv1.NodePending && strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

var dagOnExit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-on-exit
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    dag:
      tasks:
      - name: leafA
        onExit: exitContainer1
        template: whalesay
      - name: leafB
        dependencies: [leafA]
        hooks:
          exit: 
            template: exitContainer
            arguments:
              parameters:
              - name: input
                value: '{{tasks.leafB.outputs.parameters.result}}'
        template: whalesay
  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
    outputs:
      parameters:
      - name: result
        valueFrom:
          default: "welcome"
          path: /tmp/hello_world.txt
  - name: exitContainer
    inputs:
      parameters:
      - name: input
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["goodbye world  {{inputs.parameters.input}}"]
  - name: exitContainer1
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["goodbye world"]
`

func TestDAGOnExit(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(dagOnExit)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded, withOutputs(wfv1.Outputs{Parameters: []wfv1.Parameter{{}}}))
	woc1 := newWorkflowOperationCtx(woc.wf, controller)
	woc1.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc1.wf.Status.Phase)
	onExitNodeIsPresent := false
	for _, node := range woc1.wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)

	makePodsPhase(ctx, woc1, apiv1.PodSucceeded)
	woc2 := newWorkflowOperationCtx(woc1.wf, controller)
	woc2.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc2.wf.Status.Phase)
	makePodsPhase(ctx, woc2, apiv1.PodSucceeded)
	for idx, node := range woc2.wf.Status.Nodes {
		if strings.Contains(node.Name, ".leafB") {
			node.Outputs = &wfv1.Outputs{
				Parameters: []wfv1.Parameter{
					{
						Name:  "result",
						Value: wfv1.AnyStringPtr("Welcome"),
					},
				},
			}
			woc2.wf.Status.Nodes[idx] = node
		}
	}
	woc3 := newWorkflowOperationCtx(woc2.wf, controller)
	woc3.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc3.wf.Status.Phase)
	onExitNodeIsPresent = false
	for _, node := range woc3.wf.Status.Nodes {
		if node.Phase == wfv1.NodePending && strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

var dagOnExitAndRetryStrategy = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-workflow-with-retry-strategy8h899
spec:
  entrypoint: WORKFLOW
  templates:
  - name: WORKFLOW
    steps:
    - - name: Execute
        template: DAG
  - container:
      args:
      - -c
      - set -xe && ls -ltr /
      command:
      - sh
      image: alpine:latest
    inputs:
      parameters:
      - name: IMAGE
    name: LinuxExitHandler
  - container:
      args:
      - -c
      - set -xe && ls -ltr /
      command:
      - sh
      image: alpine:latest
    name: LinuxJobBase
    retryStrategy:
      limit: "3"
      retryPolicy: OnError
  - dag:
      tasks:
      - hooks:
          exit:
            arguments:
              parameters:
              - name: IMAGE
                value: alpine:latest
            template: LinuxExitHandler
        name: Python2Compile
        template: LinuxJobBase
      - depends: Python2Compile.Succeeded
        hooks:
          exit:
            arguments:
              parameters:
              - name: IMAGE
                value: alpine:latest
            template: LinuxExitHandler
        name: DependencyTesting
        template: LinuxJobBase
    name: DAG
status:
  nodes:
    test-workflow-with-retry-strategy8h899:
      children:
      - test-workflow-with-retry-strategy8h899-1555287363
      displayName: test-workflow-with-retry-strategy8h899
      finishedAt: "2021-07-29T16:16:47Z"
      id: test-workflow-with-retry-strategy8h899
      name: test-workflow-with-retry-strategy8h899
      outboundNodes:
      - test-workflow-with-retry-strategy8h899-4242067666
      phase: Running
      startedAt: "2021-07-29T16:16:07Z"
      templateName: WORKFLOW
      templateScope: local/test-workflow-with-retry-strategy8h899
      type: Steps
    test-workflow-with-retry-strategy8h899-379180998:
      boundaryID: test-workflow-with-retry-strategy8h899-3078096906
      displayName: DependencyTesting(0)
      finishedAt: "2021-07-29T16:16:23Z"
      id: test-workflow-with-retry-strategy8h899-379180998
      name: test-workflow-with-retry-strategy8h899[0].Execute.DependencyTesting(0)
      phase: Succeeded
      startedAt: "2021-07-29T16:16:17Z"
      templateName: LinuxJobBase
      templateScope: local/test-workflow-with-retry-strategy8h899
      type: Pod
    test-workflow-with-retry-strategy8h899-961031240:
      boundaryID: test-workflow-with-retry-strategy8h899-3078096906
      children:
      - test-workflow-with-retry-strategy8h899-3783705931
      displayName: Python2Compile(0)
      finishedAt: "2021-07-29T16:16:13Z"
      id: test-workflow-with-retry-strategy8h899-961031240
      name: test-workflow-with-retry-strategy8h899[0].Execute.Python2Compile(0)
      phase: Succeeded
      startedAt: "2021-07-29T16:16:07Z"
      templateName: LinuxJobBase
      templateScope: local/test-workflow-with-retry-strategy8h899
      type: Pod
    test-workflow-with-retry-strategy8h899-1555287363:
      boundaryID: test-workflow-with-retry-strategy8h899
      children:
      - test-workflow-with-retry-strategy8h899-3078096906
      displayName: '[0]'
      id: test-workflow-with-retry-strategy8h899-1555287363
      name: test-workflow-with-retry-strategy8h899[0]
      phase: Running
      startedAt: "2021-07-29T16:16:07Z"
      templateScope: local/test-workflow-with-retry-strategy8h899
      type: StepGroup
    test-workflow-with-retry-strategy8h899-3078096906:
      boundaryID: test-workflow-with-retry-strategy8h899
      children:
      - test-workflow-with-retry-strategy8h899-3585476721
      displayName: Execute
      id: test-workflow-with-retry-strategy8h899-3078096906
      name: test-workflow-with-retry-strategy8h899[0].Execute
      outboundNodes:
      - test-workflow-with-retry-strategy8h899-4242067666
      phase: Running
      startedAt: "2021-07-29T16:16:07Z"
      templateName: DAG
      templateScope: local/test-workflow-with-retry-strategy8h899
      type: DAG
    test-workflow-with-retry-strategy8h899-3585476721:
      boundaryID: test-workflow-with-retry-strategy8h899-3078096906
      children:
      - test-workflow-with-retry-strategy8h899-961031240
      - test-workflow-with-retry-strategy8h899-3756356520
      displayName: Python2Compile
      finishedAt: "2021-07-29T16:16:17Z"
      id: test-workflow-with-retry-strategy8h899-3585476721
      name: test-workflow-with-retry-strategy8h899[0].Execute.Python2Compile
      phase: Succeeded
      startedAt: "2021-07-29T16:16:07Z"
      templateName: LinuxJobBase
      templateScope: local/test-workflow-with-retry-strategy8h899
      type: Retry
    test-workflow-with-retry-strategy8h899-3756356520:
      boundaryID: test-workflow-with-retry-strategy8h899-3078096906
      displayName: Python2Compile.onExit
      finishedAt: "2021-07-29T16:16:33Z"
      id: test-workflow-with-retry-strategy8h899-3756356520
      inputs:
        parameters:
        - name: IMAGE
          value: alpine:latest
      name: test-workflow-with-retry-strategy8h899[0].Execute.Python2Compile.onExit
      phase: Succeeded
      startedAt: "2021-07-29T16:16:27Z"
      templateName: LinuxExitHandler
      templateScope: local/test-workflow-with-retry-strategy8h899
      type: Pod
    test-workflow-with-retry-strategy8h899-3783705931:
      boundaryID: test-workflow-with-retry-strategy8h899-3078096906
      children:
      - test-workflow-with-retry-strategy8h899-379180998
      - test-workflow-with-retry-strategy8h899-4242067666
      displayName: DependencyTesting
      finishedAt: "2021-07-29T16:16:27Z"
      id: test-workflow-with-retry-strategy8h899-3783705931
      name: test-workflow-with-retry-strategy8h899[0].Execute.DependencyTesting
      phase: Succeeded
      startedAt: "2021-07-29T16:16:17Z"
      templateName: LinuxJobBase
      templateScope: local/test-workflow-with-retry-strategy8h899
      type: Retry
    test-workflow-with-retry-strategy8h899-4242067666:
      boundaryID: test-workflow-with-retry-strategy8h899-3078096906
      displayName: DependencyTesting.onExit
      finishedAt: "2021-07-29T16:16:43Z"
      id: test-workflow-with-retry-strategy8h899-4242067666
      inputs:
        parameters:
        - name: IMAGE
          value: alpine:latest
      name: test-workflow-with-retry-strategy8h899[0].Execute.DependencyTesting.onExit
      phase: Succeeded
      startedAt: "2021-07-29T16:16:27Z"
      templateName: LinuxExitHandler
      templateScope: local/test-workflow-with-retry-strategy8h899
      type: Pod
  phase: Running
  startedAt: "2021-07-29T16:16:07Z"
`

func TestDagOnExitAndRetryStrategy(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(dagOnExitAndRetryStrategy)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

var testWorkflowOnExitHttpReconciliation = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-sx6lw
spec:
  entrypoint: whalesay
  onExit: exit-handler
  templates:
  - container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
    name: whalesay
  - http:
      url: https://example.com
    name: exit-handler
status:
  nodes:
    hello-world-sx6lw:
      displayName: hello-world-sx6lw
      finishedAt: "2021-10-27T14:38:30Z"
      hostNodeName: k3d-k3s-default-server-0
      id: hello-world-sx6lw
      name: hello-world-sx6lw
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2021-10-27T14:38:27Z"
      templateName: whalesay
      templateScope: local/hello-world-sx6lw
      type: Pod
  phase: Running
  startedAt: "2021-10-27T14:38:27Z"
`

func TestWorkflowOnExitHttpReconciliation(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testWorkflowOnExitHttpReconciliation)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)

	taskSets, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("").List(ctx, v1.ListOptions{})
	if assert.NoError(t, err) {
		assert.Len(t, taskSets.Items, 0)
	}
	woc.operate(ctx)

	assert.Len(t, woc.wf.Status.Nodes, 2)
	taskSets, err = woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("").List(ctx, v1.ListOptions{})
	if assert.NoError(t, err) {
		assert.Len(t, taskSets.Items, 1)
	}
}
