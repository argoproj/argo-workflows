package controller

import (
	"fmt"
	"testing"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test"
	"github.com/argoproj/argo/workflow/util"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestOperateWorkflowPanicRecover ensures we can recover from unexpected panics
func TestOperateWorkflowPanicRecover(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fail()
		}
	}()
	controller := newController()
	// intentionally set clientset to nil to induce panic
	controller.kubeclientset = nil
	wf := unmarshalWF(helloWorldWf)
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("").Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
}

var sidecarWithVol = `
# Verifies sidecars can reference volumeClaimTemplates
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: sidecar-with-volumes
spec:
  entrypoint: sidecar-with-volumes
  volumeClaimTemplates:
  - metadata:
      name: claim-vol
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 1Gi
  volumes:
  - name: existing-vol
    persistentVolumeClaim:
      claimName: my-existing-volume
  templates:
  - name: sidecar-with-volumes
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        print("hello world")
    sidecars:
    - name: sidevol
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["echo generating message in volume; cowsay hello world | tee /mnt/vol/hello_world.txt; sleep 9999"]
      volumeMounts:
      - name: claim-vol
        mountPath: /mnt/vol
      - name: existing-vol
        mountPath: /mnt/existing-vol
`

// TestSidecarWithVolume verifies ia sidecar can have a volumeMount reference to both existing or volumeClaimTemplate volumes
func TestSidecarWithVolume(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(sidecarWithVol)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	pods, err := controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.True(t, len(pods.Items) > 0, "pod was not created successfully")
	pod := pods.Items[0]

	claimVolFound := false
	existingVolFound := false
	for _, ctr := range pod.Spec.Containers {
		if ctr.Name == "sidevol" {
			for _, vol := range ctr.VolumeMounts {
				if vol.Name == "claim-vol" {
					claimVolFound = true
				}
				if vol.Name == "existing-vol" {
					existingVolFound = true
				}
			}
		}
	}
	assert.True(t, claimVolFound, "claim vol was not referenced by sidecar")
	assert.True(t, existingVolFound, "existing vol was not referenced by sidecar")
}

// TestProcessNodesWithRetries tests the processNodesWithRetries() method.
func TestProcessNodesWithRetries(t *testing.T) {
	controller := newController()
	assert.NotNil(t, controller)
	wf := unmarshalWF(helloWorldWf)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)

	// Verify that there are no nodes in the wf status.
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	var retryLimit int32
	retryLimit = 2
	retries.Limit = &retryLimit
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild, err := woc.getLastChildNode(node)
	assert.Nil(t, err)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := 0; i < 2; i++ {
		childNode := fmt.Sprintf("child-node-%d", i)
		woc.initializeNode(childNode, wfv1.NodeTypePod, "", "", wfv1.NodeRunning)
		woc.addChildNode(nodeName, childNode)
	}

	n := woc.getNodeByName(nodeName)
	lastChild, err = woc.getLastChildNode(n)
	assert.Nil(t, err)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodesWithRetries() should return false since
	// there should be no retries at this point.
	err = woc.processNodeRetries(n, retries)
	assert.Nil(t, err)
	n = woc.getNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Mark lastChild as successful.
	woc.markNodePhase(lastChild.Name, wfv1.NodeSucceeded)
	err = woc.processNodeRetries(n, retries)
	assert.Nil(t, err)
	// The parent node also gets marked as Succeeded.
	n = woc.getNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeSucceeded)

	// Mark the parent node as running again and the lastChild as failed.
	woc.markNodePhase(n.Name, wfv1.NodeRunning)
	woc.markNodePhase(lastChild.Name, wfv1.NodeFailed)
	woc.processNodeRetries(n, retries)
	n = woc.getNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Add a third node that has failed.
	childNode := "child-node-3"
	woc.initializeNode(childNode, wfv1.NodeTypePod, "", "", wfv1.NodeFailed)
	woc.addChildNode(nodeName, childNode)
	n = woc.getNodeByName(nodeName)
	err = woc.processNodeRetries(n, retries)
	assert.Nil(t, err)
	n = woc.getNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeFailed)
}

var workflowParallelismLimit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: parallelism-limit
spec:
  entrypoint: parallelism-limit
  parallelism: 2
  templates:
  - name: parallelism-limit
    steps:
    - - name: sleep
        template: sleep
        withItems:
        - this
        - workflow
        - should
        - take
        - at
        - least
        - 60
        - seconds
        - to
        - complete

  - name: sleep
    container:
      image: alpine:latest
      command: [sh, -c, sleep 10]
`

// TestWorkflowParallelismLimit verifies parallelism at a workflow level is honored.
func TestWorkflowParallelismLimit(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(workflowParallelismLimit)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(pods.Items))
	// operate again and make sure we don't schedule any more pods
	makePodsRunning(t, controller.kubeclientset, wf.ObjectMeta.Namespace)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	// wfBytes, _ := json.MarshalIndent(wf, "", "  ")
	// log.Printf("%s", wfBytes)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(pods.Items))
}

var stepsTemplateParallelismLimit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-parallelism-limit
spec:
  entrypoint: steps-parallelism-limit
  templates:
  - name: steps-parallelism-limit
    parallelism: 2
    steps:
    - - name: sleep
        template: sleep
        withItems:
        - this
        - workflow
        - should
        - take
        - at
        - least
        - 60
        - seconds
        - to
        - complete

  - name: sleep
    container:
      image: alpine:latest
      command: [sh, -c, sleep 10]
`

// TestStepsTemplateParallelismLimit verifies parallelism at a steps level is honored.
func TestStepsTemplateParallelismLimit(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(stepsTemplateParallelismLimit)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(pods.Items))

	// operate again and make sure we don't schedule any more pods
	makePodsRunning(t, controller.kubeclientset, wf.ObjectMeta.Namespace)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	// wfBytes, _ := json.MarshalIndent(wf, "", "  ")
	// log.Printf("%s", wfBytes)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(pods.Items))
}

var dagTemplateParallelismLimit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-parallelism-limit
spec:
  entrypoint: dag-parallelism-limit
  templates:
  - name: dag-parallelism-limit
    parallelism: 2
    dag:
      tasks:
      - name: a
        template: sleep
      - name: b
        template: sleep
      - name: c
        template: sleep
      - name: d
        template: sleep
      - name: e
        template: sleep
  - name: sleep
    container:
      image: alpine:latest
      command: [sh, -c, sleep 10]
`

// TestDAGTemplateParallelismLimit verifies parallelism at a dag level is honored.
func TestDAGTemplateParallelismLimit(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(dagTemplateParallelismLimit)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(pods.Items))

	// operate again and make sure we don't schedule any more pods
	makePodsRunning(t, controller.kubeclientset, wf.ObjectMeta.Namespace)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	// wfBytes, _ := json.MarshalIndent(wf, "", "  ")
	// log.Printf("%s", wfBytes)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(pods.Items))
}

var nestedParallelism = `
# Example with vertical and horizontal scalability                                                                                                                                                                                      
#                                                                                                                                                                                                                                                            
# Imagine we have 'M' workers which work in parallel,                                                                                                                                                                                                        
# each worker should performs 'N' loops sequentially                                                                                                                                                                                                         
#                                                                                                                                                                                                                                                            
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: parallelism-nested-
spec:
  arguments:
    parameters:
    - name: seq-list
      value: |
        ["a","b","c","d"]
    - name: parallel-list
      value: |
        [1,2,3,4]

  entrypoint: parallel-worker
  templates:
  - name: parallel-worker
    inputs:
      parameters:
      - name: seq-list
      - name: parallel-list
    steps:
    - - name: parallel-worker
        template: seq-worker
        arguments:
          parameters:
          - name: seq-list
            value: "{{inputs.parameters.seq-list}}"
          - name: parallel-id
            value: "{{item}}"
        withParam: "{{inputs.parameters.parallel-list}}"

  - name: seq-worker
    parallelism: 1
    inputs:
      parameters:
      - name: seq-list
      - name: parallel-id
    steps:
    - - name: seq-step
        template: one-job
        arguments:
          parameters:
          - name: parallel-id
            value: "{{inputs.parameters.parallel-id}}"
          - name: seq-id
            value: "{{item}}"
        withParam: "{{inputs.parameters.seq-list}}"

  - name: one-job
    inputs:
      parameters:
      - name: seq-id
      - name: parallel-id
    container:
      image: alpine
      command: ['/bin/sh', '-c']
      args: ["echo {{inputs.parameters.parallel-id}} {{inputs.parameters.seq-id}}; sleep 10"]
`

func TestNestedTemplateParallelismLimit(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(nestedParallelism)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 4, len(pods.Items))
}

// TestSidecarResourceLimits verifies resource limits on the sidecar can be set in the controller config
func TestSidecarResourceLimits(t *testing.T) {
	controller := newController()
	controller.Config.Executor = &apiv1.Container{
		Resources: apiv1.ResourceRequirements{
			Limits: apiv1.ResourceList{
				apiv1.ResourceCPU:    resource.MustParse("0.5"),
				apiv1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Requests: apiv1.ResourceList{
				apiv1.ResourceCPU:    resource.MustParse("0.1"),
				apiv1.ResourceMemory: resource.MustParse("64Mi"),
			},
		},
	}
	wf := unmarshalWF(helloWorldWf)
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("").Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pod, err := controller.kubeclientset.CoreV1().Pods("").Get("hello-world", metav1.GetOptions{})
	assert.Nil(t, err)
	var waitCtr *apiv1.Container
	for _, ctr := range pod.Spec.Containers {
		if ctr.Name == "wait" {
			waitCtr = &ctr
			break
		}
	}
	assert.NotNil(t, waitCtr)
	assert.Equal(t, 2, len(waitCtr.Resources.Limits))
	assert.Equal(t, 2, len(waitCtr.Resources.Requests))
}

// TestSuspendResume tests the suspend and resume feature
func TestSuspendResume(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(stepsTemplateParallelismLimit)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)

	// suspend the workflow
	err = util.SuspendWorkflow(wfcset, wf.ObjectMeta.Name)
	assert.Nil(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.True(t, *wf.Spec.Suspend)

	// operate should not result in no workflows being created since it is suspended
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// resume the workflow and operate again. two pods should be able to be scheduled
	err = util.ResumeWorkflow(wfcset, wf.ObjectMeta.Name)
	assert.Nil(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Nil(t, wf.Spec.Suspend)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 2, len(pods.Items))
}

var expandWithItems = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: expand-with-items
spec:
  entrypoint: expand-with-items
  templates:
  - name: expand-with-items
    steps:
    - - name: whalesay
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "{{item}}"
        withItems:
        - string
        - 0
        - 0
        - false
        - 1.3

  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay {{inputs.parameters.message}}"]
`

func TestExpandWithItems(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	wf := unmarshalWF(expandWithItems)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	newSteps, err := woc.expandStep(wf.Spec.Templates[0].Steps[0][0])
	assert.Nil(t, err)
	assert.Equal(t, 5, len(newSteps))
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 5, len(pods.Items))
}

var expandWithItemsMap = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: expand-with-items
spec:
  entrypoint: expand-with-items
  templates:
  - name: expand-with-items
    steps:
    - - name: whalesay
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "{{item.os}} {{item.version}}"
        withItems:
        - {os: debian, version: 9.1}
        - {os: debian, version: 9.1}
        - {os: ubuntu, version: 16.10}

  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay {{inputs.parameters.message}}"]
`

func TestExpandWithItemsMap(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(expandWithItemsMap)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	newSteps, err := woc.expandStep(wf.Spec.Templates[0].Steps[0][0])
	assert.Nil(t, err)
	assert.Equal(t, 3, len(newSteps))
}

var suspendTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: suspend-template
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: approve
        template: approve
    - - name: release
        template: whalesay

  - name: approve
    suspend: {}

  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
`

func TestSuspendTemplate(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	wf := unmarshalWF(suspendTemplate)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// resume the workflow. verify resume workflow edits nodestatus correctly
	util.ResumeWorkflow(wfcset, wf.ObjectMeta.Name)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.False(t, util.IsWorkflowSuspended(wf))

	// operate the workflow. it should reach the second step
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(pods.Items))
}

var volumeWithParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: volume-with-param
spec:
  entrypoint: append-to-accesslog
  arguments:
    parameters:
    - name: volname
      value: my-volume
    - name: node-selctor
      value: my-node

  nodeSelector:
    kubernetes.io/hostname: my-host

  volumes:
  - name: workdir
    persistentVolumeClaim:
      claimName: "{{workflow.parameters.volname}}"

  templates:
  - name: append-to-accesslog
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo accessed at: $(date) | tee -a /mnt/vol/accesslog"]
      volumeMounts:
      - name: workdir
        mountPath: /mnt/vol
`

// Tests ability to reference workflow parameters from within top level spec fields (e.g. spec.volumes)
func TestWorkflowSpecParam(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(volumeWithParam)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	pod, err := controller.kubeclientset.CoreV1().Pods("").Get(wf.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	found := false
	for _, vol := range pod.Spec.Volumes {
		if vol.Name == "workdir" {
			assert.Equal(t, "my-volume", vol.PersistentVolumeClaim.ClaimName)
			found = true
		}
	}
	assert.True(t, found)

	assert.Equal(t, "my-host", pod.Spec.NodeSelector["kubernetes.io/hostname"])
}

func TestAddGlobalParamToScope(t *testing.T) {
	woc := newWoc()
	woc.globalParams = make(map[string]string)
	testVal := "test-value"
	param := wfv1.Parameter{
		Name:  "test-param",
		Value: &testVal,
	}
	// Make sure if the param is not global, don't add to scope
	woc.addParamToGlobalScope(param)
	assert.Nil(t, woc.wf.Status.Outputs)

	// Now set it as global. Verify it is added to workflow outputs
	param.GlobalName = "global-param"
	woc.addParamToGlobalScope(param)
	assert.Equal(t, 1, len(woc.wf.Status.Outputs.Parameters))
	assert.Equal(t, param.GlobalName, woc.wf.Status.Outputs.Parameters[0].Name)
	assert.Equal(t, testVal, *woc.wf.Status.Outputs.Parameters[0].Value)
	assert.Equal(t, testVal, woc.globalParams["workflow.outputs.parameters.global-param"])

	// Change the value and verify it is reflected in workflow outputs
	newValue := "new-value"
	param.Value = &newValue
	woc.addParamToGlobalScope(param)
	assert.Equal(t, 1, len(woc.wf.Status.Outputs.Parameters))
	assert.Equal(t, param.GlobalName, woc.wf.Status.Outputs.Parameters[0].Name)
	assert.Equal(t, newValue, *woc.wf.Status.Outputs.Parameters[0].Value)
	assert.Equal(t, newValue, woc.globalParams["workflow.outputs.parameters.global-param"])

	// Add a new global parameter
	param.GlobalName = "global-param2"
	woc.addParamToGlobalScope(param)
	assert.Equal(t, 2, len(woc.wf.Status.Outputs.Parameters))
	assert.Equal(t, param.GlobalName, woc.wf.Status.Outputs.Parameters[1].Name)
	assert.Equal(t, newValue, *woc.wf.Status.Outputs.Parameters[1].Value)
	assert.Equal(t, newValue, woc.globalParams["workflow.outputs.parameters.global-param2"])

}

func TestAddGlobalArtifactToScope(t *testing.T) {
	woc := newWoc()
	art := wfv1.Artifact{
		Name: "test-art",
		ArtifactLocation: wfv1.ArtifactLocation{
			S3: &wfv1.S3Artifact{
				S3Bucket: wfv1.S3Bucket{
					Bucket: "my-bucket",
				},
				Key: "some/key",
			},
		},
	}
	// Make sure if the artifact is not global, don't add to scope
	woc.addArtifactToGlobalScope(art, nil)
	assert.Nil(t, woc.wf.Status.Outputs)

	// Now mark it as global. Verify it is added to workflow outputs
	art.GlobalName = "global-art"
	woc.addArtifactToGlobalScope(art, nil)
	assert.Equal(t, 1, len(woc.wf.Status.Outputs.Artifacts))
	assert.Equal(t, art.GlobalName, woc.wf.Status.Outputs.Artifacts[0].Name)
	assert.Equal(t, "some/key", woc.wf.Status.Outputs.Artifacts[0].S3.Key)

	// Change the value and verify update is reflected
	art.S3.Key = "new/key"
	woc.addArtifactToGlobalScope(art, nil)
	assert.Equal(t, 1, len(woc.wf.Status.Outputs.Artifacts))
	assert.Equal(t, art.GlobalName, woc.wf.Status.Outputs.Artifacts[0].Name)
	assert.Equal(t, "new/key", woc.wf.Status.Outputs.Artifacts[0].S3.Key)

	// Add a new global artifact
	art.GlobalName = "global-art2"
	art.S3.Key = "new/new/key"
	woc.addArtifactToGlobalScope(art, nil)
	assert.Equal(t, 2, len(woc.wf.Status.Outputs.Artifacts))
	assert.Equal(t, art.GlobalName, woc.wf.Status.Outputs.Artifacts[1].Name)
	assert.Equal(t, "new/new/key", woc.wf.Status.Outputs.Artifacts[1].S3.Key)
}

func TestParamSubstitutionWithArtifact(t *testing.T) {
	wf := test.LoadE2EWorkflow("functional/param-sub-with-artifacts.yaml")
	woc := newWoc(*wf)
	woc.operate()
	wf, err := woc.controller.wfclientset.ArgoprojV1alpha1().Workflows("").Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, wf.Status.Phase, wfv1.NodeRunning)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, len(pods.Items), 1)
}

func TestGlobalParamSubstitutionWithArtifact(t *testing.T) {
	wf := test.LoadE2EWorkflow("functional/global-param-sub-with-artifacts.yaml")
	woc := newWoc(*wf)
	woc.operate()
	wf, err := woc.controller.wfclientset.ArgoprojV1alpha1().Workflows("").Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, wf.Status.Phase, wfv1.NodeRunning)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, len(pods.Items), 1)
}

func TestExpandWithSequence(t *testing.T) {
	var seq wfv1.Sequence
	var items []wfv1.Item
	var err error

	seq = wfv1.Sequence{
		Count: "10",
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(items))
	assert.Equal(t, "0", items[0].Value.(string))
	assert.Equal(t, "9", items[9].Value.(string))

	seq = wfv1.Sequence{
		Start: "101",
		Count: "10",
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(items))
	assert.Equal(t, "101", items[0].Value.(string))
	assert.Equal(t, "110", items[9].Value.(string))

	seq = wfv1.Sequence{
		Start: "50",
		End:   "60",
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 11, len(items))
	assert.Equal(t, "50", items[0].Value.(string))
	assert.Equal(t, "60", items[10].Value.(string))

	seq = wfv1.Sequence{
		Start: "60",
		End:   "50",
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 11, len(items))
	assert.Equal(t, "60", items[0].Value.(string))
	assert.Equal(t, "50", items[10].Value.(string))

	seq = wfv1.Sequence{
		Count: "0",
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(items))

	seq = wfv1.Sequence{
		Start: "8",
		End:   "8",
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "8", items[0].Value.(string))

	seq = wfv1.Sequence{
		Format: "testuser%02X",
		Count:  "10",
		Start:  "1",
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(items))
	assert.Equal(t, "testuser01", items[0].Value.(string))
	assert.Equal(t, "testuser0A", items[9].Value.(string))
}

var metadataTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: metadata-template
  labels:
    image: foo:bar
  annotations:
    k8s-webhook-handler.io/repo: "git@github.com:argoproj/argo.git"
    k8s-webhook-handler.io/revision: 1e111caa1d2cc672b3b53c202b96a5f660a7e9b2
spec:
  entrypoint: foo
  templates:
    - name: foo
      container:
        image: "{{workflow.labels.image}}"
        env:
          - name: REPO
            value: "{{workflow.annotations.k8s-webhook-handler.io/repo}}"
          - name: REVISION
            value: "{{workflow.annotations.k8s-webhook-handler.io/revision}}"
        command: [sh, -c]
        args: ["echo hello world"]
`

func TestMetadataPassing(t *testing.T) {
	controller := newController()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(metadataTemplate)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	pods, err := controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.True(t, len(pods.Items) > 0, "pod was not created successfully")

	var (
		pod       = pods.Items[0]
		container = pod.Spec.Containers[1]
		foundRepo = false
		foundRev  = false
	)
	for _, ev := range container.Env {
		switch ev.Name {
		case "REPO":
			assert.Equal(t, "git@github.com:argoproj/argo.git", ev.Value)
			foundRepo = true
		case "REVISION":
			assert.Equal(t, "1e111caa1d2cc672b3b53c202b96a5f660a7e9b2", ev.Value)
			foundRev = true
		}
	}
	assert.True(t, foundRepo)
	assert.True(t, foundRev)
	assert.Equal(t, "foo:bar", container.Image)
}

var ioPathPlaceholders = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifact-path-placeholders-
spec:
  entrypoint: head-lines
  arguments:
    parameters:
    - name: lines-count
      value: 3
    artifacts:
    - name: text
      raw:
        data: |
          1
          2
          3
          4
          5
  templates:
  - name: head-lines
    inputs:
      parameters:
      - name: lines-count
      artifacts:
      - name: text
        path: /inputs/text/data
    outputs:
      parameters:
      - name: actual-lines-count
        valueFrom:
          path: /outputs/actual-lines-count/data
      artifacts:
      - name: text
        path: /outputs/text/data
    container:
      image: busybox
      command: [sh, -c, 'head -n {{inputs.parameters.lines-count}} <"{{inputs.artifacts.text.path}}" | tee "{{outputs.artifacts.text.path}}" | wc -l > "{{outputs.parameters.actual-lines-count.path}}"']
`

func TestResolveIOPathPlaceholders(t *testing.T) {
	wf := unmarshalWF(ioPathPlaceholders)
	woc := newWoc(*wf)
	woc.controller.Config.ArtifactRepository.S3 = new(S3ArtifactRepository)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.True(t, len(pods.Items) > 0, "pod was not created successfully")

	assert.Equal(t, []string{"sh", "-c", "head -n 3 <\"/inputs/text/data\" | tee \"/outputs/text/data\" | wc -l > \"/outputs/actual-lines-count/data\""}, pods.Items[0].Spec.Containers[1].Command)
}
