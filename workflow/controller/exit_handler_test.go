package controller

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
)

var stepsOnExitTmpl = `
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
        exitTemplate: 
          template: exitContainer
          arguments:
            parameters:
            - name: input
              value: '{{steps.leafA.outputs.parameters.result}}'
        template: whalesay
    - - name: leafB
        exitTemplate: 
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
	wf := unmarshalWF(stepsOnExitTmpl)
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

var dagOnExitTmpl = `
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
        exitTemplate: 
          template: exitContainer
          arguments:
            parameters:
            - name: input
              value: '{{tasks.leafA.outputs.parameters.result}}'
        template: whalesay
      - name: leafB
        dependencies: [leafA]
        exitTemplate: 
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
	wf := unmarshalWF(dagOnExitTmpl)
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
        onExit: exitContainer
        template: whalesay
    - - name: leafB
        exitTemplate: 
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

func TestStepsTmplOnExit(t *testing.T) {
	wf := unmarshalWF(stepsTmplOnExit)
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
        onExit: exitContainer
        template: whalesay
      - name: leafB
        dependencies: [leafA]
        exitTemplate: 
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

func TestDAGOnExit(t *testing.T) {
	wf := unmarshalWF(dagOnExit)
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
