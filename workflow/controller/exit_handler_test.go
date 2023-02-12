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

var testWorkflowOnExitStepsHttpReconciliation = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-647r7
spec:
  arguments: {}
  entrypoint: whalesay
  onExit: exit-handler
  templates:
  - container:
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
  - inputs: {}
    metadata: {}
    name: exit-handler
    outputs: {}
    steps:
    - - arguments: {}
        name: run-example-com
        template: example-com
  - http:
      url: https://example.com
    inputs: {}
    metadata: {}
    name: example-com
    outputs: {}
status:
  nodes:
    hello-world-647r7:
      displayName: hello-world-647r7
      finishedAt: "2021-12-09T04:11:35Z"
      hostNodeName: dev-capact-control-plane
      id: hello-world-647r7
      name: hello-world-647r7
      outputs:
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      startedAt: "2021-12-09T04:11:30Z"
      templateName: whalesay
      templateScope: local/hello-world-647r7
      type: Pod
    hello-world-647r7-206029318:
      children:
      - hello-world-647r7-1045616760
      displayName: hello-world-647r7.onExit
      finishedAt: null
      id: hello-world-647r7-206029318
      name: hello-world-647r7.onExit
      phase: Running
      progress: 0/1
      startedAt: "2021-12-09T04:11:36Z"
      templateName: exit-handler
      templateScope: local/hello-world-647r7
      type: Steps
    hello-world-647r7-1045616760:
      boundaryID: hello-world-647r7-206029318
      children:
      - hello-world-647r7-370991976
      displayName: '[0]'
      finishedAt: null
      id: hello-world-647r7-1045616760
      name: hello-world-647r7.onExit[0]
      phase: Running
      progress: 0/1
      startedAt: "2021-12-09T04:11:36Z"
      templateScope: local/hello-world-647r7
      type: StepGroup
  phase: Running
  startedAt: "2021-12-09T04:11:30Z"
`

func TestWorkflowOnExitStepsHttpReconciliation(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testWorkflowOnExitStepsHttpReconciliation)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)

	taskSets, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("").List(ctx, v1.ListOptions{})
	if assert.NoError(t, err) {
		assert.Len(t, taskSets.Items, 0)
	}

	woc.operate(ctx)

	assert.Len(t, woc.wf.Status.Nodes, 4)
	taskSets, err = woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("").List(ctx, v1.ListOptions{})
	if assert.NoError(t, err) {
		assert.Len(t, taskSets.Items, 1)
	}
}

func TestWorkflowOnExitWorkflowStatus(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: exit-handler-with-param-96rrj
spec:
  entrypoint: first
  templates:
  - dag:
      tasks:
      - hooks:
          exit:
            arguments:
              parameters:
              - name: message
                value: '{{tasks.step-1.status}}'
            template: exit
        name: step-1
        template: output
    name: first
  - container:
      args:
      - echo -n hello world > /tmp/hello_world.txt
      command:
      - sh
      - -c
      image: python:alpine3.6
      name: ""
    name: output
    outputs:
      parameters:
      - name: result
        valueFrom:
          default: Foobar
          path: /tmp/hello_world.txt
  - inputs:
      parameters:
      - name: message
    name: exit
    script:
      command:
      - python
      image: python:alpine3.6
      name: ""
      source: |
        print("{{inputs.parameters.message}}")
status:
  nodes:
    exit-handler-with-param-96rrj:
      children:
      - exit-handler-with-param-96rrj-588897729
      displayName: exit-handler-with-param-96rrj
      finishedAt: "2022-08-17T15:59:10Z"
      id: exit-handler-with-param-96rrj
      name: exit-handler-with-param-96rrj
      outboundNodes:
      - exit-handler-with-param-96rrj-588897729
      phase: Running
      progress: 2/2
      resourcesDuration:
        cpu: 7
        memory: 5
      startedAt: "2022-08-17T15:58:59Z"
      templateName: first
      templateScope: local/exit-handler-with-param-96rrj
      type: DAG
    exit-handler-with-param-96rrj-588897729:
      boundaryID: exit-handler-with-param-96rrj
      children:
      - exit-handler-with-param-96rrj-1481430296
      displayName: step-1
      finishedAt: "2022-08-17T15:59:03Z"
      hostNodeName: kind-control-plane
      id: exit-handler-with-param-96rrj-588897729
      name: exit-handler-with-param-96rrj.step-1
      outputs:
        artifacts:
        - name: main-logs
          s3:
            key: exit-handler-with-param-96rrj/exit-handler-with-param-96rrj-output-588897729/main.log
        exitCode: "0"
        parameters:
        - name: result
          value: hello world
          valueFrom:
            default: Foobar
            path: /tmp/hello_world.txt
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 4
        memory: 3
      startedAt: "2022-08-17T15:58:59Z"
      templateName: output
      templateScope: local/exit-handler-with-param-96rrj
      type: Pod
#    exit-handler-with-param-96rrj-1481430296:
#      boundaryID: exit-handler-with-param-96rrj
#      displayName: step-1.onExit
#      finishedAt: "2022-08-17T15:59:09Z"
#      hostNodeName: kind-control-plane
#      id: exit-handler-with-param-96rrj-1481430296
#      inputs:
#        parameters:
#        - name: message
#          value: Succeeded
#      name: exit-handler-with-param-96rrj.step-1.onExit
#      outputs:
#        artifacts:
#        - name: main-logs
#          s3:
#            key: exit-handler-with-param-96rrj/exit-handler-with-param-96rrj-exit-1481430296/main.log
#        exitCode: "0"
#      phase: Succeeded
#      progress: 1/1
#      resourcesDuration:
#        cpu: 3
#        memory: 2
#      startedAt: "2022-08-17T15:59:05Z"
#      templateName: exit
#      templateScope: local/exit-handler-with-param-96rrj
#      type: Pod
  phase: Running
  progress: 2/2
`)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)

	taskSets, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets("").List(ctx, v1.ListOptions{})
	if assert.NoError(t, err) {
		assert.Len(t, taskSets.Items, 0)
	}
	woc.operate(ctx)
	assert.Equal(t, woc.wf.Status.Phase, wfv1.WorkflowRunning)
}
