package controller

import (
	"context"
	"fmt"
	"github.com/argoproj/argo/workflow/common"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test"
)

// TestDagXfail verifies a DAG can fail properly
func TestDagXfail(t *testing.T) {
	wf := test.LoadTestWorkflow("testdata/dag_xfail.yaml")
	woc := newWoc(*wf)
	woc.operate()
	assert.Equal(t, string(wfv1.NodeFailed), string(woc.wf.Status.Phase))
}

// TestDagRetrySucceeded verifies a DAG will be marked Succeeded if retry was successful
func TestDagRetrySucceeded(t *testing.T) {
	wf := test.LoadTestWorkflow("testdata/dag_retry_succeeded.yaml")
	woc := newWoc(*wf)
	woc.operate()
	assert.Equal(t, string(wfv1.NodeSucceeded), string(woc.wf.Status.Phase))
}

// TestDagRetryExhaustedXfail verifies we fail properly when we exhaust our retries
func TestDagRetryExhaustedXfail(t *testing.T) {
	wf := test.LoadTestWorkflow("testdata/dag-exhausted-retries-xfail.yaml")
	woc := newWoc(*wf)
	woc.operate()
	assert.Equal(t, string(wfv1.NodeFailed), string(woc.wf.Status.Phase))
}

// TestDagDisableFailFast test disable fail fast function
func TestDagDisableFailFast(t *testing.T) {
	wf := test.LoadTestWorkflow("testdata/dag-disable-fail-fast.yaml")
	woc := newWoc(*wf)
	woc.operate()
	assert.Equal(t, string(wfv1.NodeFailed), string(woc.wf.Status.Phase))
}

var dynamicSingleDag = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
 generateName: dag-diamond-
spec:
 entrypoint: diamond
 templates:
 - name: diamond
   dag:
     tasks:
     - name: A
       template: %s
       %s
     - name: TestSingle
       template: Succeeded
       depends: A.%s

 - name: Succeeded
   container:
     image: alpine:3.7
     command: [sh, -c, "exit 0"]

 - name: Failed
   container:
     image: alpine:3.7
     command: [sh, -c, "exit 1"]

 - name: Skipped
   when: "False"
   container:
     image: alpine:3.7
     command: [sh, -c, "echo Hello"]
`

func TestSingleDependency(t *testing.T) {
	statusMap := map[string]v1.PodPhase{"Succeeded": v1.PodSucceeded, "Failed": v1.PodFailed}
	var closer context.CancelFunc
	var controller *WorkflowController
	for _, status := range []string{"Succeeded", "Failed", "Skipped"} {
		fmt.Printf("\n\n\nCurrent status %s\n\n\n", status)
		closer, controller = newController()
		wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

		// If the status is "skipped" skip the root node.
		var wfString string
		if status == "Skipped" {
			wfString = fmt.Sprintf(dynamicSingleDag, status, `when: "False == True"`, status)
		} else {
			wfString = fmt.Sprintf(dynamicSingleDag, status, "", status)
		}
		wf := unmarshalWF(wfString)
		wf, err := wfcset.Create(wf)
		assert.Nil(t, err)
		wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
		assert.Nil(t, err)
		woc := newWorkflowOperationCtx(wf, controller)

		woc.operate()
		// Mark the status of the pod according to the test
		if _, ok := statusMap[status]; ok {
			makePodsPhase(t, statusMap[status], controller.kubeclientset, wf.ObjectMeta.Namespace)
		}

		woc.operate()
		found := false
		for _, node := range woc.wf.Status.Nodes {
			if strings.Contains(node.Name, "TestSingle") {
				found = true
				assert.Equal(t, wfv1.NodePending, node.Phase)
			}
		}
		assert.True(t, found)
	}
	if closer != nil {
		closer()
	}
}

func TestGetDagTaskFromNode(t *testing.T) {
	task := wfv1.DAGTask{Name: "test-task"}
	d := dagContext{
		boundaryID: "test-boundary",
		tasks:      []wfv1.DAGTask{task},
	}
	node := wfv1.NodeStatus{Name: d.taskNodeName(task.Name)}
	taskFromNode := d.getTaskFromNode(&node)
	assert.Equal(t, &task, taskFromNode)
}

var artifactResolutionWhenSkippedDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: conditional-artifact-passing-
spec:
  entrypoint: artifact-example
  templates:
  - name: artifact-example
    dag:
      tasks:
      - name: generate-artifact
        template: whalesay
        when: "false"
      - name: consume-artifact
        dependencies: [generate-artifact]
        template: print-message
        when: "false"
        arguments:
          artifacts:
          - name: message
            from: "{{tasks.generate-artifact.outputs.artifacts.hello-art}}"

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
func TestArtifactResolutionWhenSkippedDAG(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(artifactResolutionWhenSkippedDAG)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
}

func TestEvaluateDependsLogic(t *testing.T) {
	testTasks := []wfv1.DAGTask{
		{
			Name: "A",
		},
		{
			Name:    "B",
			Depends: "A",
		},
		{
			Name:    "C", // This task should fail
			Depends: "A",
		},
		{
			Name:    "should-execute-1",
			Depends: "A && C.Completed",
		},
		{
			Name:    "should-execute-2",
			Depends: "B || C",
		},
		{
			Name:    "should-not-execute",
			Depends: "B && C",
		},
		{
			Name:    "should-execute-3",
			Depends: "should-execute-2 || should-not-execute",
		},
	}

	d := &dagContext{
		boundaryName: "test",
		tasks:        testTasks,
		wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
		dependencies: make(map[string][]string),
		dependsLogic: make(map[string]string),
	}

	// Task A is running
	d.wf = &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: map[string]wfv1.NodeStatus{
				d.taskNodeID("A"): {Phase: wfv1.NodeRunning},
			},
		},
	}

	// Task B should not proceed, task A is still running
	execute, proceed, err := d.evaluateDependsLogic("B")
	assert.NoError(t, err)
	assert.False(t, proceed)
	assert.False(t, execute)

	// Task A succeeded
	d.wf.Status.Nodes[d.taskNodeID("A")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}

	// Task B and C should proceed and execute
	execute, proceed, err = d.evaluateDependsLogic("B")
	assert.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	execute, proceed, err = d.evaluateDependsLogic("C")
	assert.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	// Other tasks should not
	execute, proceed, err = d.evaluateDependsLogic("should-execute-1")
	assert.NoError(t, err)
	assert.False(t, proceed)
	assert.False(t, execute)

	// Tasks B succeeded, C failed
	d.wf.Status.Nodes[d.taskNodeID("B")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}
	d.wf.Status.Nodes[d.taskNodeID("C")] = wfv1.NodeStatus{Phase: wfv1.NodeFailed}

	// Tasks should-execute-1 and should-execute-2 should proceed and execute
	execute, proceed, err = d.evaluateDependsLogic("should-execute-1")
	assert.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	execute, proceed, err = d.evaluateDependsLogic("should-execute-2")
	assert.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	// Task should-not-execute should proceed, but not execute
	execute, proceed, err = d.evaluateDependsLogic("should-not-execute")
	assert.NoError(t, err)
	assert.True(t, proceed)
	assert.False(t, execute)

	// Tasks should-execute-1 and should-execute-2 succeeded, should-not-execute skipped
	d.wf.Status.Nodes[d.taskNodeID("should-execute-1")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}
	d.wf.Status.Nodes[d.taskNodeID("should-execute-2")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}
	d.wf.Status.Nodes[d.taskNodeID("should-not-execute")] = wfv1.NodeStatus{Phase: wfv1.NodeSkipped}

	// Tasks should-execute-3 should proceed and execute
	execute, proceed, err = d.evaluateDependsLogic("should-execute-3")
	assert.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
}

func TestAllEvaluateDependsLogic(t *testing.T) {
	statusMap := map[common.TaskResult]wfv1.NodePhase{
		common.TaskResultSucceeded: wfv1.NodeSucceeded,
		common.TaskResultFailed: wfv1.NodeFailed,
		common.TaskResultSkipped: wfv1.NodeSkipped,
		common.TaskResultCompleted: wfv1.NodeSucceeded,
		common.TaskResultAny: wfv1.NodeSkipped,
	}
	for _, status := range []common.TaskResult{common.TaskResultSucceeded, common.TaskResultFailed, common.TaskResultSkipped,
		common.TaskResultCompleted, common.TaskResultAny} {
		testTasks := []wfv1.DAGTask{
			{
				Name: "same",
			},
			{
				Name:    "Run",
				Depends: fmt.Sprintf("same.%s", status),
			},
			{
				Name:    "NotRun",
				Depends: fmt.Sprintf("!same.%s", status),
			},
		}

		d := &dagContext{
			boundaryName: "test",
			tasks:        testTasks,
			wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
			dependencies: make(map[string][]string),
			dependsLogic: make(map[string]string),
		}

		// Task A is running
		d.wf = &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
			Status: wfv1.WorkflowStatus{
				Nodes: map[string]wfv1.NodeStatus{
					d.taskNodeID("same"): {Phase: statusMap[status]},
				},
			},
		}

		execute, proceed, err := d.evaluateDependsLogic("Run")
		assert.NoError(t, err)
		assert.True(t, proceed)
		assert.True(t, execute)
		execute, proceed, err = d.evaluateDependsLogic("NotRun")
		assert.NoError(t, err)
		assert.True(t, proceed)
		assert.False(t, execute)
	}
}
