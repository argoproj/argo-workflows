package controller

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestKillDaemonChildrenUnmarkPod(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, &v1alpha1.Workflow{
		Status: v1alpha1.WorkflowStatus{
			Nodes: v1alpha1.Nodes{
				"a": v1alpha1.NodeStatus{
					ID:         "a",
					BoundaryID: "a",
					Daemoned:   ptr.To(true),
				},
			},
		},
	}, controller)

	assert.NotNil(t, woc.wf.Status.Nodes["a"].Daemoned)
	// Error will be that it cannot find the pod, but we only care about the node status for this test
	woc.killDaemonedChildren(ctx, "a")
	assert.Nil(t, woc.wf.Status.Nodes["a"].Daemoned)
}

var workflowWithContainerSetPodInPending = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  annotations:
    workflows.argoproj.io/pod-name-format: v1
  generateName: container-set-termination-demo
  generation: 10
  labels:
    workflows.argoproj.io/phase: Running
    workflows.argoproj.io/resubmitted-from-workflow: container-set-termination-demob7c6c
  name: container-set-termination-demopw5vv
  namespace: argo
  resourceVersion: "88102"
  uid: 2a5a4c10-3a5c-4fb4-8931-20ac78cabfee
spec:
  entrypoint: main
  shutdown: Terminate
  templates:
  - name: main
    dag:
      tasks:
        - name: using-container-set-template
          template: problematic-container-set
  - name: problematic-container-set
    containerSet:
      containers:
      - command:
        - sh
        - -c
        - sleep 10
        image: alpine
        name: step-1
      - command:
        - sh
        - -c
        - sleep 10
        image: alpine
        name: step-2
status:
  phase: Running
  conditions:
  - status: "True"
    type: PodRunning
  finishedAt: null
  nodes:
    container-set-termination-demopw5vv:
      children:
      - container-set-termination-demopw5vv-2652912851
      displayName: container-set-termination-demopw5vv
      finishedAt: null
      id: container-set-termination-demopw5vv
      name: container-set-termination-demopw5vv
      phase: Running
      progress: 2/2
      startedAt: "2022-01-27T17:45:59Z"
      templateName: main
      templateScope: local/container-set-termination-demopw5vv
      type: DAG
    container-set-termination-demopw5vv-842041608:
      boundaryID: container-set-termination-demopw5vv
      children:
      - container-set-termination-demopw5vv-893664226
      - container-set-termination-demopw5vv-876886607
      displayName: using-container-set-template
      finishedAt: "2022-01-27T17:46:16Z"
      hostNodeName: k3d-argo-workflow-server-0
      id: container-set-termination-demopw5vv-842041608
      name: container-set-termination-demopw5vv.using-container-set-template
      phase: Pending
      progress: 1/1
      startedAt: "2022-01-27T17:46:14Z"
      templateName: problematic-container-set
      templateScope: local/container-set-termination-demopw5vv
      type: Pod
    container-set-termination-demopw5vv-876886607:
      boundaryID: container-set-termination-demopw5vv-842041608
      displayName: step-2
      finishedAt: null
      id: container-set-termination-demopw5vv-876886607
      name: container-set-termination-demopw5vv.using-container-set-template.step-2
      phase: Pending
      startedAt: "2022-01-27T17:46:14Z"
      templateName: problematic-container-set
      templateScope: local/container-set-termination-demopw5vv
      type: Container
    container-set-termination-demopw5vv-893664226:
      boundaryID: container-set-termination-demopw5vv-842041608
      displayName: step-1
      finishedAt: null
      id: container-set-termination-demopw5vv-893664226
      name: container-set-termination-demopw5vv.using-container-set-template.step-1
      phase: Pending
      startedAt: "2022-01-27T17:46:14Z"
      templateName: problematic-container-set
      templateScope: local/container-set-termination-demopw5vv
      type: Container
`

func TestHandleExecutionControlErrorMarksProvidedNode(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	workflow := v1alpha1.MustUnmarshalWorkflow(workflowWithContainerSetPodInPending)

	woc := newWorkflowOperationCtx(ctx, workflow, controller)

	containerSetNodeName := "container-set-termination-demopw5vv-842041608"

	assert.Equal(t, v1alpha1.NodePending, woc.wf.Status.Nodes[containerSetNodeName].Phase)

	woc.handleExecutionControlError(ctx, containerSetNodeName, &sync.RWMutex{}, "terminated")

	assert.Equal(t, v1alpha1.NodeFailed, woc.wf.Status.Nodes[containerSetNodeName].Phase)
}

func TestHandleExecutionControlErrorMarksChildNodes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	workflow := v1alpha1.MustUnmarshalWorkflow(workflowWithContainerSetPodInPending)

	woc := newWorkflowOperationCtx(ctx, workflow, controller)

	containerSetNodeName := "container-set-termination-demopw5vv-842041608"
	step1NodeName := "container-set-termination-demopw5vv-893664226"
	step2NodeName := "container-set-termination-demopw5vv-876886607"

	assert.Equal(t, v1alpha1.NodePending, woc.wf.Status.Nodes[step1NodeName].Phase)
	assert.Equal(t, v1alpha1.NodePending, woc.wf.Status.Nodes[step2NodeName].Phase)

	woc.handleExecutionControlError(ctx, containerSetNodeName, &sync.RWMutex{}, "terminated")

	assert.Equal(t, v1alpha1.NodeFailed, woc.wf.Status.Nodes[step1NodeName].Phase)
	assert.Equal(t, v1alpha1.NodeFailed, woc.wf.Status.Nodes[step2NodeName].Phase)
}
