package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// TestDagXfail verifies a DAG can fail properly
func TestDagXfail(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag_xfail.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// TestDagRetrySucceeded verifies a DAG will be marked Succeeded if retry was successful
func TestDagRetrySucceeded(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag_retry_succeeded.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestDagRetryExhaustedXfail verifies we fail properly when we exhaust our retries
func TestDagRetryExhaustedXfail(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag-exhausted-retries-xfail.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

// TestDagDisableFailFast test disable fail fast function
func TestDagDisableFailFast(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag-disable-fail-fast.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
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
     image: alpine:3.23
     command: [sh, -c, "exit 0"]

 - name: Failed
   container:
     image: alpine:3.23
     command: [sh, -c, "exit 1"]

 - name: Skipped
   container:
     image: alpine:3.23
     command: [sh, -c, "echo Hello"]
`

func TestSingleDependency(t *testing.T) {
	statusMap := map[string]v1.PodPhase{"Succeeded": v1.PodSucceeded, "Failed": v1.PodFailed}
	var closer context.CancelFunc
	var controller *WorkflowController
	for _, status := range []string{"Succeeded", "Failed", "Skipped"} {
		fmt.Printf("\n\n\nCurrent status %s\n\n\n", status)
		ctx := logging.TestContext(t.Context())
		closer, controller = newController(ctx)
		wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

		// If the status is "skipped" skip the root node.
		var wfString string
		if status == "Skipped" {
			wfString = fmt.Sprintf(dynamicSingleDag, status, `when: "False == True"`, status)
		} else {
			wfString = fmt.Sprintf(dynamicSingleDag, status, "", status)
		}
		wf := wfv1.MustUnmarshalWorkflow(wfString)

		wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
		require.NoError(t, err)
		wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
		require.NoError(t, err)
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		woc.operate(ctx)
		// Mark the status of the pod according to the test
		if _, ok := statusMap[status]; ok {
			makePodsPhase(ctx, woc, statusMap[status])
		} else {
			makePodsPhase(ctx, woc, v1.PodPending)
		}

		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		found := false
		for _, node := range woc.wf.Status.Nodes {
			if strings.Contains(node.Name, "TestSingle") {
				found = true
				assert.Equal(t, wfv1.NodePending, node.Phase)
			}
		}
		assert.True(t, found)
		if closer != nil {
			closer()
		}
	}
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
      - name: sequence-param
        template: print-message
        dependencies: [generate-artifact]
        when: "false"
        arguments:
          artifacts:
          - name: message
            from: "{{tasks.generate-artifact.outputs.artifacts.hello-art}}"
        withSequence:
          count: "5"

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
func TestArtifactResolutionWhenSkippedDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(artifactResolutionWhenSkippedDAG)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestExpandTaskWithParam(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	task := wfv1.DAGTask{
		Name:     "fanout-param",
		Template: "tmpl",
		Arguments: wfv1.Arguments{
			Parameters: []wfv1.Parameter{{
				Name:  "msg",
				Value: wfv1.AnyStringPtr("{{item}}"),
			}},
		},
		WithParam: `[1234, "foo\tbar", true, []]`,
	}

	expanded, err := expandTask(ctx, task, map[string]string{})
	require.NoError(t, err)
	require.Len(t, expanded, 4)

	expectedExpandedTasks := []struct {
		Name      string
		Parameter string
	}{
		{
			Name:      "fanout-param(0:1234)",
			Parameter: "1234",
		},
		{
			Name:      `fanout-param(1:foo\tbar)`,
			Parameter: "foo\tbar",
		},
		{
			Name:      "fanout-param(2:true)",
			Parameter: "true",
		},
		{
			Name:      "fanout-param(3:[])",
			Parameter: "[]",
		},
	}

	for i, expected := range expectedExpandedTasks {
		assert.Equal(t, expected.Name, expanded[i].Name)
		assert.Equal(t, "tmpl", expanded[i].Template)
		assert.Equal(t, expected.Parameter, expanded[i].Arguments.Parameters[0].Value.String())
	}
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
			Depends: "A && (C.Succeeded || C.Failed)",
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
	ctx := logging.TestContext(t.Context())
	d := &dagContext{
		boundaryName: "test",
		tasks:        testTasks,
		wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
		dependencies: make(map[string][]string),
		dependsLogic: make(map[string]string),
		log:          logging.RequireLoggerFromContext(ctx),
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
	execute, proceed, err := d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.False(t, proceed)
	assert.False(t, execute)

	// Task A succeeded
	d.wf.Status.Nodes[d.taskNodeID("A")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}

	// Task B and C should proceed and execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	execute, proceed, err = d.evaluateDependsLogic(ctx, "C")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	// Other tasks should not
	execute, proceed, err = d.evaluateDependsLogic(ctx, "should-execute-1")
	require.NoError(t, err)
	assert.False(t, proceed)
	assert.False(t, execute)

	// Tasks B succeeded, C failed
	d.wf.Status.Nodes[d.taskNodeID("B")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}
	d.wf.Status.Nodes[d.taskNodeID("C")] = wfv1.NodeStatus{Phase: wfv1.NodeFailed}

	// Tasks should-execute-1 and should-execute-2 should proceed and execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "should-execute-1")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	execute, proceed, err = d.evaluateDependsLogic(ctx, "should-execute-2")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
	// Task should-not-execute should proceed, but not execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "should-not-execute")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.False(t, execute)

	// Tasks should-execute-1 and should-execute-2 succeeded, should-not-execute skipped
	d.wf.Status.Nodes[d.taskNodeID("should-execute-1")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}
	d.wf.Status.Nodes[d.taskNodeID("should-execute-2")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}
	d.wf.Status.Nodes[d.taskNodeID("should-not-execute")] = wfv1.NodeStatus{Phase: wfv1.NodeSkipped}

	// Tasks should-execute-3 should proceed and execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "should-execute-3")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
}

func TestEvaluateAnyAllDependsLogic(t *testing.T) {
	testTasks := []wfv1.DAGTask{
		{
			Name: "A",
		},
		{
			Name: "A-1",
		},
		{
			Name: "A-2",
		},
		{
			Name:    "B",
			Depends: "A.AnySucceeded",
		},
		{
			Name: "B-1",
		},
		{
			Name: "B-2",
		},
		{
			Name:    "C",
			Depends: "B.AllFailed",
		},
	}

	ctx := logging.TestContext(t.Context())
	d := &dagContext{
		boundaryName: "test",
		tasks:        testTasks,
		wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
		dependencies: make(map[string][]string),
		dependsLogic: make(map[string]string),
		log:          logging.RequireLoggerFromContext(ctx),
	}

	// Task A is still running, A-1 succeeded but A-2 failed
	d.wf = &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: map[string]wfv1.NodeStatus{
				d.taskNodeID("A"): {
					Phase:    wfv1.NodeRunning,
					Type:     wfv1.NodeTypeTaskGroup,
					Children: []string{d.taskNodeID("A-1"), d.taskNodeID("A-2")},
				},
				d.taskNodeID("A-1"): {Phase: wfv1.NodeRunning},
				d.taskNodeID("A-2"): {Phase: wfv1.NodeRunning},
			},
		},
	}

	// Task B should not proceed as task A is still running
	execute, proceed, err := d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.False(t, proceed)
	assert.False(t, execute)

	// Task A succeeded
	d.wf.Status.Nodes[d.taskNodeID("A")] = wfv1.NodeStatus{
		Phase:    wfv1.NodeSucceeded,
		Type:     wfv1.NodeTypeTaskGroup,
		Children: []string{d.taskNodeID("A-1"), d.taskNodeID("A-2")},
	}

	// Task B should proceed, but not execute as none of the children have succeeded yet
	execute, proceed, err = d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.False(t, execute)

	// Task A-2 succeeded
	d.wf.Status.Nodes[d.taskNodeID("A-2")] = wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}

	// Task B should now proceed and execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)

	// Task B succeeds and B-1 fails
	d.wf.Status.Nodes[d.taskNodeID("B")] = wfv1.NodeStatus{
		Phase:    wfv1.NodeSucceeded,
		Type:     wfv1.NodeTypeTaskGroup,
		Children: []string{d.taskNodeID("B-1"), d.taskNodeID("B-2")},
	}
	d.wf.Status.Nodes[d.taskNodeID("B-1")] = wfv1.NodeStatus{Phase: wfv1.NodeFailed}

	// Task C should proceed, but not execute as not all of B's children have failed yet
	execute, proceed, err = d.evaluateDependsLogic(ctx, "C")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.False(t, execute)

	d.wf.Status.Nodes[d.taskNodeID("B-2")] = wfv1.NodeStatus{Phase: wfv1.NodeFailed}

	// Task C should now proceed and execute as all of B's children have failed
	execute, proceed, err = d.evaluateDependsLogic(ctx, "C")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
}

func TestEvaluateDependsLogicWhenDaemonFailed(t *testing.T) {
	testTasks := []wfv1.DAGTask{
		{
			Name: "A",
		},
		{
			Name:    "B",
			Depends: "A",
		},
	}

	ctx := logging.TestContext(t.Context())
	d := &dagContext{
		boundaryName: "test",
		tasks:        testTasks,
		wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
		dependencies: make(map[string][]string),
		dependsLogic: make(map[string]string),
		log:          logging.RequireLoggerFromContext(ctx),
	}

	// Task A is running
	daemon := true
	d.wf = &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: map[string]wfv1.NodeStatus{
				d.taskNodeID("A"): {Phase: wfv1.NodeRunning, Daemoned: &daemon},
			},
		},
	}

	// Task B should proceed and execute
	execute, proceed, err := d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)

	// Task B running
	d.wf.Status.Nodes[d.taskNodeID("B")] = wfv1.NodeStatus{Phase: wfv1.NodeRunning}

	// Task A failed or error
	d.wf.Status.Nodes[d.taskNodeID("A")] = wfv1.NodeStatus{Phase: wfv1.NodeFailed}

	// Task B should proceed and execute
	execute, proceed, err = d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
}

func TestEvaluateDependsLogicWhenTaskOmitted(t *testing.T) {
	testTasks := []wfv1.DAGTask{
		{
			Name: "A",
		},
		{
			Name:    "B",
			Depends: "A.Omitted",
		},
	}

	ctx := logging.TestContext(t.Context())
	d := &dagContext{
		boundaryName: "test",
		tasks:        testTasks,
		wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
		dependencies: make(map[string][]string),
		dependsLogic: make(map[string]string),
		log:          logging.RequireLoggerFromContext(ctx),
	}

	// Task A is running
	d.wf = &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-wf"},
		Status: wfv1.WorkflowStatus{
			Nodes: map[string]wfv1.NodeStatus{
				d.taskNodeID("A"): {Phase: wfv1.NodeOmitted},
			},
		},
	}

	// Task B should proceed and execute
	execute, proceed, err := d.evaluateDependsLogic(ctx, "B")
	require.NoError(t, err)
	assert.True(t, proceed)
	assert.True(t, execute)
}

func TestAllEvaluateDependsLogic(t *testing.T) {
	statusMap := map[common.TaskResult]wfv1.NodePhase{
		common.TaskResultSucceeded: wfv1.NodeSucceeded,
		common.TaskResultFailed:    wfv1.NodeFailed,
		common.TaskResultSkipped:   wfv1.NodeSkipped,
	}
	for _, status := range []common.TaskResult{common.TaskResultSucceeded, common.TaskResultFailed, common.TaskResultSkipped} {
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

		ctx := logging.TestContext(t.Context())
		d := &dagContext{
			boundaryName: "test",
			tasks:        testTasks,
			wf:           &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{Name: "test-wf"}},
			dependencies: make(map[string][]string),
			dependsLogic: make(map[string]string),
			log:          logging.RequireLoggerFromContext(ctx),
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

		execute, proceed, err := d.evaluateDependsLogic(ctx, "Run")
		require.NoError(t, err)
		assert.True(t, proceed)
		assert.True(t, execute)
		execute, proceed, err = d.evaluateDependsLogic(ctx, "NotRun")
		require.NoError(t, err)
		assert.True(t, proceed)
		assert.False(t, execute)
	}
}

var dagAssessPhaseContinueOnExpandedTaskVariables = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: parameter-aggregation-one-will-fail2-jt776
spec:

  entrypoint: parameter-aggregation-one-will-fail2
  templates:
  -
    dag:
      tasks:
      -
        continueOn:
          failed: true
        name: generate
        template: gen-number-list
      - arguments:
          parameters:
          - name: num
            value: '{{item}}'
        continueOn:
          failed: true
        dependencies:
        - generate
        name: one-will-fail
        template: one-will-fail
        withParam: '{{tasks.generate.outputs.result}}'
      -
        continueOn:
          failed: true
        dependencies:
        - one-will-fail
        name: whalesay
        template: whalesay
    inputs: {}
    metadata: {}
    name: parameter-aggregation-one-will-fail2
    outputs: {}
  -
    container:
      args:
      - |
        if [ $(({{inputs.parameters.num}})) == 1 ]; then
          exit 1;
        else
          echo {{inputs.parameters.num}}
        fi
      command:
      - sh
      - -xc
      image: alpine:3.23
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: num
    metadata: {}
    name: one-will-fail
    outputs: {}
  -
    container:
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
  -
    inputs: {}
    metadata: {}
    name: gen-number-list
    outputs: {}
    script:
      command:
      - python
      image: python:alpine3.23
      name: ""
      resources: {}
      source: |
        import json
        import sys
        json.dump([i for i in range(0, 2)], sys.stdout)
status:
  nodes:
    parameter-aggregation-one-will-fail2-jt776:
      children:
      - parameter-aggregation-one-will-fail2-jt776-1457662774
      displayName: parameter-aggregation-one-will-fail2-jt776
      id: parameter-aggregation-one-will-fail2-jt776
      name: parameter-aggregation-one-will-fail2-jt776
      outboundNodes:
      - parameter-aggregation-one-will-fail2-jt776-3936077093
      phase: Running
      startedAt: "2020-04-20T16:39:00Z"
      templateName: parameter-aggregation-one-will-fail2
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: DAG
    parameter-aggregation-one-will-fail2-jt776-6921149:
      boundaryID: parameter-aggregation-one-will-fail2-jt776
      children:
      - parameter-aggregation-one-will-fail2-jt776-1842114754
      - parameter-aggregation-one-will-fail2-jt776-4113411742
      displayName: one-will-fail
      finishedAt: "2020-04-20T16:39:09Z"
      id: parameter-aggregation-one-will-fail2-jt776-6921149
      name: parameter-aggregation-one-will-fail2-jt776.one-will-fail
      phase: Failed
      startedAt: "2020-04-20T16:39:03Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: TaskGroup
    parameter-aggregation-one-will-fail2-jt776-1457662774:
      boundaryID: parameter-aggregation-one-will-fail2-jt776
      children:
      - parameter-aggregation-one-will-fail2-jt776-6921149
      displayName: generate
      finishedAt: "2020-04-20T16:39:02Z"
      id: parameter-aggregation-one-will-fail2-jt776-1457662774
      name: parameter-aggregation-one-will-fail2-jt776.generate
      outputs:
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
            key: parameter-aggregation-one-will-fail2-jt776/parameter-aggregation-one-will-fail2-jt776-1457662774/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
        result: '[0, 1]'
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2020-04-20T16:39:00Z"
      templateName: gen-number-list
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: Pod
    parameter-aggregation-one-will-fail2-jt776-1842114754:
      boundaryID: parameter-aggregation-one-will-fail2-jt776
      children:
      - parameter-aggregation-one-will-fail2-jt776-3936077093
      displayName: one-will-fail(0:0)
      finishedAt: "2020-04-20T16:39:06Z"
      id: parameter-aggregation-one-will-fail2-jt776-1842114754
      inputs:
        parameters:
        - name: num
          value: "0"
      name: parameter-aggregation-one-will-fail2-jt776.one-will-fail(0:0)
      outputs:
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
            key: parameter-aggregation-one-will-fail2-jt776/parameter-aggregation-one-will-fail2-jt776-1842114754/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2020-04-20T16:39:03Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: Pod
    parameter-aggregation-one-will-fail2-jt776-3936077093:
      boundaryID: parameter-aggregation-one-will-fail2-jt776
      displayName: whalesay
      finishedAt: "2020-04-20T16:39:14Z"
      id: parameter-aggregation-one-will-fail2-jt776-3936077093
      name: parameter-aggregation-one-will-fail2-jt776.whalesay
      outputs:
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
            key: parameter-aggregation-one-will-fail2-jt776/parameter-aggregation-one-will-fail2-jt776-3936077093/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2020-04-20T16:39:10Z"
      templateName: whalesay
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: Pod
    parameter-aggregation-one-will-fail2-jt776-4113411742:
      boundaryID: parameter-aggregation-one-will-fail2-jt776
      children:
      - parameter-aggregation-one-will-fail2-jt776-3936077093
      displayName: one-will-fail(1:1)
      finishedAt: "2020-04-20T16:39:07Z"
      id: parameter-aggregation-one-will-fail2-jt776-4113411742
      inputs:
        parameters:
        - name: num
          value: "1"
      message: failed with exit code 1
      name: parameter-aggregation-one-will-fail2-jt776.one-will-fail(1:1)
      outputs:
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
            key: parameter-aggregation-one-will-fail2-jt776/parameter-aggregation-one-will-fail2-jt776-4113411742/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2020-04-20T16:39:03Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail2-jt776
      type: Pod
  phase: Running
  resourcesDuration:
    cpu: 10
    memory: 0
  startedAt: "2020-04-20T16:39:00Z"
`

// Tests whether assessPhase marks a DAG as successful when it contains failed tasks with continueOn failed
func TestDagAssessPhaseContinueOnExpandedTaskVariables(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagAssessPhaseContinueOnExpandedTaskVariables)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

var dagAssessPhaseContinueOnExpandedTask = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: parameter-aggregation-one-will-fail-69x7k
spec:

  entrypoint: parameter-aggregation-one-will-fail
  templates:
  -
    dag:
      tasks:
      - arguments:
          parameters:
          - name: num
            value: '{{item}}'
        continueOn:
          failed: true
        name: one-will-fail
        template: one-will-fail
        withItems:
        - 1
        - 2
      -
        continueOn:
          failed: true
        dependencies:
        - one-will-fail
        name: whalesay
        template: whalesay
    inputs: {}
    metadata: {}
    name: parameter-aggregation-one-will-fail
    outputs: {}
  -
    container:
      args:
      - |
        if [ $(({{inputs.parameters.num}})) == 1 ]; then
          exit 1;
        else
          echo {{inputs.parameters.num}}
        fi
      command:
      - sh
      - -xc
      image: alpine:3.23
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: num
    metadata: {}
    name: one-will-fail
    outputs: {}
  -
    container:
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
status:
  nodes:
    parameter-aggregation-one-will-fail-69x7k:
      children:
      - parameter-aggregation-one-will-fail-69x7k-4292161196
      displayName: parameter-aggregation-one-will-fail-69x7k
      id: parameter-aggregation-one-will-fail-69x7k
      name: parameter-aggregation-one-will-fail-69x7k
      outboundNodes:
      - parameter-aggregation-one-will-fail-69x7k-3555414042
      phase: Running
      startedAt: "2020-04-20T16:47:22Z"
      templateName: parameter-aggregation-one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail-69x7k
      type: DAG
    parameter-aggregation-one-will-fail-69x7k-1324058456:
      boundaryID: parameter-aggregation-one-will-fail-69x7k
      children:
      - parameter-aggregation-one-will-fail-69x7k-3555414042
      displayName: one-will-fail(0:1)
      finishedAt: "2020-04-20T16:47:26Z"
      id: parameter-aggregation-one-will-fail-69x7k-1324058456
      inputs:
        parameters:
        - name: num
          value: "1"
      message: failed with exit code 1
      name: parameter-aggregation-one-will-fail-69x7k.one-will-fail(0:1)
      outputs:
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
            key: parameter-aggregation-one-will-fail-69x7k/parameter-aggregation-one-will-fail-69x7k-1324058456/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2020-04-20T16:47:22Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail-69x7k
      type: Pod
    parameter-aggregation-one-will-fail-69x7k-3086527730:
      boundaryID: parameter-aggregation-one-will-fail-69x7k
      children:
      - parameter-aggregation-one-will-fail-69x7k-3555414042
      displayName: one-will-fail(1:2)
      finishedAt: "2020-04-20T16:47:28Z"
      id: parameter-aggregation-one-will-fail-69x7k-3086527730
      inputs:
        parameters:
        - name: num
          value: "2"
      name: parameter-aggregation-one-will-fail-69x7k.one-will-fail(1:2)
      outputs:
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
            key: parameter-aggregation-one-will-fail-69x7k/parameter-aggregation-one-will-fail-69x7k-3086527730/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 4
        memory: 0
      startedAt: "2020-04-20T16:47:22Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail-69x7k
      type: Pod
    parameter-aggregation-one-will-fail-69x7k-3555414042:
      boundaryID: parameter-aggregation-one-will-fail-69x7k
      displayName: whalesay
      finishedAt: "2020-04-20T16:47:33Z"
      id: parameter-aggregation-one-will-fail-69x7k-3555414042
      name: parameter-aggregation-one-will-fail-69x7k.whalesay
      outputs:
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
            key: parameter-aggregation-one-will-fail-69x7k/parameter-aggregation-one-will-fail-69x7k-3555414042/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 3
        memory: 0
      startedAt: "2020-04-20T16:47:30Z"
      templateName: whalesay
      templateScope: local/parameter-aggregation-one-will-fail-69x7k
      type: Pod
    parameter-aggregation-one-will-fail-69x7k-4292161196:
      boundaryID: parameter-aggregation-one-will-fail-69x7k
      children:
      - parameter-aggregation-one-will-fail-69x7k-1324058456
      - parameter-aggregation-one-will-fail-69x7k-3086527730
      displayName: one-will-fail
      finishedAt: "2020-04-20T16:47:29Z"
      id: parameter-aggregation-one-will-fail-69x7k-4292161196
      name: parameter-aggregation-one-will-fail-69x7k.one-will-fail
      phase: Failed
      startedAt: "2020-04-20T16:47:22Z"
      templateName: one-will-fail
      templateScope: local/parameter-aggregation-one-will-fail-69x7k
      type: TaskGroup
  phase: Running
  resourcesDuration:
    cpu: 9
    memory: 0
  startedAt: "2020-04-20T16:47:22Z"
`

// Tests whether assessPhase marks a DAG as successful when it contains failed tasks with continueOn failed
func TestDagAssessPhaseContinueOnExpandedTask(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagAssessPhaseContinueOnExpandedTask)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

var dagWithParamAndGlobalParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-with-param-and-global-param-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: workspace
      value: /argo_workspace/{{workflow.uid}}
  templates:
  - name: main
    dag:
      tasks:
      - name: use-with-param
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

func TestDAGWithParamAndGlobalParam(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(dagWithParamAndGlobalParam)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var terminatingDAGWithRetryStrategyNodes = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-diamond-xfww2
spec:

  entrypoint: diamond
  shutdown: Terminate
  templates:
  -
    dag:
      tasks:
      -
        name: A
        template: echo
      -
        dependencies:
        - A
        name: B
        template: echo
      -
        dependencies:
        - A
        name: C
        template: echo
      -
        dependencies:
        - B
        - C
        name: D
        template: echo
    inputs: {}
    metadata: {}
    name: diamond
    outputs: {}
  -
    container:
      args:
      - sleep 10
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: echo
    outputs: {}
    retryStrategy:
      limit: 4
status:
  finishedAt: null
  nodes:
    dag-diamond-xfww2:
      children:
      - dag-diamond-xfww2-1488588956
      displayName: dag-diamond-xfww2
      finishedAt: null
      id: dag-diamond-xfww2
      name: dag-diamond-xfww2
      phase: Running
      startedAt: "2020-05-06T16:15:38Z"
      templateName: diamond
      templateScope: local/dag-diamond-xfww2
      type: DAG
    dag-diamond-xfww2-990947287:
      boundaryID: dag-diamond-xfww2
      children:
      - dag-diamond-xfww2-1522144194
      - dag-diamond-xfww2-1538921813
      displayName: A(0)
      finishedAt: "2020-05-06T16:15:50Z"
      hostNodeName: minikube
      id: dag-diamond-xfww2-990947287
      name: dag-diamond-xfww2.A(0)
      outputs:
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
            key: dag-diamond-xfww2/dag-diamond-xfww2-990947287/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      phase: Succeeded
      resourcesDuration:
        cpu: 21
        memory: 0
      startedAt: "2020-05-06T16:15:38Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Pod
    dag-diamond-xfww2-1488588956:
      boundaryID: dag-diamond-xfww2
      children:
      - dag-diamond-xfww2-990947287
      displayName: A
      finishedAt: "2020-05-06T16:15:51Z"
      id: dag-diamond-xfww2-1488588956
      name: dag-diamond-xfww2.A
      outputs:
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
            key: dag-diamond-xfww2/dag-diamond-xfww2-990947287/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      phase: Succeeded
      startedAt: "2020-05-06T16:15:38Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Retry
    dag-diamond-xfww2-1522144194:
      boundaryID: dag-diamond-xfww2
      children:
      - dag-diamond-xfww2-2043927737
      displayName: C
      finishedAt: "2020-05-06T16:15:59Z"
      id: dag-diamond-xfww2-1522144194
      message: Stopped with strategy 'Terminate'
      name: dag-diamond-xfww2.C
      phase: Failed
      startedAt: "2020-05-06T16:15:51Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Retry
    dag-diamond-xfww2-1538921813:
      boundaryID: dag-diamond-xfww2
      children:
      - dag-diamond-xfww2-3629114292
      displayName: B
      finishedAt: "2020-05-06T16:15:59Z"
      id: dag-diamond-xfww2-1538921813
      message: Stopped with strategy 'Terminate'
      name: dag-diamond-xfww2.B
      phase: Failed
      startedAt: "2020-05-06T16:15:52Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Retry
    dag-diamond-xfww2-2043927737:
      boundaryID: dag-diamond-xfww2
      displayName: C(0)
      finishedAt: "2020-05-06T16:15:58Z"
      hostNodeName: minikube
      id: dag-diamond-xfww2-2043927737
      message: terminated
      name: dag-diamond-xfww2.C(0)
      outputs:
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
            key: dag-diamond-xfww2/dag-diamond-xfww2-2043927737/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      phase: Failed
      resourcesDuration:
        cpu: 11
        memory: 0
      startedAt: "2020-05-06T16:15:51Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Pod
    dag-diamond-xfww2-3629114292:
      boundaryID: dag-diamond-xfww2
      displayName: B(0)
      finishedAt: "2020-05-06T16:15:58Z"
      hostNodeName: minikube
      id: dag-diamond-xfww2-3629114292
      message: terminated
      name: dag-diamond-xfww2.B(0)
      outputs:
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
            key: dag-diamond-xfww2/dag-diamond-xfww2-3629114292/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      phase: Failed
      resourcesDuration:
        cpu: 9
        memory: 0
      startedAt: "2020-05-06T16:15:52Z"
      templateName: echo
      templateScope: local/dag-diamond-xfww2
      type: Pod
  phase: Running
  startedAt: "2020-05-06T16:15:38Z"
`

// This tests that a DAG with retry strategy in its tasks fails successfully when terminated
func TestTerminatingDAGWithRetryStrategyNodes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(terminatingDAGWithRetryStrategyNodes)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

var terminateDAGWithMaxDurationLimitExpiredAndMoreAttempts = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-diamond-dj7q5
spec:

  entrypoint: diamond
  templates:
  -
    dag:
      tasks:
      -
        name: A
        template: echo
      -
        dependencies:
        - A
        name: B
        template: echo
    inputs: {}
    metadata: {}
    name: diamond
    outputs: {}
  -
    container:
      args:
      - exit 1
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: echo
    outputs: {}
    retryStrategy:
      backoff:
        duration: "1"
        maxDuration: "5"
      limit: 10
status:
  nodes:
    dag-diamond-dj7q5:
      children:
      - dag-diamond-dj7q5-2391658435
      displayName: dag-diamond-dj7q5
      finishedAt: "2020-05-27T15:42:01Z"
      id: dag-diamond-dj7q5
      name: dag-diamond-dj7q5
      phase: Running
      startedAt: "2020-05-27T15:41:54Z"
      templateName: diamond
      templateScope: local/dag-diamond-dj7q5
      type: DAG
    dag-diamond-dj7q5-2241203531:
      boundaryID: dag-diamond-dj7q5
      displayName: A(1)
      finishedAt: "2020-05-27T15:41:59Z"
      hostNodeName: minikube
      id: dag-diamond-dj7q5-2241203531
      message: failed with exit code 1
      name: dag-diamond-dj7q5.A(1)
      outputs:
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
            key: dag-diamond-dj7q5/dag-diamond-dj7q5-2241203531/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2020-05-27T15:41:57Z"
      templateName: echo
      templateScope: local/dag-diamond-dj7q5
      type: Pod
    dag-diamond-dj7q5-2391658435:
      boundaryID: dag-diamond-dj7q5
      children:
      - dag-diamond-dj7q5-2845344910
      - dag-diamond-dj7q5-2241203531
      displayName: A
      finishedAt: "2020-05-27T15:42:01Z"
      id: dag-diamond-dj7q5-2391658435
      name: dag-diamond-dj7q5.A
      phase: Running
      startedAt: "2020-05-27T15:41:54Z"
      templateName: echo
      templateScope: local/dag-diamond-dj7q5
      type: Retry
    dag-diamond-dj7q5-2845344910:
      boundaryID: dag-diamond-dj7q5
      displayName: A(0)
      finishedAt: "2020-05-27T15:41:56Z"
      hostNodeName: minikube
      id: dag-diamond-dj7q5-2845344910
      message: failed with exit code 1
      name: dag-diamond-dj7q5.A(0)
      outputs:
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
            key: dag-diamond-dj7q5/dag-diamond-dj7q5-2845344910/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2020-05-27T15:41:54Z"
      templateName: echo
      templateScope: local/dag-diamond-dj7q5
      type: Pod
      nodeFlag:
        retried: true
  phase: Running
  resourcesDuration:
    cpu: 2
    memory: 0
  startedAt: "2020-05-27T15:41:54Z"
`

// This tests that a DAG with retry strategy in its tasks fails successfully when terminated
func TestTerminateDAGWithMaxDurationLimitExpiredAndMoreAttempts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(terminateDAGWithMaxDurationLimitExpiredAndMoreAttempts)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	retryNode, err := woc.wf.GetNodeByName("dag-diamond-dj7q5.A")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeFailed, retryNode.Phase)
	assert.Contains(t, retryNode.Message, "Max duration limit exceeded")

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// This is the crucial part of the test
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

var testRetryStrategyNodes = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: wf-retry-pol
spec:

  entrypoint: run-steps
  onExit: onExit
  templates:
  -
    inputs: {}
    metadata: {}
    name: run-steps
    outputs: {}
    steps:
    - -
        name: run-dag
        template: run-dag
    - -
        name: manual-onExit
        template: onExit
  -
    dag:
      tasks:
      -
        name: A
        template: fail
      -
        dependencies:
        - A
        name: B
        template: onExit
    inputs: {}
    metadata: {}
    name: run-dag
    outputs: {}
  -
    container:
      args:
      - exit 2
      command:
      - sh
      - -c
      image: alpine
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: fail
    outputs: {}
    retryStrategy:
      limit: 100
      retryPolicy: OnError
  -
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: onExit
    outputs: {}
status:
  nodes:
    wf-retry-pol:
      children:
      - wf-retry-pol-3488382045
      displayName: wf-retry-pol
      finishedAt: "2020-05-27T16:03:00Z"
      id: wf-retry-pol
      name: wf-retry-pol
      outboundNodes:
      - wf-retry-pol-2278798366
      phase: Running
      startedAt: "2020-05-27T16:02:49Z"
      templateName: run-steps
      templateScope: local/wf-retry-pol
      type: Steps
    wf-retry-pol-2616013767:
      boundaryID: wf-retry-pol
      children:
      - wf-retry-pol-3151556158
      displayName: run-dag
      id: wf-retry-pol-2616013767
      name: wf-retry-pol[0].run-dag
      outboundNodes:
      - wf-retry-pol-3134778539
      phase: Running
      startedAt: "2020-05-27T16:02:49Z"
      templateName: run-dag
      templateScope: local/wf-retry-pol
      type: DAG
    wf-retry-pol-3148069997:
      boundaryID: wf-retry-pol-2616013767
      children:
      - wf-retry-pol-3134778539
      displayName: A(0)
      finishedAt: "2020-05-27T16:02:53Z"
      hostNodeName: minikube
      id: wf-retry-pol-3148069997
      message: failed with exit code 2
      name: wf-retry-pol[0].run-dag.A(0)
      outputs:
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
            key: wf-retry-pol/wf-retry-pol-3148069997/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "2"
      phase: Failed
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2020-05-27T16:02:49Z"
      templateName: fail
      templateScope: local/wf-retry-pol
      type: Pod
    wf-retry-pol-3151556158:
      boundaryID: wf-retry-pol-2616013767
      children:
      - wf-retry-pol-3148069997
      displayName: A
      id: wf-retry-pol-3151556158
      message: failed with exit code 2
      name: wf-retry-pol[0].run-dag.A
      phase: Running
      startedAt: "2020-05-27T16:02:49Z"
      templateName: fail
      templateScope: local/wf-retry-pol
      type: Retry
    wf-retry-pol-3488382045:
      boundaryID: wf-retry-pol
      children:
      - wf-retry-pol-2616013767
      displayName: '[0]'
      id: wf-retry-pol-3488382045
      name: wf-retry-pol[0]
      phase: Running
      startedAt: "2020-05-27T16:02:49Z"
      templateName: run-steps
      templateScope: local/wf-retry-pol
      type: StepGroup
  phase: Running
  resourcesDuration:
    cpu: 6
    memory: 2
  startedAt: "2020-05-27T16:02:49Z"
`

func TestRetryStrategyNodes(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testRetryStrategyNodes)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	retryNode, err := woc.wf.GetNodeByName("wf-retry-pol")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeFailed, retryNode.Phase)

	onExitNode, err := woc.wf.GetNodeByName("wf-retry-pol.onExit")
	require.NoError(t, err)
	assert.NotNil(t, onExitNode)
	assert.True(t, onExitNode.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, onExitNode.Phase)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var testOnExitNodeDAGPhase = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-diamond-88trp
spec:

  entrypoint: diamond
  templates:
  -
    dag:
      failFast: false
      tasks:
      -
        name: A
        template: echo
      -
        dependencies:
        - A
        name: B
        onExit: echo
        template: echo
    inputs: {}
    metadata: {}
    name: diamond
    outputs: {}
  -
    container:
      args:
      - exit 0
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: echo
    outputs: {}
  -
    container:
      args:
      - exit 1
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: fail
    outputs: {}
status:
  nodes:
    dag-diamond-88trp:
      children:
      - dag-diamond-88trp-2052796420
      displayName: dag-diamond-88trp
      id: dag-diamond-88trp
      name: dag-diamond-88trp
      outboundNodes:
      - dag-diamond-88trp-2103129277
      phase: Running
      startedAt: "2020-05-29T18:11:55Z"
      templateName: diamond
      templateScope: local/dag-diamond-88trp
      type: DAG
    dag-diamond-88trp-2052796420:
      boundaryID: dag-diamond-88trp
      children:
      - dag-diamond-88trp-2103129277
      displayName: A
      finishedAt: "2020-05-29T18:11:58Z"
      hostNodeName: minikube
      id: dag-diamond-88trp-2052796420
      name: dag-diamond-88trp.A
      outputs:
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
            key: dag-diamond-88trp/dag-diamond-88trp-2052796420/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2020-05-29T18:11:55Z"
      templateName: echo
      templateScope: local/dag-diamond-88trp
      type: Pod
    dag-diamond-88trp-2103129277:
      boundaryID: dag-diamond-88trp
      displayName: B
      finishedAt: "2020-05-29T18:12:01Z"
      hostNodeName: minikube
      id: dag-diamond-88trp-2103129277
      name: dag-diamond-88trp.B
      outputs:
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
            key: dag-diamond-88trp/dag-diamond-88trp-2103129277/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2020-05-29T18:11:59Z"
      templateName: echo
      templateScope: local/dag-diamond-88trp
      type: Pod
  phase: Running
  resourcesDuration:
    cpu: 5
    memory: 2
  startedAt: "2020-05-29T18:11:55Z"
`

func TestOnExitDAGPhase(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOnExitNodeDAGPhase)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	retryNode, err := woc.wf.GetNodeByName("dag-diamond-88trp")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodeRunning, retryNode.Phase)

	retryNode, err = woc.wf.GetNodeByName("dag-diamond-88trp.B.onExit")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.True(t, retryNode.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, retryNode.Phase)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var testOnExitNonLeaf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: exit-handler-bug-example
spec:

  entrypoint: dag
  templates:
  -
    dag:
      tasks:
      -
        name: step-2
        onExit: on-exit
        template: step-template
      -
        dependencies:
        - step-2
        name: step-3
        onExit: on-exit
        template: step-template
    inputs: {}
    metadata: {}
    name: dag
    outputs: {}
  -
    container:
      args:
      - echo exit-handler-step-{{pod.name}}
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: on-exit
    outputs: {}
  -
    container:
      args:
      - echo step {{pod.name}}
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: step-template
    outputs: {}
status:
  nodes:
    exit-handler-bug-example:
      children:
      - exit-handler-bug-example-3054913383
      displayName: exit-handler-bug-example
      finishedAt: "2020-07-07T16:15:54Z"
      id: exit-handler-bug-example
      name: exit-handler-bug-example
      outboundNodes:
      - exit-handler-bug-example-3038135764
      phase: Running
      startedAt: "2020-07-07T16:15:33Z"
      templateName: dag
      templateScope: local/exit-handler-bug-example
      type: DAG
    exit-handler-bug-example-3054913383:
      boundaryID: exit-handler-bug-example
      displayName: step-2
      finishedAt: "2020-07-07T16:15:37Z"
      hostNodeName: minikube
      id: exit-handler-bug-example-3054913383
      name: exit-handler-bug-example.step-2
      outputs:
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
            key: exit-handler-bug-example/exit-handler-bug-example-3054913383/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 4
        memory: 2
      startedAt: "2020-07-07T16:15:33Z"
      templateName: step-template
      templateScope: local/exit-handler-bug-example
      type: Pod
  phase: Running
  startedAt: "2020-07-07T16:15:33Z"
`

func TestOnExitNonLeaf(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testOnExitNonLeaf)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	retryNode, err := woc.wf.GetNodeByName("exit-handler-bug-example.step-2.onExit")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.True(t, retryNode.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, retryNode.Phase)

	_, err = woc.wf.GetNodeByName("exit-handler-bug-example.step-3")
	require.Error(t, err)

	retryNode.Phase = wfv1.NodeSucceeded
	woc.wf.Status.Nodes[retryNode.ID] = *retryNode
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	retryNode, err = woc.wf.GetNodeByName("exit-handler-bug-example.step-3")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	assert.Equal(t, wfv1.NodePending, retryNode.Phase)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var testDagOptionalInputArtifacts = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-optional-inputartifacts
spec:
  entrypoint: test
  templates:
  - name: condition
    outputs:
      artifacts:
      - {name: A-out, from: '{{tasks.A.outputs.artifacts.A-out}}'}
    dag:
      tasks:
      - {name: A, template: A}
  - name: A
    container:
      args: ['mkdir -p /tmp/outputs/A && echo "exist" > /tmp/outputs/A/data']
      command: [sh, -c]
      image: alpine:3.23
    outputs:
      artifacts:
      - {name: A-out, path: /tmp/outputs/A/data}
  - name: B
    container:
      args: ['[ -f /tmp/outputs/condition/data ] && cat /tmp/outputs/condition/data || echo not exist']
      command: [sh, -c]
      image: alpine:3.23
    inputs:
      artifacts:
      - {name: B-in, optional: true,  path: /tmp/outputs/condition/data}
  - name: test
    dag:
      tasks:
      - name: condition
        template: condition
        when: 'false'
      - name: B
        template: B
        dependencies: [condition]
        arguments:
          artifacts:
          - {name: B-in, optional: true, from: '{{tasks.condition.outputs.artifacts.A-out}}'}
  arguments:
    parameters: []
status:
  conditions:
  - status: "True"
    type: Completed
  finishedAt: "2020-07-21T01:56:24Z"
  nodes:
    dag-optional-inputartifacts:
      children:
      - dag-optional-inputartifacts-3418089753
      displayName: dag-optional-inputartifacts
      finishedAt: "2020-07-21T01:56:24Z"
      id: dag-optional-inputartifacts
      name: dag-optional-inputartifacts
      outboundNodes:
      - dag-optional-inputartifacts-1920355018
      phase: Running
      startedAt: "2020-07-21T01:56:18Z"
      templateName: test
      templateScope: local/dag-optional-inputartifacts
      type: DAG
    dag-optional-inputartifacts-3418089753:
      boundaryID: dag-optional-inputartifacts
      children:
      - dag-optional-inputartifacts-1920355018
      displayName: condition
      finishedAt: "2020-07-21T01:56:18Z"
      id: dag-optional-inputartifacts-3418089753
      message: when 'false' evaluated false
      name: dag-optional-inputartifacts.condition
      phase: Skipped
      startedAt: "2020-07-21T01:56:18Z"
      templateName: condition
      templateScope: local/dag-optional-inputartifacts
      type: Skipped
  phase: Running
  resourcesDuration:
    cpu: 1
    memory: 0
  startedAt: "2020-07-21T01:56:18Z"
`

func TestDagOptionalInputArtifacts(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(testDagOptionalInputArtifacts)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	optionalInputArtifactsNode, err := woc.wf.GetNodeByName("dag-optional-inputartifacts.B")
	require.NoError(t, err)
	assert.NotNil(t, optionalInputArtifactsNode)
	assert.Equal(t, wfv1.NodePending, optionalInputArtifactsNode.Phase)
}

var testDagTargetTaskOnExit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-primay-branch-6bnnl
spec:

  entrypoint: statis
  templates:
  -
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: a
    outputs: {}
  -
    container:
      args:
      - exit!
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: exit
    outputs: {}
  -
    inputs: {}
    metadata: {}
    name: steps
    outputs: {}
    steps:
    - -
        name: step-a
        template: a
  -
    dag:
      tasks:
      -
        name: A
        onExit: exit
        template: steps
    inputs: {}
    metadata: {}
    name: statis
    outputs: {}
status:
  nodes:
    dag-primay-branch-6bnnl:
      children:
      - dag-primay-branch-6bnnl-1650817843
      displayName: dag-primay-branch-6bnnl
      finishedAt: "2020-08-10T14:30:19Z"
      id: dag-primay-branch-6bnnl
      name: dag-primay-branch-6bnnl
      outboundNodes:
      - dag-primay-branch-6bnnl-1181733215
      phase: Running
      startedAt: "2020-08-10T14:30:08Z"
      templateName: statis
      templateScope: local/dag-primay-branch-6bnnl
      type: DAG
    dag-primay-branch-6bnnl-1181733215:
      boundaryID: dag-primay-branch-6bnnl-1650817843
      displayName: step-a
      finishedAt: "2020-08-10T14:30:11Z"
      hostNodeName: minikube
      id: dag-primay-branch-6bnnl-1181733215
      name: dag-primay-branch-6bnnl.A[0].step-a
      outputs:
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
            key: dag-primay-branch-6bnnl/dag-primay-branch-6bnnl-1181733215/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 1
      startedAt: "2020-08-10T14:30:08Z"
      templateName: a
      templateScope: local/dag-primay-branch-6bnnl
      type: Pod
    dag-primay-branch-6bnnl-1650817843:
      boundaryID: dag-primay-branch-6bnnl
      children:
      - dag-primay-branch-6bnnl-3841864351
      - dag-primay-branch-6bnnl-1342580575
      displayName: A
      finishedAt: "2020-08-10T14:30:13Z"
      id: dag-primay-branch-6bnnl-1650817843
      name: dag-primay-branch-6bnnl.A
      outboundNodes:
      - dag-primay-branch-6bnnl-1181733215
      phase: Running
      startedAt: "2020-08-10T14:30:08Z"
      templateName: steps
      templateScope: local/dag-primay-branch-6bnnl
      type: Steps
    dag-primay-branch-6bnnl-3841864351:
      boundaryID: dag-primay-branch-6bnnl-1650817843
      children:
      - dag-primay-branch-6bnnl-1181733215
      displayName: '[0]'
      finishedAt: "2020-08-10T14:30:13Z"
      id: dag-primay-branch-6bnnl-3841864351
      name: dag-primay-branch-6bnnl.A[0]
      phase: Running
      startedAt: "2020-08-10T14:30:08Z"
      templateName: steps
      templateScope: local/dag-primay-branch-6bnnl
      type: StepGroup
  phase: Running
  resourcesDuration:
    cpu: 5
    memory: 2
  startedAt: "2020-08-10T14:30:08Z"
`

func TestDagTargetTaskOnExit(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testDagTargetTaskOnExit)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	onExitNode, err := woc.wf.GetNodeByName("dag-primay-branch-6bnnl.A.onExit")
	require.NoError(t, err)
	assert.NotNil(t, onExitNode)
	assert.True(t, onExitNode.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, onExitNode.Phase)
}

var testEmptyWithParamDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-hang-pcwmr
spec:

  entrypoint: dag
  templates:
  -
    dag:
      tasks:
      -
        name: scheduler
        template: job-scheduler
      - arguments:
          parameters:
          - name: job_name
            value: '{{item.job_name}}'
        dependencies:
        - scheduler
        name: children
        template: whalesay
        withParam: '{{tasks.scheduler.outputs.parameters.scheduled-jobs}}'
      -
        dependencies:
        - children
        name: postprocess
        template: whalesay
    inputs: {}
    metadata: {}
    name: dag
    outputs: {}
  -
    container:
      args:
      - echo Decided not to schedule any jobs
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: job-scheduler
    outputs:
      parameters:
      - name: scheduled-jobs
        value: '[]'
  -
    container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: whalesay
    outputs: {}
status:
  finishedAt: null
  nodes:
    dag-hang-pcwmr:
      children:
      - dag-hang-pcwmr-1789179473
      displayName: dag-hang-pcwmr
      finishedAt: null
      id: dag-hang-pcwmr
      name: dag-hang-pcwmr
      phase: Running
      startedAt: "2020-08-11T18:19:28Z"
      templateName: dag
      templateScope: local/dag-hang-pcwmr
      type: DAG
    dag-hang-pcwmr-1415348083:
      boundaryID: dag-hang-pcwmr
      displayName: postprocess
      finishedAt: "2020-08-11T18:19:40Z"
      hostNodeName: dech117
      id: dag-hang-pcwmr-1415348083
      name: dag-hang-pcwmr.postprocess
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 3
        memory: 3
      startedAt: "2020-08-11T18:19:36Z"
      templateName: whalesay
      templateScope: local/dag-hang-pcwmr
      type: Pod
    dag-hang-pcwmr-1738428083:
      boundaryID: dag-hang-pcwmr
      children:
      - dag-hang-pcwmr-1415348083
      displayName: children
      finishedAt: "2020-08-11T18:19:35Z"
      id: dag-hang-pcwmr-1738428083
      name: dag-hang-pcwmr.children
      phase: Succeeded
      startedAt: "2020-08-11T18:19:35Z"
      templateName: whalesay
      templateScope: local/dag-hang-pcwmr
      type: TaskGroup
    dag-hang-pcwmr-1789179473:
      boundaryID: dag-hang-pcwmr
      children:
      - dag-hang-pcwmr-1738428083
      displayName: scheduler
      finishedAt: "2020-08-11T18:19:33Z"
      hostNodeName: dech113
      id: dag-hang-pcwmr-1789179473
      name: dag-hang-pcwmr.scheduler
      outputs:
        exitCode: "0"
        parameters:
        - name: scheduled-jobs
          value: '[]'
      phase: Succeeded
      resourcesDuration:
        cpu: 4
        memory: 4
      startedAt: "2020-08-11T18:19:28Z"
      templateName: job-scheduler
      templateScope: local/dag-hang-pcwmr
      type: Pod
  phase: Running
  startedAt: "2020-08-11T18:19:28Z"
`

func TestEmptyWithParamDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testEmptyWithParamDAG)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

var testFailsWithParamDAG = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: reproduce-bug-9tpfr
spec:

  entrypoint: start
  serviceAccountName: argo-workflow
  templates:
  -
    dag:
      tasks:
      -
        name: gen-tasks
        template: gen-tasks
      - arguments:
          parameters:
          - name: chunk
            value: '{{item}}'
        dependencies:
        - gen-tasks
        name: process-tasks
        template: process-tasks
        withParam: '{{tasks.gen-tasks.outputs.result}}'
      -
        dependencies:
        - process-tasks
        name: finish
        template: finish
    inputs: {}
    metadata: {}
    name: start
    outputs: {}
  - activeDeadlineSeconds: 300

    inputs: {}
    metadata: {}
    name: gen-tasks
    outputs: {}
    retryStrategy:
      backoff:
        duration: 15s
        factor: 2
      limit: 5
      retryPolicy: Always
    script:
      command:
      - bash
      image: python:3
      name: ""
      resources:
        requests:
          cpu: 250m
      source: |
        set -e
        python3 -c 'import os, json; print(json.dumps([str(i) for i in range(10)]))'
  - activeDeadlineSeconds: 1800

    inputs:
      parameters:
      - name: chunk
    metadata: {}
    name: process-tasks
    outputs: {}
    retryStrategy:
      backoff:
        duration: 15s
        factor: 2
      limit: 2
      retryPolicy: Always
    script:
      command:
      - bash
      image: python:3
      name: ""
      resources:
        requests:
          cpu: 100m
      source: |
        set -e
        chunk="{{inputs.parameters.chunk}}"
        if [[ $chunk == "3" ]]; then
          echo "failed"
          exit 1
        fi
        echo "process $chunk"
  - activeDeadlineSeconds: 300

    inputs: {}
    metadata: {}
    name: finish
    outputs: {}
    script:
      command:
      - sh
      image: busybox
      name: ""
      resources:
        requests:
          cpu: 100m
      source: |
        echo fin
status:
  nodes:
    reproduce-bug-9tpfr:
      children:
      - reproduce-bug-9tpfr-1525049382
      displayName: reproduce-bug-9tpfr
      finishedAt: "2020-08-14T03:49:42Z"
      id: reproduce-bug-9tpfr
      name: reproduce-bug-9tpfr
      outboundNodes:
      - reproduce-bug-9tpfr-247809182
      phase: Running
      startedAt: "2020-08-14T03:47:23Z"
      templateName: start
      templateScope: local/reproduce-bug-9tpfr
      type: DAG
    reproduce-bug-9tpfr-247809182:
      boundaryID: reproduce-bug-9tpfr
      displayName: finish
      finishedAt: "2020-08-14T03:49:42Z"
      id: reproduce-bug-9tpfr-247809182
      message: 'omitted: depends condition not met'
      name: reproduce-bug-9tpfr.finish
      phase: Omitted
      startedAt: "2020-08-14T03:49:42Z"
      templateName: finish
      templateScope: local/reproduce-bug-9tpfr
      type: Skipped
    reproduce-bug-9tpfr-546685502:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-3207714925
      displayName: process-tasks(6:6)
      finishedAt: "2020-08-14T03:47:37Z"
      id: reproduce-bug-9tpfr-546685502
      inputs:
        parameters:
        - name: chunk
          value: "6"
      name: reproduce-bug-9tpfr.process-tasks(6:6)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-585646929:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(7:7)(0)
      finishedAt: "2020-08-14T03:47:33Z"
      hostNodeName: gke-dotdb3-dotdb-pool-c3910e16-nb0l
      id: reproduce-bug-9tpfr-585646929
      inputs:
        parameters:
        - name: chunk
          value: "7"
      name: reproduce-bug-9tpfr.process-tasks(7:7)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-600389385:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(5:5)(0)
      finishedAt: "2020-08-14T03:47:33Z"
      hostNodeName: gke-dotdb3-dotdb-pool-c3910e16-nb0l
      id: reproduce-bug-9tpfr-600389385
      inputs:
        parameters:
        - name: chunk
          value: "5"
      name: reproduce-bug-9tpfr.process-tasks(5:5)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-850457006:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-3450100829
      displayName: process-tasks(2:2)
      finishedAt: "2020-08-14T03:47:36Z"
      id: reproduce-bug-9tpfr-850457006
      inputs:
        parameters:
        - name: chunk
          value: "2"
      name: reproduce-bug-9tpfr.process-tasks(2:2)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-988357889:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(1:1)(0)
      finishedAt: "2020-08-14T03:47:34Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-988357889
      inputs:
        parameters:
        - name: chunk
          value: "1"
      name: reproduce-bug-9tpfr.process-tasks(1:1)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 2
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-1384515437:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(8:8)(0)
      finishedAt: "2020-08-14T03:47:42Z"
      hostNodeName: gke-dotdb3-dotdb-pool-2e8afe0d-km90
      id: reproduce-bug-9tpfr-1384515437
      inputs:
        parameters:
        - name: chunk
          value: "8"
      name: reproduce-bug-9tpfr.process-tasks(8:8)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 6
        memory: 6
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-1525049382:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-3595395365
      displayName: gen-tasks
      finishedAt: "2020-08-14T03:47:29Z"
      id: reproduce-bug-9tpfr-1525049382
      name: reproduce-bug-9tpfr.gen-tasks
      outputs:
        exitCode: "0"
        result: '["0", "1", "2", "3", "4", "5", "6", "7", "8", "9"]'
      phase: Succeeded
      startedAt: "2020-08-14T03:47:23Z"
      templateName: gen-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-1602779214:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-3522592509
      displayName: process-tasks(4:4)
      finishedAt: "2020-08-14T03:47:45Z"
      id: reproduce-bug-9tpfr-1602779214
      inputs:
        parameters:
        - name: chunk
          value: "4"
      name: reproduce-bug-9tpfr.process-tasks(4:4)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-1670762753:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-2788826366
      - reproduce-bug-9tpfr-3412607194
      - reproduce-bug-9tpfr-850457006
      - reproduce-bug-9tpfr-3505932898
      - reproduce-bug-9tpfr-1602779214
      - reproduce-bug-9tpfr-4143798034
      - reproduce-bug-9tpfr-546685502
      - reproduce-bug-9tpfr-3415721642
      - reproduce-bug-9tpfr-3221614398
      - reproduce-bug-9tpfr-3518128714
      displayName: process-tasks
      finishedAt: "2020-08-14T03:49:42Z"
      id: reproduce-bug-9tpfr-1670762753
      name: reproduce-bug-9tpfr.process-tasks
      phase: Failed
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: TaskGroup
    reproduce-bug-9tpfr-2008331609:
      boundaryID: reproduce-bug-9tpfr
      displayName: process-tasks(3:3)(0)
      finishedAt: "2020-08-14T03:47:42Z"
      hostNodeName: gke-dotdb3-dotdb-pool-2e8afe0d-km90
      id: reproduce-bug-9tpfr-2008331609
      inputs:
        parameters:
        - name: chunk
          value: "3"
      message: failed with exit code 1
      name: reproduce-bug-9tpfr.process-tasks(3:3)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 6
        memory: 6
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-2475724017:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(9:9)(0)
      finishedAt: "2020-08-14T03:47:34Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-2475724017
      inputs:
        parameters:
        - name: chunk
          value: "9"
      name: reproduce-bug-9tpfr.process-tasks(9:9)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-2612472988:
      boundaryID: reproduce-bug-9tpfr
      displayName: process-tasks(3:3)(1)
      finishedAt: "2020-08-14T03:48:05Z"
      hostNodeName: gke-dotdb3-dotdb-pool-2e8afe0d-km90
      id: reproduce-bug-9tpfr-2612472988
      inputs:
        parameters:
        - name: chunk
          value: "3"
      message: failed with exit code 1
      name: reproduce-bug-9tpfr.process-tasks(3:3)(1)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 2
        memory: 2
      startedAt: "2020-08-14T03:48:00Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-2788826366:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-3662458541
      displayName: process-tasks(0:0)
      finishedAt: "2020-08-14T03:47:44Z"
      id: reproduce-bug-9tpfr-2788826366
      inputs:
        parameters:
        - name: chunk
          value: "0"
      name: reproduce-bug-9tpfr.process-tasks(0:0)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3207714925:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(6:6)(0)
      finishedAt: "2020-08-14T03:47:34Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-3207714925
      inputs:
        parameters:
        - name: chunk
          value: "6"
      name: reproduce-bug-9tpfr.process-tasks(6:6)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 2
        memory: 2
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-3221614398:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-1384515437
      displayName: process-tasks(8:8)
      finishedAt: "2020-08-14T03:47:44Z"
      id: reproduce-bug-9tpfr-3221614398
      inputs:
        parameters:
        - name: chunk
          value: "8"
      name: reproduce-bug-9tpfr.process-tasks(8:8)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3412607194:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-988357889
      displayName: process-tasks(1:1)
      finishedAt: "2020-08-14T03:47:36Z"
      id: reproduce-bug-9tpfr-3412607194
      inputs:
        parameters:
        - name: chunk
          value: "1"
      name: reproduce-bug-9tpfr.process-tasks(1:1)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3415721642:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-585646929
      displayName: process-tasks(7:7)
      finishedAt: "2020-08-14T03:47:36Z"
      id: reproduce-bug-9tpfr-3415721642
      inputs:
        parameters:
        - name: chunk
          value: "7"
      name: reproduce-bug-9tpfr.process-tasks(7:7)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3450100829:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(2:2)(0)
      finishedAt: "2020-08-14T03:47:33Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-3450100829
      inputs:
        parameters:
        - name: chunk
          value: "2"
      name: reproduce-bug-9tpfr.process-tasks(2:2)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-3505932898:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-2008331609
      - reproduce-bug-9tpfr-2612472988
      - reproduce-bug-9tpfr-4222579959
      displayName: process-tasks(3:3)
      finishedAt: "2020-08-14T03:49:42Z"
      id: reproduce-bug-9tpfr-3505932898
      inputs:
        parameters:
        - name: chunk
          value: "3"
      message: No more retries left
      name: reproduce-bug-9tpfr.process-tasks(3:3)
      phase: Failed
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3518128714:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-2475724017
      displayName: process-tasks(9:9)
      finishedAt: "2020-08-14T03:47:37Z"
      id: reproduce-bug-9tpfr-3518128714
      inputs:
        parameters:
        - name: chunk
          value: "9"
      name: reproduce-bug-9tpfr.process-tasks(9:9)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:30Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-3522592509:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(4:4)(0)
      finishedAt: "2020-08-14T03:47:42Z"
      hostNodeName: gke-dotdb3-dotdb-pool-2e8afe0d-km90
      id: reproduce-bug-9tpfr-3522592509
      inputs:
        parameters:
        - name: chunk
          value: "4"
      name: reproduce-bug-9tpfr.process-tasks(4:4)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 6
        memory: 6
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-3595395365:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-1670762753
      displayName: gen-tasks(0)
      finishedAt: "2020-08-14T03:47:26Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-3595395365
      name: reproduce-bug-9tpfr.gen-tasks(0)
      outputs:
        exitCode: "0"
        result: '["0", "1", "2", "3", "4", "5", "6", "7", "8", "9"]'
      phase: Succeeded
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:47:23Z"
      templateName: gen-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-3662458541:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(0:0)(0)
      finishedAt: "2020-08-14T03:47:42Z"
      hostNodeName: gke-dotdb3-dotdb-pool-2e8afe0d-km90
      id: reproduce-bug-9tpfr-3662458541
      inputs:
        parameters:
        - name: chunk
          value: "0"
      name: reproduce-bug-9tpfr.process-tasks(0:0)(0)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 6
        memory: 6
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
    reproduce-bug-9tpfr-4143798034:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-600389385
      displayName: process-tasks(5:5)
      finishedAt: "2020-08-14T03:47:36Z"
      id: reproduce-bug-9tpfr-4143798034
      inputs:
        parameters:
        - name: chunk
          value: "5"
      name: reproduce-bug-9tpfr.process-tasks(5:5)
      outputs:
        exitCode: "0"
      phase: Succeeded
      startedAt: "2020-08-14T03:47:29Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Retry
    reproduce-bug-9tpfr-4222579959:
      boundaryID: reproduce-bug-9tpfr
      children:
      - reproduce-bug-9tpfr-247809182
      displayName: process-tasks(3:3)(2)
      finishedAt: "2020-08-14T03:48:40Z"
      hostNodeName: gke-dotdb3-dotdb-pool-3779d805-hkcv
      id: reproduce-bug-9tpfr-4222579959
      inputs:
        parameters:
        - name: chunk
          value: "3"
      message: failed with exit code 1
      name: reproduce-bug-9tpfr.process-tasks(3:3)(2)
      nodeFlag:
        retried: true
      outputs:
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 1
        memory: 1
      startedAt: "2020-08-14T03:48:37Z"
      templateName: process-tasks
      templateScope: local/reproduce-bug-9tpfr
      type: Pod
  phase: Running
  resourcesDuration:
    cpu: 36
    memory: 36
  startedAt: "2020-08-14T03:47:23Z"
`

func TestFailsWithParamDAG(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := wfv1.MustUnmarshalWorkflow(testFailsWithParamDAG)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
}

var testLeafContinueOn = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: build-wf-kpxvm
spec:

  entrypoint: test-workflow
  templates:
  -
    dag:
      tasks:
      -
        name: A
        template: ok
      -
        continueOn:
          failed: true
        dependencies:
        - A
        name: B
        template: fail
    inputs: {}
    metadata: {}
    name: test-workflow
    outputs: {}
  -
    container:
      args:
      - |
        exit 0
      command:
      - sh
      - -c
      image: busybox
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: ok
    outputs: {}
  -
    container:
      args:
      - |
        exit 1
      command:
      - sh
      - -c
      image: busybox
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: fail
    outputs: {}
status:
  finishedAt: "2020-11-04T16:17:59Z"
  nodes:
    build-wf-kpxvm:
      children:
      - build-wf-kpxvm-2225940411
      displayName: build-wf-kpxvm
      finishedAt: "2020-11-04T16:17:59Z"
      id: build-wf-kpxvm
      name: build-wf-kpxvm
      outboundNodes:
      - build-wf-kpxvm-2242718030
      phase: Running
      progress: 3/3
      resourcesDuration:
        cpu: 13
        memory: 6
      startedAt: "2020-11-04T16:17:43Z"
      templateName: test-workflow
      templateScope: local/build-wf-kpxvm
      type: DAG
    build-wf-kpxvm-2225940411:
      boundaryID: build-wf-kpxvm
      children:
      - build-wf-kpxvm-2242718030
      displayName: A
      finishedAt: "2020-11-04T16:17:51Z"
      hostNodeName: minikube
      id: build-wf-kpxvm-2225940411
      name: build-wf-kpxvm.A
      phase: Succeeded
      startedAt: "2020-11-04T16:17:43Z"
      templateName: ok
      templateScope: local/build-wf-kpxvm
      type: Pod
    build-wf-kpxvm-2242718030:
      boundaryID: build-wf-kpxvm
      displayName: B
      finishedAt: "2020-11-04T16:17:57Z"
      hostNodeName: minikube
      id: build-wf-kpxvm-2242718030
      message: failed with exit code 1
      name: build-wf-kpxvm.B
      phase: Failed
      startedAt: "2020-11-04T16:17:53Z"
      templateName: fail
      templateScope: local/build-wf-kpxvm
      type: Pod
  phase: Running
  progress: 3/3
  resourcesDuration:
    cpu: 13
    memory: 6
  startedAt: "2020-11-04T16:17:43Z"

`

func TestLeafContinueOn(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testLeafContinueOn)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

var dagOutputsReferTaskAggregatedOuputs = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: parameter-aggregation-dag-h8b82
spec:

  entrypoint: parameter-aggregation
  templates:
  -
    dag:
      tasks:
      - arguments:
          parameters:
          - name: num
            value: '{{item}}'
        name: odd-or-even
        template: odd-or-even
        withItems:
        - 1
        - 2
    inputs: {}
    metadata: {}
    name: parameter-aggregation
    outputs:
      parameters:
      - name: dag-nums
        valueFrom:
          parameter: '{{tasks.odd-or-even.outputs.parameters.num}}'
      - name: dag-evenness
        valueFrom:
          parameter: '{{tasks.odd-or-even.outputs.parameters.evenness}}'
  -
    container:
      args:
      - |
        sleep 1 &&
        echo {{inputs.parameters.num}} > /tmp/num &&
        if [ $(({{inputs.parameters.num}}%2)) -eq 0 ]; then
          echo "even" > /tmp/even;
        else
          echo "odd" > /tmp/even;
        fi
      command:
      - sh
      - -c
      image: alpine:3.23
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: num
    metadata: {}
    name: odd-or-even
    outputs:
      parameters:
      - name: num
        valueFrom:
          path: /tmp/num
      - name: evenness
        valueFrom:
          path: /tmp/even
status:
  nodes:
    parameter-aggregation-dag-h8b82:
      children:
      - parameter-aggregation-dag-h8b82-3379492521
      displayName: parameter-aggregation-dag-h8b82
      finishedAt: "2020-12-09T15:37:07Z"
      id: parameter-aggregation-dag-h8b82
      name: parameter-aggregation-dag-h8b82
      outboundNodes:
      - parameter-aggregation-dag-h8b82-3175470584
      - parameter-aggregation-dag-h8b82-2243926302
      phase: Running
      startedAt: "2020-12-09T15:36:46Z"
      templateName: parameter-aggregation
      templateScope: local/parameter-aggregation-dag-h8b82
      type: DAG
    parameter-aggregation-dag-h8b82-1440345089:
      boundaryID: parameter-aggregation-dag-h8b82
      displayName: odd-or-even(1:2)
      finishedAt: "2020-12-09T15:36:54Z"
      hostNodeName: minikube
      id: parameter-aggregation-dag-h8b82-1440345089
      inputs:
        parameters:
        - name: num
          value: "2"
      name: parameter-aggregation-dag-h8b82.odd-or-even(1:2)
      outputs:
        exitCode: "0"
        parameters:
        - name: num
          value: "2"
          valueFrom:
            path: /tmp/num
        - name: evenness
          value: even
          valueFrom:
            path: /tmp/even
      phase: Succeeded
      startedAt: "2020-12-09T15:36:46Z"
      templateName: odd-or-even
      templateScope: local/parameter-aggregation-dag-h8b82
      type: Pod
    parameter-aggregation-dag-h8b82-3379492521:
      boundaryID: parameter-aggregation-dag-h8b82
      children:
      - parameter-aggregation-dag-h8b82-3572919299
      - parameter-aggregation-dag-h8b82-1440345089
      displayName: odd-or-even
      finishedAt: "2020-12-09T15:36:55Z"
      id: parameter-aggregation-dag-h8b82-3379492521
      name: parameter-aggregation-dag-h8b82.odd-or-even
      phase: Succeeded
      startedAt: "2020-12-09T15:36:46Z"
      templateName: odd-or-even
      templateScope: local/parameter-aggregation-dag-h8b82
      type: TaskGroup
    parameter-aggregation-dag-h8b82-3572919299:
      boundaryID: parameter-aggregation-dag-h8b82
      displayName: odd-or-even(0:1)
      finishedAt: "2020-12-09T15:36:53Z"
      hostNodeName: minikube
      id: parameter-aggregation-dag-h8b82-3572919299
      inputs:
        parameters:
        - name: num
          value: "1"
      name: parameter-aggregation-dag-h8b82.odd-or-even(0:1)
      outputs:
        exitCode: "0"
        parameters:
        - name: num
          value: "1"
          valueFrom:
            path: /tmp/num
        - name: evenness
          value: odd
          valueFrom:
            path: /tmp/even
      phase: Succeeded
      startedAt: "2020-12-09T15:36:46Z"
      templateName: odd-or-even
      templateScope: local/parameter-aggregation-dag-h8b82
      type: Pod
  phase: Succeeded
  startedAt: "2020-12-09T15:36:46Z"
`

func TestDAGReferTaskAggregatedOutputs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(dagOutputsReferTaskAggregatedOuputs)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	dagNode := woc.wf.Status.Nodes.FindByDisplayName("parameter-aggregation-dag-h8b82")
	require.NotNil(t, dagNode)
	require.NotNil(t, dagNode.Outputs)
	require.Len(t, dagNode.Outputs.Parameters, 2)
	assert.Equal(t, `["1","2"]`, dagNode.Outputs.Parameters[0].Value.String())
	assert.Equal(t, `["odd","even"]`, dagNode.Outputs.Parameters[1].Value.String())
}

var dagHTTPChildrenAssigned = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: http-template-nv52d
spec:
  entrypoint: main
  templates:
  - dag:
      tasks:
      - arguments:
          parameters:
          - name: url
            value: https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json
        name: good1
        template: http
      - arguments:
          parameters:
          - name: url
            value: https://raw.githubusercontent.com/argoproj/argo-workflows/4e450e250168e6b4d51a126b784e90b11a0162bc/pkg/apis/workflow/v1alpha1/generated.swagger.json
        dependencies:
        - good1
        name: good2
        template: http
    name: main
  - http:
      url: '{{inputs.parameters.url}}'
    inputs:
      parameters:
      - name: url
    name: http
status:
  nodes:
    http-template-nv52d:
      children:
      - http-template-nv52d-444770636
      displayName: http-template-nv52d
      id: http-template-nv52d
      name: http-template-nv52d
      outboundNodes:
      - http-template-nv52d-478325874
      phase: Running
      startedAt: "2021-10-27T13:46:08Z"
      templateName: main
      templateScope: local/http-template-nv52d
      type: DAG
    http-template-nv52d-444770636:
      boundaryID: http-template-nv52d
      children:
      - http-template-nv52d-495103493
      displayName: good1
      finishedAt: null
      id: http-template-nv52d-444770636
      name: http-template-nv52d.good1
      phase: Succeeded
      startedAt: "2021-10-27T13:46:08Z"
      templateName: http
      templateScope: local/http-template-nv52d
      type: HTTP
  phase: Running
  startedAt: "2021-10-27T13:46:08Z"
`

func TestDagHttpChildrenAssigned(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(dagHTTPChildrenAssigned)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	dagNode := woc.wf.Status.Nodes.FindByDisplayName("good2")
	assert.NotNil(t, dagNode)

	dagNode = woc.wf.Status.Nodes.FindByDisplayName("good1")
	require.NotNil(t, dagNode)
	require.Len(t, dagNode.Children, 1)
	assert.Equal(t, "http-template-nv52d-495103493", dagNode.Children[0])
}

var retryTypeDagTaskRunExitNodeAfterCompleted = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  labels:
    workflows.argoproj.io/phase: Running
  name: test-workflow-with-hang-cztfs
  namespace: argo-system
spec:
  entrypoint: dag
  templates:
  - name: linuxExitHandler
    steps:
    - - name: print-exit
        template: print-exit
  - container:
      args:
      - echo
      - exit
      command:
      - /argosay
      image: argoproj/argosay:v2
      name: ""
    name: print-exit
  - container:
      args:
      - echo
      - a
      command:
      - /argosay
      image: argoproj/argosay:v2
      name: ""
    name: printA
    retryStrategy:
      limit: "3"
      retryPolicy: OnError
  - dag:
      tasks:
      - hooks:
          exit:
            template: linuxExitHandler
        name: printA
        template: printA
      - depends: printA.Succeeded
        hooks:
          exit:
            template: linuxExitHandler
        name: dependencyTesting
        template: printA
    name: dag
status:
  nodes:
    test-workflow-with-hang-cztfs:
      children:
      - test-workflow-with-hang-cztfs-1556528266
      displayName: test-workflow-with-hang-cztfs
      finishedAt: null
      id: test-workflow-with-hang-cztfs
      name: test-workflow-with-hang-cztfs
      phase: Running
      progress: 4/4
      startedAt: "2022-08-04T02:28:38Z"
      templateName: dag
      templateScope: local/test-workflow-with-hang-cztfs
      type: DAG
    test-workflow-with-hang-cztfs-589413809:
      boundaryID: test-workflow-with-hang-cztfs
      children:
      - test-workflow-with-hang-cztfs-527957059
      displayName: printA(0)
      finishedAt: "2022-08-04T02:28:43Z"
      hostNodeName: node2
      id: test-workflow-with-hang-cztfs-589413809
      name: test-workflow-with-hang-cztfs.printA(0)
      outputs:
        exitCode: "0"
      phase: Succeeded
      progress: 1/1
      resourcesDuration:
        cpu: 2
        memory: 2
      startedAt: "2022-08-04T02:28:38Z"
      templateName: printA
      templateScope: local/test-workflow-with-hang-cztfs
      type: Pod
    test-workflow-with-hang-cztfs-1556528266:
      boundaryID: test-workflow-with-hang-cztfs
      children:
      - test-workflow-with-hang-cztfs-589413809
      displayName: printA
      finishedAt: "2022-08-04T02:28:48Z"
      id: test-workflow-with-hang-cztfs-1556528266
      name: test-workflow-with-hang-cztfs.printA
      outputs:
        exitCode: "0"
      phase: Succeeded
      progress: 4/4
      resourcesDuration:
        cpu: 5
        memory: 5
      startedAt: "2022-08-04T02:28:38Z"
      templateName: printA
      templateScope: local/test-workflow-with-hang-cztfs
      type: Retry
  phase: Running
  progress: 4/4
  resourcesDuration:
    cpu: 5
    memory: 5
  startedAt: "2022-08-04T02:28:38Z"
`

func TestRetryTypeDagTaskRunExitNodeAfterCompleted(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(retryTypeDagTaskRunExitNodeAfterCompleted)
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	// retryTypeDAGTask completed
	printAChild := woc.wf.Status.Nodes.FindByDisplayName("printA(0)")
	assert.Equal(t, wfv1.NodeSucceeded, printAChild.Phase)

	// run ExitNode
	woc.operate(ctx)
	onExitNode := woc.wf.Status.Nodes.FindByDisplayName("printA.onExit")
	assert.NotNil(t, onExitNode)
	assert.Equal(t, wfv1.NodeRunning, onExitNode.Phase)
	assert.True(t, onExitNode.NodeFlag.Hooked)

	// exitNode succeeded
	makePodsPhase(ctx, woc, v1.PodSucceeded)
	woc.operate(ctx)
	onExitNode = woc.wf.Status.Nodes.FindByDisplayName("printA.onExit")
	assert.Equal(t, wfv1.NodeSucceeded, onExitNode.Phase)
	assert.True(t, onExitNode.NodeFlag.Hooked)

	// run next DAGTask
	woc.operate(ctx)
	nextDAGTaskNode := woc.wf.Status.Nodes.FindByDisplayName("dependencyTesting")
	assert.NotNil(t, nextDAGTaskNode)
	assert.Equal(t, wfv1.NodeRunning, nextDAGTaskNode.Phase)
}

func TestDagParallelism(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-parallelism
  namespace: argo
spec:
  entrypoint: main
  parallelism: 1
  templates:
    - name: main
      dag:
        tasks:
          - name: do-it-once
            template: do-it
            arguments:
              parameters:
                - name: thing
                  value: 1
          - name: do-it-twice
            template: do-it
            arguments:
              parameters:
                - name: thing
                  value: 2
          - name: do-it-thrice
            template: do-it
            arguments:
              parameters:
                - name: thing
                  value: 3
    - name: do-it
      inputs:
        parameters:
          - name: thing
      container:
        image: docker/whalesay:latest
        command: [cowsay]
        args: ["I have a {{inputs.parameters.thing}}"]`)

	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	woc1 := newWoc(ctx, *woc.wf)
	woc1.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestDagWftmplHookWithRetry(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/dag_wftmpl_hook_with_retry.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)

	// assert task kicked
	taskNode := woc.wf.Status.Nodes.FindByDisplayName("task")
	assert.Equal(t, wfv1.NodePending, taskNode.Phase)

	// task failed
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc.operate(ctx)

	// onFailure retry hook(0) kicked
	taskNode = woc.wf.Status.Nodes.FindByDisplayName("task")
	assert.Equal(t, wfv1.NodeFailed, taskNode.Phase)
	failHookRetryNode := woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure")
	failHookChild0Node := woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure(0)")
	assert.Equal(t, wfv1.NodeRunning, failHookRetryNode.Phase)
	assert.Equal(t, wfv1.NodePending, failHookChild0Node.Phase)

	// onFailure retry hook(0) failed
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc.operate(ctx)

	// onFailure retry hook(1) kicked
	taskNode = woc.wf.Status.Nodes.FindByDisplayName("task")
	assert.Equal(t, wfv1.NodeFailed, taskNode.Phase)
	failHookRetryNode = woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure")
	failHookChild0Node = woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure(0)")
	failHookChild1Node := woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure(1)")
	assert.Equal(t, wfv1.NodeRunning, failHookRetryNode.Phase)
	assert.Equal(t, wfv1.NodeFailed, failHookChild0Node.Phase)
	assert.Equal(t, wfv1.NodePending, failHookChild1Node.Phase)

	// onFailure retry hook(1) failed
	makePodsPhase(ctx, woc, v1.PodFailed)
	woc.operate(ctx)

	// onFailure retry node faled
	taskNode = woc.wf.Status.Nodes.FindByDisplayName("task")
	assert.Equal(t, wfv1.NodeFailed, taskNode.Phase)
	failHookRetryNode = woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure")
	failHookChild0Node = woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure(0)")
	failHookChild1Node = woc.wf.Status.Nodes.FindByDisplayName("task.hooks.failure(1)")
	assert.Equal(t, wfv1.NodeFailed, failHookRetryNode.Phase)
	assert.Equal(t, wfv1.NodeFailed, failHookChild0Node.Phase)
	assert.Equal(t, wfv1.NodeFailed, failHookChild1Node.Phase)
	// finish Node skipped
	finishNode := woc.wf.Status.Nodes.FindByDisplayName("finish")
	assert.Equal(t, wfv1.NodeOmitted, finishNode.Phase)
}
