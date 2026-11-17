package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

var containerSetOutputWorkflow = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: outputs-result-
  name: outputs-result-test
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: a
            template: group
          - name: b
            template: verify
            arguments:
              parameters:
                - name: x
                  value: "{{tasks.a.outputs.result}}"
            dependencies: [ "a" ]

    - name: group
      containerSet:
        containers:
          - name: main
            image: python:alpine3.23
            command: ["python", "-c", "print('hi')"]

    - name: verify
      inputs:
        parameters:
          - name: x
      script:
        image: python:alpine3.23
        command: ["python"]
        source: |
          print("verified")
`

func TestContainerSetImplicitOutput(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(containerSetOutputWorkflow)
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Simulate DAG node
	dagNodeID := wf.Name
	woc.wf.Status.Nodes = make(wfv1.Nodes)
	woc.wf.Status.Nodes[dagNodeID] = wfv1.NodeStatus{
		ID:            dagNodeID,
		Name:          wf.Name,
		TemplateName:  "main",
		Phase:         wfv1.NodeRunning,
		Type:          wfv1.NodeTypeDAG,
		TemplateScope: "local/main", // Simplified scope
	}

	// Simulate node 'a' is running
	nodeName := wf.Name + ".a"
	nodeID := "node-a-id"
	woc.wf.Status.Nodes[nodeID] = wfv1.NodeStatus{
		ID:           nodeID,
		Name:         nodeName,
		TemplateName: "group",
		Phase:        wfv1.NodeRunning,
		Type:         wfv1.NodeTypePod,
		BoundaryID:   dagNodeID,
	}

	// Pod has finished successfully
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName, // In real world this is usually different but for assessNodeStatus it mostly matters for logs
			Labels: map[string]string{
				"workflows.argoproj.io/workflow": wf.Name,
			},
			Namespace: "default",
		},
		Status: apiv1.PodStatus{
			Phase: apiv1.PodSucceeded,
			ContainerStatuses: []apiv1.ContainerStatus{
				{
					Name: "main",
					State: apiv1.ContainerState{
						Terminated: &apiv1.ContainerStateTerminated{
							ExitCode: 0,
						},
					},
				},
			},
		},
	}

	// Call assessNodeStatus
	// Since outputs are NOT yet in the node status, it should return NodeRunning (old phase)
	// because includeScriptOutput logic should detect we need a result.
	node := woc.wf.Status.Nodes[nodeID]
	updated := woc.assessNodeStatus(ctx, pod, &node)

	// With the fix, this should be Running. Without the fix, it would be Succeeded.
	assert.Equal(t, wfv1.NodeRunning, updated.Phase, "Should stay Running waiting for outputs")

	// Now simulate outputs populated (e.g. by taskResultReconciliation)
	// We update the map
	node = woc.wf.Status.Nodes[nodeID]
	nodeWithOutputs := node.DeepCopy()
	nodeWithOutputs.Outputs = &wfv1.Outputs{
		Result: new("hi"),
	}
	woc.wf.Status.Nodes[nodeID] = *nodeWithOutputs

	// Refresh node from map
	node = woc.wf.Status.Nodes[nodeID]

	updated = woc.assessNodeStatus(ctx, pod, &node)
	assert.Equal(t, wfv1.NodeSucceeded, updated.Phase, "Should be Succeeded now that outputs are present")
}
