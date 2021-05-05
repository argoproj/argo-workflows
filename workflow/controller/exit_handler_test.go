package controller

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"

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
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc1 := newWorkflowOperationCtx(woc.wf, controller)
	woc1.operate(ctx)
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
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
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

	makePodsPhase(ctx, woc1, apiv1.PodSucceeded)
	woc2 := newWorkflowOperationCtx(woc1.wf, controller)
	woc2.operate(ctx)
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
	onExitNodeIsPresent = false
	for _, node := range woc3.wf.Status.Nodes {
		if node.Phase == wfv1.NodePending && strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}
