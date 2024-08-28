package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

const stepsFailFast = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2021-03-12T15:28:29Z"
  name: seq-loop-pz4hh
spec:
  activeDeadlineSeconds: 300
  arguments:
    parameters:
    - name: items
      value: |
        ["a", "b", "c"]
  entrypoint: seq-loop
  templates:
  - failFast: true
    inputs:
      parameters:
      - name: items
    name: seq-loop
    parallelism: 1
    steps:
    - - name: iteration
        template: iteration
        withParam: '{{inputs.parameters.items}}'
  - name: iteration
    steps:
    - - name: step1
        template: succeed-step
    - - name: step2
        template: failed-step
  - container:
      args:
      - exit 0
      command:
      - /bin/sh
      - -c
      image: alpine
    name: succeed-step
  - container:
      args:
      - exit 1
      command:
      - /bin/sh
      - -c
      image: alpine
    name: failed-step
    retryStrategy:
      limit: 1
  ttlStrategy:
    secondsAfterCompletion: 600
status:
  nodes:
    seq-loop-pz4hh:
      children:
      - seq-loop-pz4hh-3652003332
      displayName: seq-loop-pz4hh
      id: seq-loop-pz4hh
      inputs:
        parameters:
        - name: items
          value: |
            ["a", "b", "c"]
      name: seq-loop-pz4hh
      outboundNodes:
      - seq-loop-pz4hh-4172612902
      phase: Running
      startedAt: "2021-03-12T15:28:29Z"
      templateName: seq-loop
      templateScope: local/seq-loop-pz4hh
      type: Steps
    seq-loop-pz4hh-347271843:
      boundaryID: seq-loop-pz4hh-1269516111
      displayName: step2(0)
      finishedAt: "2021-03-12T15:28:39Z"
      hostNodeName: k3d-k3s-default-server-0
      id: seq-loop-pz4hh-347271843
      message: Error (exit code 1)
      name: seq-loop-pz4hh[0].iteration(0:a)[1].step2(0)
      phase: Failed
      startedAt: "2021-03-12T15:28:33Z"
      templateName: failed-step
      templateScope: local/seq-loop-pz4hh
      type: Pod
    seq-loop-pz4hh-1269516111:
      boundaryID: seq-loop-pz4hh
      children:
      - seq-loop-pz4hh-3596771579
      displayName: iteration(0:a)
      id: seq-loop-pz4hh-1269516111
      name: seq-loop-pz4hh[0].iteration(0:a)
      outboundNodes:
      - seq-loop-pz4hh-4172612902
      phase: Running
      startedAt: "2021-03-12T15:28:29Z"
      templateName: iteration
      templateScope: local/seq-loop-pz4hh
      type: Steps
    seq-loop-pz4hh-1287186880:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-347271843
      - seq-loop-pz4hh-4172612902
      displayName: step2
      id: seq-loop-pz4hh-1287186880
      name: seq-loop-pz4hh[0].iteration(0:a)[1].step2
      phase: Failed
      startedAt: "2021-03-12T15:28:33Z"
      templateName: failed-step
      templateScope: local/seq-loop-pz4hh
      type: Retry
    seq-loop-pz4hh-3596771579:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-4031713604
      displayName: '[0]'
      finishedAt: "2021-03-12T15:28:33Z"
      id: seq-loop-pz4hh-3596771579
      name: seq-loop-pz4hh[0].iteration(0:a)[0]
      phase: Succeeded
      startedAt: "2021-03-12T15:28:29Z"
      templateScope: local/seq-loop-pz4hh
      type: StepGroup
    seq-loop-pz4hh-3652003332:
      boundaryID: seq-loop-pz4hh
      children:
      - seq-loop-pz4hh-1269516111
      displayName: '[0]'
      id: seq-loop-pz4hh-3652003332
      name: seq-loop-pz4hh[0]
      phase: Running
      startedAt: "2021-03-12T15:28:29Z"
      templateScope: local/seq-loop-pz4hh
      type: StepGroup
    seq-loop-pz4hh-3664029150:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-1287186880
      displayName: '[1]'
      id: seq-loop-pz4hh-3664029150
      name: seq-loop-pz4hh[0].iteration(0:a)[1]
      phase: Running
      startedAt: "2021-03-12T15:28:33Z"
      templateScope: local/seq-loop-pz4hh
      type: StepGroup
    seq-loop-pz4hh-4031713604:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-3664029150
      displayName: step1
      finishedAt: "2021-03-12T15:28:32Z"
      hostNodeName: k3d-k3s-default-server-0
      id: seq-loop-pz4hh-4031713604
      name: seq-loop-pz4hh[0].iteration(0:a)[0].step1
      phase: Succeeded
      startedAt: "2021-03-12T15:28:29Z"
      templateName: succeed-step
      templateScope: local/seq-loop-pz4hh
      type: Pod
    seq-loop-pz4hh-4172612902:
      boundaryID: seq-loop-pz4hh-1269516111
      displayName: step2(1)
      finishedAt: "2021-03-12T15:28:47Z"
      hostNodeName: k3d-k3s-default-server-0
      id: seq-loop-pz4hh-4172612902
      message: Error (exit code 1)
      name: seq-loop-pz4hh[0].iteration(0:a)[1].step2(1)
      phase: Failed
      startedAt: "2021-03-12T15:28:41Z"
      templateName: failed-step
      templateScope: local/seq-loop-pz4hh
      type: Pod
  phase: Running
  startedAt: "2021-03-12T15:28:29Z"

`

func TestStepsFailFast(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(stepsFailFast)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	node := woc.wf.Status.Nodes.FindByDisplayName("iteration(0:a)")
	if assert.NotNil(t, node) {
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
	}
	node = woc.wf.Status.Nodes.FindByDisplayName("seq-loop-pz4hh")
	if assert.NotNil(t, node) {
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
	}
}

const stepsFailFastWithRetries = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2021-03-12T15:28:29Z"
  name: seq-loop-pz4hh
spec:
  activeDeadlineSeconds: 300
  arguments:
    parameters:
    - name: items
      value: |
        ["a", "b", "c"]
  entrypoint: seq-loop
  templates:
  - failFast: true
    inputs:
      parameters:
      - name: items
    name: seq-loop
    retryStrategy:
      limit: 2
      retryPolicy: Always
    parallelism: 1
    steps:
    - - name: iteration
        template: iteration
        withParam: '{{inputs.parameters.items}}'
  - name: iteration
    steps:
    - - name: step1
        template: succeed-step
    - - name: step2
        template: failed-step
  - container:
      args:
      - exit 0
      command:
      - /bin/sh
      - -c
      image: alpine
    name: succeed-step
  - container:
      args:
      - exit 1
      command:
      - /bin/sh
      - -c
      image: alpine
    name: failed-step
    retryStrategy:
      limit: 1
  ttlStrategy:
    secondsAfterCompletion: 600
status:
  nodes:
    seq-loop-pz4hh:
      children:
      - seq-loop-pz4hh-3652003332
      displayName: seq-loop-pz4hh
      id: seq-loop-pz4hh
      inputs:
        parameters:
        - name: items
          value: |
            ["a", "b", "c"]
      name: seq-loop-pz4hh
      outboundNodes:
      - seq-loop-pz4hh-4172612902
      phase: Running
      startedAt: "2021-03-12T15:28:29Z"
      templateName: seq-loop
      templateScope: local/seq-loop-pz4hh
      type: Steps
    seq-loop-pz4hh-347271843:
      boundaryID: seq-loop-pz4hh-1269516111
      displayName: step2(0)
      finishedAt: "2021-03-12T15:28:39Z"
      hostNodeName: k3d-k3s-default-server-0
      id: seq-loop-pz4hh-347271843
      message: Error (exit code 1)
      name: seq-loop-pz4hh[0].iteration(0:a)[1].step2(0)
      phase: Failed
      startedAt: "2021-03-12T15:28:33Z"
      templateName: failed-step
      templateScope: local/seq-loop-pz4hh
      type: Pod
    seq-loop-pz4hh-1269516111:
      boundaryID: seq-loop-pz4hh
      children:
      - seq-loop-pz4hh-3596771579
      displayName: iteration(0:a)
      id: seq-loop-pz4hh-1269516111
      name: seq-loop-pz4hh[0].iteration(0:a)
      outboundNodes:
      - seq-loop-pz4hh-4172612902
      phase: Running
      startedAt: "2021-03-12T15:28:29Z"
      templateName: iteration
      templateScope: local/seq-loop-pz4hh
      type: Steps
    seq-loop-pz4hh-1287186880:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-347271843
      - seq-loop-pz4hh-4172612902
      displayName: step2
      id: seq-loop-pz4hh-1287186880
      name: seq-loop-pz4hh[0].iteration(0:a)[1].step2
      phase: Failed
      startedAt: "2021-03-12T15:28:33Z"
      templateName: failed-step
      templateScope: local/seq-loop-pz4hh
      type: Retry
    seq-loop-pz4hh-3596771579:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-4031713604
      displayName: '[0]'
      finishedAt: "2021-03-12T15:28:33Z"
      id: seq-loop-pz4hh-3596771579
      name: seq-loop-pz4hh[0].iteration(0:a)[0]
      phase: Succeeded
      startedAt: "2021-03-12T15:28:29Z"
      templateScope: local/seq-loop-pz4hh
      type: StepGroup
    seq-loop-pz4hh-3652003332:
      boundaryID: seq-loop-pz4hh
      children:
      - seq-loop-pz4hh-1269516111
      displayName: '[0]'
      id: seq-loop-pz4hh-3652003332
      name: seq-loop-pz4hh[0]
      phase: Running
      startedAt: "2021-03-12T15:28:29Z"
      templateScope: local/seq-loop-pz4hh
      type: StepGroup
    seq-loop-pz4hh-3664029150:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-1287186880
      displayName: '[1]'
      id: seq-loop-pz4hh-3664029150
      name: seq-loop-pz4hh[0].iteration(0:a)[1]
      phase: Running
      startedAt: "2021-03-12T15:28:33Z"
      templateScope: local/seq-loop-pz4hh
      type: StepGroup
    seq-loop-pz4hh-4031713604:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-3664029150
      displayName: step1
      finishedAt: "2021-03-12T15:28:32Z"
      hostNodeName: k3d-k3s-default-server-0
      id: seq-loop-pz4hh-4031713604
      name: seq-loop-pz4hh[0].iteration(0:a)[0].step1
      phase: Succeeded
      startedAt: "2021-03-12T15:28:29Z"
      templateName: succeed-step
      templateScope: local/seq-loop-pz4hh
      type: Pod
    seq-loop-pz4hh-4172612902:
      boundaryID: seq-loop-pz4hh-1269516111
      displayName: step2(1)
      finishedAt: "2021-03-12T15:28:47Z"
      hostNodeName: k3d-k3s-default-server-0
      id: seq-loop-pz4hh-4172612902
      message: Error (exit code 1)
      name: seq-loop-pz4hh[0].iteration(0:a)[1].step2(1)
      phase: Failed
      startedAt: "2021-03-12T15:28:41Z"
      templateName: failed-step
      templateScope: local/seq-loop-pz4hh
      type: Pod
  phase: Running
  startedAt: "2021-03-12T15:28:29Z"

`

func TestStepsDontFailFastWithRetryStrategy(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(stepsFailFastWithRetries)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	node := woc.wf.Status.Nodes.FindByDisplayName("iteration(0:a)")
	if assert.NotNil(t, node) {
		assert.Equal(t, wfv1.NodeRunning, node.Phase)
	}
	node = woc.wf.Status.Nodes.FindByDisplayName("seq-loop-pz4hh")
	if assert.NotNil(t, node) {
		assert.Equal(t, wfv1.NodeRunning, node.Phase)
	}
}

const stepsFailFastWithFailedRetry = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2021-03-12T15:28:29Z"
  name: seq-loop-pz4hh
spec:
  activeDeadlineSeconds: 300
  arguments:
    parameters:
    - name: items
      value: |
        ["a", "b", "c"]
  entrypoint: seq-loop
  templates:
  - failFast: true
    inputs:
      parameters:
      - name: items
    name: seq-loop
    parallelism: 1
    steps:
    - - name: iteration
        template: iteration
        withParam: '{{inputs.parameters.items}}'
  - name: iteration
    steps:
    - - name: step1
        template: succeed-step
    - - name: step2
        template: failed-step
  - container:
      args:
      - exit 0
      command:
      - /bin/sh
      - -c
      image: alpine
    name: succeed-step
  - container:
      args:
      - exit 1
      command:
      - /bin/sh
      - -c
      image: alpine
    name: failed-step
    retryStrategy:
      expression: asInt(lastRetry.exitCode) != 13
      limit: 2
      retryPolicy: Always
  ttlStrategy:
    secondsAfterCompletion: 600
status:
  nodes:
    seq-loop-pz4hh:
      children:
      - seq-loop-pz4hh-3652003332
      displayName: seq-loop-pz4hh
      id: seq-loop-pz4hh
      inputs:
        parameters:
        - name: items
          value: |
            ["a", "b", "c"]
      name: seq-loop-pz4hh
      outboundNodes:
      - seq-loop-pz4hh-4172612902
      phase: Running
      startedAt: "2021-03-12T15:28:29Z"
      templateName: seq-loop
      templateScope: local/seq-loop-pz4hh
      type: Steps
    seq-loop-pz4hh-347271843:
      boundaryID: seq-loop-pz4hh-1269516111
      displayName: step2(0)
      finishedAt: "2021-03-12T15:28:39Z"
      hostNodeName: k3d-k3s-default-server-0
      id: seq-loop-pz4hh-347271843
      message: Error (exit code 13)
      outputs:
        exitCode: 13
      name: seq-loop-pz4hh[0].iteration(0:a)[1].step2(0)
      phase: Failed
      startedAt: "2021-03-12T15:28:33Z"
      templateName: failed-step
      templateScope: local/seq-loop-pz4hh
      type: Pod
    seq-loop-pz4hh-1269516111:
      boundaryID: seq-loop-pz4hh
      children:
      - seq-loop-pz4hh-3596771579
      displayName: iteration(0:a)
      id: seq-loop-pz4hh-1269516111
      name: seq-loop-pz4hh[0].iteration(0:a)
      outboundNodes:
      - seq-loop-pz4hh-4172612902
      phase: Running
      startedAt: "2021-03-12T15:28:29Z"
      templateName: iteration
      templateScope: local/seq-loop-pz4hh
      type: Steps
    seq-loop-pz4hh-1287186880:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-347271843
      - seq-loop-pz4hh-4172612902
      displayName: step2
      id: seq-loop-pz4hh-1287186880
      name: seq-loop-pz4hh[0].iteration(0:a)[1].step2
      phase: Failed
      outputs:
        exitCode: 13
      startedAt: "2021-03-12T15:28:33Z"
      templateName: failed-step
      templateScope: local/seq-loop-pz4hh
      type: Retry
    seq-loop-pz4hh-3596771579:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-4031713604
      displayName: '[0]'
      finishedAt: "2021-03-12T15:28:33Z"
      id: seq-loop-pz4hh-3596771579
      name: seq-loop-pz4hh[0].iteration(0:a)[0]
      phase: Succeeded
      startedAt: "2021-03-12T15:28:29Z"
      templateScope: local/seq-loop-pz4hh
      type: StepGroup
    seq-loop-pz4hh-3652003332:
      boundaryID: seq-loop-pz4hh
      children:
      - seq-loop-pz4hh-1269516111
      displayName: '[0]'
      id: seq-loop-pz4hh-3652003332
      name: seq-loop-pz4hh[0]
      phase: Running
      startedAt: "2021-03-12T15:28:29Z"
      templateScope: local/seq-loop-pz4hh
      type: StepGroup
    seq-loop-pz4hh-3664029150:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-1287186880
      displayName: '[1]'
      id: seq-loop-pz4hh-3664029150
      name: seq-loop-pz4hh[0].iteration(0:a)[1]
      phase: Running
      startedAt: "2021-03-12T15:28:33Z"
      templateScope: local/seq-loop-pz4hh
      type: StepGroup
    seq-loop-pz4hh-4031713604:
      boundaryID: seq-loop-pz4hh-1269516111
      children:
      - seq-loop-pz4hh-3664029150
      displayName: step1
      finishedAt: "2021-03-12T15:28:32Z"
      hostNodeName: k3d-k3s-default-server-0
      id: seq-loop-pz4hh-4031713604
      name: seq-loop-pz4hh[0].iteration(0:a)[0].step1
      phase: Succeeded
      startedAt: "2021-03-12T15:28:29Z"
      templateName: succeed-step
      templateScope: local/seq-loop-pz4hh
      type: Pod
    seq-loop-pz4hh-4172612902:
      boundaryID: seq-loop-pz4hh-1269516111
      displayName: step2(1)
      finishedAt: "2021-03-12T15:28:47Z"
      hostNodeName: k3d-k3s-default-server-0
      id: seq-loop-pz4hh-4172612902
      message: Error (exit code 1)
      name: seq-loop-pz4hh[0].iteration(0:a)[1].step2(1)
      phase: Failed
      startedAt: "2021-03-12T15:28:41Z"
      templateName: failed-step
      templateScope: local/seq-loop-pz4hh
      type: Pod
  phase: Running
  startedAt: "2021-03-12T15:28:29Z"

`

func TestStepsDontFailFastWithNoneRetryStrategy(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(stepsFailFastWithFailedRetry)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	node := woc.wf.Status.Nodes.FindByDisplayName("iteration(0:a)")
	if assert.NotNil(t, node) {
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
	}
	node = woc.wf.Status.Nodes.FindByDisplayName("seq-loop-pz4hh")
	if assert.NotNil(t, node) {
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
	}
}
