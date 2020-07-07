package controller

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/argoproj/argo/workflow/common"

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
	assert.Equal(t, wfv1.NodeSucceeded, woc.wf.Status.Phase)
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

var dagAssessPhaseContinueOnExpandedTaskVariables = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: parameter-aggregation-one-will-fail2-jt776
spec:
  arguments: {}
  entrypoint: parameter-aggregation-one-will-fail2
  templates:
  - arguments: {}
    dag:
      tasks:
      - arguments: {}
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
      - arguments: {}
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
  - arguments: {}
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
      image: alpine:latest
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: num
    metadata: {}
    name: one-will-fail
    outputs: {}
  - arguments: {}
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
  - arguments: {}
    inputs: {}
    metadata: {}
    name: gen-number-list
    outputs: {}
    script:
      command:
      - python
      image: python:alpine3.6
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(dagAssessPhaseContinueOnExpandedTaskVariables)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	woc.operate()
	assert.Equal(t, wfv1.NodeSucceeded, woc.wf.Status.Phase)
}

var dagAssessPhaseContinueOnExpandedTask = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: parameter-aggregation-one-will-fail-69x7k
spec:
  arguments: {}
  entrypoint: parameter-aggregation-one-will-fail
  templates:
  - arguments: {}
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
      - arguments: {}
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
  - arguments: {}
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
      image: alpine:latest
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: num
    metadata: {}
    name: one-will-fail
    outputs: {}
  - arguments: {}
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(dagAssessPhaseContinueOnExpandedTask)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	woc.operate()
	assert.Equal(t, wfv1.NodeSucceeded, woc.wf.Status.Phase)
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(dagWithParamAndGlobalParam)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
}

var terminatingDAGWithRetryStrategyNodes = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-diamond-xfww2
spec:
  arguments: {}
  entrypoint: diamond
  shutdown: Terminate
  templates:
  - arguments: {}
    dag:
      tasks:
      - arguments: {}
        name: A
        template: echo
      - arguments: {}
        dependencies:
        - A
        name: B
        template: echo
      - arguments: {}
        dependencies:
        - A
        name: C
        template: echo
      - arguments: {}
        dependencies:
        - B
        - C
        name: D
        template: echo
    inputs: {}
    metadata: {}
    name: diamond
    outputs: {}
  - arguments: {}
    container:
      args:
      - sleep 10
      command:
      - sh
      - -c
      image: alpine:3.7
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(terminatingDAGWithRetryStrategyNodes)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Phase)
}

var terminateDAGWithMaxDurationLimitExpiredAndMoreAttempts = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-diamond-dj7q5
spec:
  arguments: {}
  entrypoint: diamond
  templates:
  - arguments: {}
    dag:
      tasks:
      - arguments: {}
        name: A
        template: echo
      - arguments: {}
        dependencies:
        - A
        name: B
        template: echo
    inputs: {}
    metadata: {}
    name: diamond
    outputs: {}
  - arguments: {}
    container:
      args:
      - exit 1
      command:
      - sh
      - -c
      image: alpine:3.7
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
  phase: Running
  resourcesDuration:
    cpu: 2
    memory: 0
  startedAt: "2020-05-27T15:41:54Z"
`

// This tests that a DAG with retry strategy in its tasks fails successfully when terminated
func TestTerminateDAGWithMaxDurationLimitExpiredAndMoreAttempts(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(terminateDAGWithMaxDurationLimitExpiredAndMoreAttempts)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()

	retryNode := woc.getNodeByName("dag-diamond-dj7q5.A")
	if assert.NotNil(t, retryNode) {
		assert.Equal(t, wfv1.NodeFailed, retryNode.Phase)
		assert.Contains(t, retryNode.Message, "Max duration limit exceeded")
	}

	woc.operate()

	// This is the crucial part of the test
	assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Phase)
}

var testRetryStrategyNodes = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: wf-retry-pol
spec:
  arguments: {}
  entrypoint: run-steps
  onExit: onExit
  templates:
  - arguments: {}
    inputs: {}
    metadata: {}
    name: run-steps
    outputs: {}
    steps:
    - - arguments: {}
        name: run-dag
        template: run-dag
    - - arguments: {}
        name: manual-onExit
        template: onExit
  - arguments: {}
    dag:
      tasks:
      - arguments: {}
        name: A
        template: fail
      - arguments: {}
        dependencies:
        - A
        name: B
        template: onExit
    inputs: {}
    metadata: {}
    name: run-dag
    outputs: {}
  - arguments: {}
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
  - arguments: {}
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(testRetryStrategyNodes)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	retryNode := woc.getNodeByName("wf-retry-pol")
	if assert.NotNil(t, retryNode) {
		assert.Equal(t, wfv1.NodeRunning, retryNode.Phase)
	}

	woc.operate()
	retryNode = woc.getNodeByName("wf-retry-pol")
	if assert.NotNil(t, retryNode) {
		assert.Equal(t, wfv1.NodeFailed, retryNode.Phase)
	}

	onExitNode := woc.getNodeByName("wf-retry-pol.onExit")
	if assert.NotNil(t, onExitNode) {
		assert.Equal(t, wfv1.NodePending, onExitNode.Phase)
	}

	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
}

var testOnExitNodeDAGPhase = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-diamond-88trp
spec:
  arguments: {}
  entrypoint: diamond
  templates:
  - arguments: {}
    dag:
      failFast: false
      tasks:
      - arguments: {}
        name: A
        template: echo
      - arguments: {}
        dependencies:
        - A
        name: B
        onExit: echo
        template: echo
    inputs: {}
    metadata: {}
    name: diamond
    outputs: {}
  - arguments: {}
    container:
      args:
      - exit 0
      command:
      - sh
      - -c
      image: alpine:3.7
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: echo
    outputs: {}
  - arguments: {}
    container:
      args:
      - exit 1
      command:
      - sh
      - -c
      image: alpine:3.7
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(testOnExitNodeDAGPhase)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	retryNode := woc.getNodeByName("dag-diamond-88trp")
	if assert.NotNil(t, retryNode) {
		assert.Equal(t, wfv1.NodeRunning, retryNode.Phase)
	}

	retryNode = woc.getNodeByName("B.onExit")
	if assert.NotNil(t, retryNode) {
		assert.Equal(t, wfv1.NodePending, retryNode.Phase)
	}

	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
}

var testOnExitNonLeaf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: exit-handler-bug-example
spec:
  arguments: {}
  entrypoint: dag
  templates:
  - arguments: {}
    dag:
      tasks:
      - arguments: {}
        name: step-2
        onExit: on-exit
        template: step-template
      - arguments: {}
        dependencies:
        - step-2
        name: step-3
        onExit: on-exit
        template: step-template
    inputs: {}
    metadata: {}
    name: dag
    outputs: {}
  - arguments: {}
    container:
      args:
      - echo exit-handler-step-{{pod.name}}
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: on-exit
    outputs: {}
  - arguments: {}
    container:
      args:
      - echo step {{pod.name}}
      command:
      - sh
      - -c
      image: alpine:latest
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(testOnExitNonLeaf)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	retryNode := woc.getNodeByName("step-2.onExit")
	if assert.NotNil(t, retryNode) {
		assert.Equal(t, wfv1.NodePending, retryNode.Phase)
	}
	retryNode = woc.getNodeByName("exit-handler-bug-example.step-3")
	if assert.NotNil(t, retryNode) {
		assert.Equal(t, wfv1.NodePending, retryNode.Phase)
	}
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
}
