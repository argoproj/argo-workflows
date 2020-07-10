package controller

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo/config"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test"
	"github.com/argoproj/argo/workflow/common"
	hydratorfake "github.com/argoproj/argo/workflow/hydrator/fake"
	"github.com/argoproj/argo/workflow/util"
)

// TestOperateWorkflowPanicRecover ensures we can recover from unexpected panics
func TestOperateWorkflowPanicRecover(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fail()
		}
	}()
	cancel, controller := newController()
	defer cancel()
	// intentionally set clientset to nil to induce panic
	controller.kubeclientset = nil
	wf := unmarshalWF(helloWorldWf)
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("").Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
}

func Test_wfOperationCtx_reapplyUpdate(t *testing.T) {
	wf := &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
		Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{Name: "my-foo"}}},
	}
	cancel, controller := newController(wf)
	defer cancel()
	controller.hydrator = hydratorfake.Always
	woc := newWorkflowOperationCtx(wf, controller)

	// fake the behaviour woc.operate()
	assert.NoError(t, controller.hydrator.Hydrate(wf))
	nodes := wfv1.Nodes{"foo": wfv1.NodeStatus{Name: "my-foo", Phase: wfv1.NodeSucceeded}}

	// now force a re-apply update
	updatedWf, err := woc.reapplyUpdate(controller.wfclientset.ArgoprojV1alpha1().Workflows(""), nodes)
	if assert.NoError(t, err) && assert.NotNil(t, updatedWf) {
		assert.True(t, woc.controller.hydrator.IsHydrated(updatedWf))
		if assert.Contains(t, updatedWf.Status.Nodes, "foo") {
			assert.Equal(t, "my-foo", updatedWf.Status.Nodes["foo"].Name)
			assert.Equal(t, wfv1.NodeSucceeded, updatedWf.Status.Nodes["foo"].Phase, "phase is merged")
		}
	}
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

func TestGlobalParams(t *testing.T) {
	wf := unmarshalWF(helloWorldWf)
	cancel, controller := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	if assert.Contains(t, woc.globalParams, "workflow.creationTimestamp") {
		assert.NotContains(t, woc.globalParams["workflow.creationTimestamp"], "UTC")
	}
	assert.Contains(t, woc.globalParams, "workflow.duration")
	assert.Contains(t, woc.globalParams, "workflow.labels.workflows.argoproj.io/phase")
	assert.Contains(t, woc.globalParams, "workflow.name")
	assert.Contains(t, woc.globalParams, "workflow.namespace")
	assert.Contains(t, woc.globalParams, "workflow.parameters")
	assert.Contains(t, woc.globalParams, "workflow.serviceAccountName")
	assert.Contains(t, woc.globalParams, "workflow.uid")
}

// TestSidecarWithVolume verifies ia sidecar can have a volumeMount reference to both existing or volumeClaimTemplate volumes
func TestSidecarWithVolume(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(sidecarWithVol)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	pods, err := controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
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
	cancel, controller := newController()
	defer cancel()
	assert.NotNil(t, controller)
	wf := unmarshalWF(helloWorldWf)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	_, _, err := woc.loadExecutionSpec()
	assert.NoError(t, err)
	// Verify that there are no nodes in the wf status.
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.Template{}, "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	retryLimit := int32(2)
	retries.Limit = &retryLimit
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := 0; i < 2; i++ {
		childNode := fmt.Sprintf("child-node-%d", i)
		woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.Template{}, "", wfv1.NodeRunning)
		woc.addChildNode(nodeName, childNode)
	}

	n := woc.wf.GetNodeByName(nodeName)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodesWithRetries() should return false since
	// there should be no retries at this point.
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.NoError(t, err)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Mark lastChild as successful.
	woc.markNodePhase(lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.NoError(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, n.Phase, wfv1.NodeSucceeded)

	// Mark the parent node as running again and the lastChild as failed.
	woc.markNodePhase(n.Name, wfv1.NodeRunning)
	woc.markNodePhase(lastChild.Name, wfv1.NodeFailed)
	_, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.NoError(t, err)
	n = woc.wf.GetNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Add a third node that has failed.
	childNode := "child-node-3"
	woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.Template{}, "", wfv1.NodeFailed)
	woc.addChildNode(nodeName, childNode)
	n = woc.wf.GetNodeByName(nodeName)
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.NoError(t, err)
	assert.Equal(t, n.Phase, wfv1.NodeFailed)
}

// TestProcessNodesWithRetries tests retrying when RetryOn.Error is enabled
func TestProcessNodesWithRetriesOnErrors(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	assert.NotNil(t, controller)
	wf := unmarshalWF(helloWorldWf)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	_, _, err := woc.loadExecutionSpec()
	assert.NoError(t, err)
	// Verify that there are no nodes in the wf status.
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.Template{}, "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	retryLimit := int32(2)
	retries.Limit = &retryLimit
	retries.RetryPolicy = wfv1.RetryPolicyAlways
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := 0; i < 2; i++ {
		childNode := fmt.Sprintf("child-node-%d", i)
		woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.Template{}, "", wfv1.NodeRunning)
		woc.addChildNode(nodeName, childNode)
	}

	n := woc.wf.GetNodeByName(nodeName)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodesWithRetries() should return false since
	// there should be no retries at this point.
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Mark lastChild as successful.
	woc.markNodePhase(lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, n.Phase, wfv1.NodeSucceeded)

	// Mark the parent node as running again and the lastChild as errored.
	n = woc.markNodePhase(n.Name, wfv1.NodeRunning)
	woc.markNodePhase(lastChild.Name, wfv1.NodeError)
	_, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.NoError(t, err)
	n = woc.wf.GetNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Add a third node that has errored.
	childNode := "child-node-3"
	woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.Template{}, "", wfv1.NodeError)
	woc.addChildNode(nodeName, childNode)
	n = woc.wf.GetNodeByName(nodeName)
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	assert.Equal(t, n.Phase, wfv1.NodeError)
}

func TestProcessNodesWithRetriesWithBackoff(t *testing.T) {
	cancel, controller := newController()
	defer cancel()

	assert.NotNil(t, controller)
	wf := unmarshalWF(helloWorldWf)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	_, _, err := woc.loadExecutionSpec()
	assert.NoError(t, err)
	// Verify that there are no nodes in the wf status.
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.Template{}, "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	retryLimit := int32(2)
	retries.Limit = &retryLimit
	retries.Backoff = &wfv1.Backoff{
		Duration:    "10s",
		Factor:      2,
		MaxDuration: "10m",
	}
	retries.RetryPolicy = wfv1.RetryPolicyAlways
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	woc.initializeNode("child-node-1", wfv1.NodeTypePod, "", &wfv1.Template{}, "", wfv1.NodeRunning)
	woc.addChildNode(nodeName, "child-node-1")

	n := woc.wf.GetNodeByName(nodeName)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodesWithRetries() should return false since
	// there should be no retries at this point.
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Mark lastChild as successful.
	woc.markNodePhase(lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, n.Phase, wfv1.NodeSucceeded)
}

// TestProcessNodesWithRetries tests retrying when RetryOn.Error is disabled
func TestProcessNodesNoRetryWithError(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	assert.NotNil(t, controller)
	wf := unmarshalWF(helloWorldWf)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	_, _, err := woc.loadExecutionSpec()
	assert.NoError(t, err)
	// Verify that there are no nodes in the wf status.
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.Template{}, "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	retryLimit := int32(2)
	retries.Limit = &retryLimit
	retries.RetryPolicy = wfv1.RetryPolicyOnFailure
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := 0; i < 2; i++ {
		childNode := fmt.Sprintf("child-node-%d", i)
		woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.Template{}, "", wfv1.NodeRunning)
		woc.addChildNode(nodeName, childNode)
	}

	n := woc.wf.GetNodeByName(nodeName)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodesWithRetries() should return false since
	// there should be no retries at this point.
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Mark lastChild as successful.
	woc.markNodePhase(lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, n.Phase, wfv1.NodeSucceeded)

	// Mark the parent node as running again and the lastChild as errored.
	// Parent node should also be errored because retry on error is disabled
	n = woc.markNodePhase(n.Name, wfv1.NodeRunning)
	woc.markNodePhase(lastChild.Name, wfv1.NodeError)
	_, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.NoError(t, err)
	n = woc.wf.GetNodeByName(nodeName)
	assert.Equal(t, wfv1.NodeError, n.Phase)
}

var backoffMessage = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-05-05T15:18:40Z"
  generateName: retry-backoff-
  generation: 21
  labels:
    workflows.argoproj.io/completed: "true"
    workflows.argoproj.io/phase: Failed
  name: retry-backoff-s69z6
  namespace: argo
  resourceVersion: "348670"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/workflows/retry-backoff-s69z6
  uid: 110dbef4-c54b-4963-9739-03e9878810d9
spec:
  arguments: {}
  entrypoint: retry-backoff
  templates:
  - arguments: {}
    container:
      args:
      - import random; import sys; exit_code = random.choice([1, 1]); sys.exit(exit_code)
      command:
      - python
      - -c
      image: python:alpine3.6
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: retry-backoff
    outputs: {}
    retryStrategy:
      backoff:
        duration: "1"
        factor: 2
        maxDuration: 1m
      limit: 10
status:
  nodes:
    retry-backoff-s69z6:
      children:
      - retry-backoff-s69z6-1807967148
      - retry-backoff-s69z6-130058153
      displayName: retry-backoff-s69z6
      id: retry-backoff-s69z6
      name: retry-backoff-s69z6
      phase: Running
      startedAt: "2020-05-05T15:18:40Z"
      templateName: retry-backoff
      templateScope: local/retry-backoff-s69z6
      type: Retry
    retry-backoff-s69z6-130058153:
      displayName: retry-backoff-s69z6(1)
      finishedAt: "2020-05-05T15:18:43Z"
      hostNodeName: minikube
      id: retry-backoff-s69z6-130058153
      message: failed with exit code 1
      name: retry-backoff-s69z6(1)
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
            key: retry-backoff-s69z6/retry-backoff-s69z6-130058153/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2020-05-05T15:18:45Z"
      templateName: retry-backoff
      templateScope: local/retry-backoff-s69z6
      type: Pod
    retry-backoff-s69z6-1807967148:
      displayName: retry-backoff-s69z6(0)
      finishedAt: "2020-05-05T15:18:43Z"
      hostNodeName: minikube
      id: retry-backoff-s69z6-1807967148
      message: failed with exit code 1
      name: retry-backoff-s69z6(0)
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
            key: retry-backoff-s69z6/retry-backoff-s69z6-1807967148/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 2
        memory: 0
      startedAt: "2020-05-05T15:18:40Z"
      templateName: retry-backoff
      templateScope: local/retry-backoff-s69z6
      type: Pod
  phase: Running
  resourcesDuration:
    cpu: 5
    memory: 0
  startedAt: "2020-05-05T15:18:40Z"
`

func TestBackoffMessage(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	assert.NotNil(t, controller)
	wf := unmarshalWF(backoffMessage)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	_, _, err := woc.loadExecutionSpec()
	assert.NoError(t, err)
	retryNode := woc.wf.GetNodeByName("retry-backoff-s69z6")

	// Simulate backoff of 4 secods
	firstNode := getChildNodeIndex(retryNode, woc.wf.Status.Nodes, 0)
	firstNode.StartedAt = metav1.Time{Time: time.Now().Add(-8 * time.Second)}
	firstNode.FinishedAt = metav1.Time{Time: time.Now().Add(-6 * time.Second)}
	woc.wf.Status.Nodes[firstNode.ID] = *firstNode
	lastNode := getChildNodeIndex(retryNode, woc.wf.Status.Nodes, -1)
	lastNode.StartedAt = metav1.Time{Time: time.Now().Add(-3 * time.Second)}
	lastNode.FinishedAt = metav1.Time{Time: time.Now().Add(-1 * time.Second)}
	woc.wf.Status.Nodes[lastNode.ID] = *lastNode

	newRetryNode, proceed, err := woc.processNodeRetries(retryNode, *woc.wf.Spec.Templates[0].RetryStrategy, &executeTemplateOpts{})
	assert.NoError(t, err)
	assert.False(t, proceed)
	assert.Equal(t, "Backoff for 4 seconds", newRetryNode.Message)

	// Advance time one second
	firstNode.StartedAt = metav1.Time{Time: time.Now().Add(-9 * time.Second)}
	firstNode.FinishedAt = metav1.Time{Time: time.Now().Add(-7 * time.Second)}
	woc.wf.Status.Nodes[firstNode.ID] = *firstNode
	lastNode.StartedAt = metav1.Time{Time: time.Now().Add(-4 * time.Second)}
	lastNode.FinishedAt = metav1.Time{Time: time.Now().Add(-2 * time.Second)}
	woc.wf.Status.Nodes[lastNode.ID] = *lastNode

	newRetryNode, proceed, err = woc.processNodeRetries(retryNode, *woc.wf.Spec.Templates[0].RetryStrategy, &executeTemplateOpts{})
	assert.NoError(t, err)
	assert.False(t, proceed)
	// Message should not change
	assert.Equal(t, "Backoff for 4 seconds", newRetryNode.Message)

	// Advance time 3 seconds
	firstNode.StartedAt = metav1.Time{Time: time.Now().Add(-12 * time.Second)}
	firstNode.FinishedAt = metav1.Time{Time: time.Now().Add(-10 * time.Second)}
	woc.wf.Status.Nodes[firstNode.ID] = *firstNode
	lastNode.StartedAt = metav1.Time{Time: time.Now().Add(-7 * time.Second)}
	lastNode.FinishedAt = metav1.Time{Time: time.Now().Add(-5 * time.Second)}
	woc.wf.Status.Nodes[lastNode.ID] = *lastNode

	newRetryNode, proceed, err = woc.processNodeRetries(retryNode, *woc.wf.Spec.Templates[0].RetryStrategy, &executeTemplateOpts{})
	assert.NoError(t, err)
	assert.True(t, proceed)
	// New node is started, message should be clear
	assert.Equal(t, "", newRetryNode.Message)
}

var retriesVariableTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: whalesay
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    retryStrategy:
      limit: 10
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay {{retries}}"]
`

func TestRetriesVariable(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(retriesVariableTemplate)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	iterations := 5
	for i := 1; i <= iterations; i++ {
		if i != 1 {
			makePodsPhase(t, apiv1.PodFailed, controller.kubeclientset, wf.ObjectMeta.Namespace)
			wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
			assert.NoError(t, err)
		}
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate()
	}

	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if assert.NoError(t, err) && assert.Len(t, pods.Items, iterations) {
		for i := 0; i < iterations; i++ {
			assert.Equal(t, fmt.Sprintf("cowsay %d", i), pods.Items[i].Spec.Containers[1].Args[0])
		}
	}
}

var stepsRetriesVariableTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: whalesay
spec:
  entrypoint: step-retry
  templates:
  - name: step-retry
    retryStrategy:
      limit: 10
    steps:
      - - name: whalesay-success
          arguments:
            parameters:
            - name: retries
              value: "{{retries}}"
          template: whalesay

  - name: whalesay
    inputs:
      parameters:
        - name: retries
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay {{inputs.parameters.retries}}"]
`

func TestStepsRetriesVariable(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(stepsRetriesVariableTemplate)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)

	iterations := 5
	for i := 1; i <= iterations; i++ {
		if i != 1 {
			makePodsPhase(t, apiv1.PodFailed, controller.kubeclientset, wf.ObjectMeta.Namespace)
			wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
			assert.Nil(t, err)
		}
		// move to next retry step
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate()
	}

	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, iterations, len(pods.Items))
	for i := 0; i < iterations; i++ {
		assert.Equal(t, fmt.Sprintf("cowsay %d", i), pods.Items[i].Spec.Containers[1].Args[0])
	}
}

func TestAssessNodeStatus(t *testing.T) {
	daemoned := true
	tests := []struct {
		name string
		pod  *apiv1.Pod
		node *wfv1.NodeStatus
		want wfv1.NodePhase
	}{{
		name: "pod pending",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodPending,
			},
		},
		node: &wfv1.NodeStatus{},
		want: wfv1.NodePending,
	}, {
		name: "pod succeeded",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodSucceeded,
			},
		},
		node: &wfv1.NodeStatus{},
		want: wfv1.NodeSucceeded,
	}, {
		name: "pod failed - daemoned",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodFailed,
			},
		},
		node: &wfv1.NodeStatus{Daemoned: &daemoned},
		want: wfv1.NodeSucceeded,
	}, {
		name: "pod failed - not daemoned",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Message: "failed for some reason",
				Phase:   apiv1.PodFailed,
			},
		},
		node: &wfv1.NodeStatus{},
		want: wfv1.NodeFailed,
	}, {
		name: "pod deleted during operation",
		pod: &apiv1.Pod{
			ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &metav1.Time{Time: time.Now()}},
			Status: apiv1.PodStatus{
				Phase: apiv1.PodRunning,
			},
		},
		node: &wfv1.NodeStatus{},
		want: wfv1.NodeError,
	}, {
		name: "pod running",
		pod: &apiv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					common.AnnotationKeyTemplate: "{}",
				},
			},
			Status: apiv1.PodStatus{
				Phase: apiv1.PodRunning,
			},
		},
		node: &wfv1.NodeStatus{},
		want: wfv1.NodeRunning,
	}, {
		name: "default",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodUnknown,
			},
		},
		node: &wfv1.NodeStatus{},
		want: wfv1.NodeError,
	}}

	wf := unmarshalWF(helloWorldWf)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cancel, controller := newController()
			defer cancel()
			woc := newWorkflowOperationCtx(wf, controller)
			got := woc.assessNodeStatus(tt.pod, tt.node)
			assert.Equal(t, tt.want, got.Phase)
		})
	}
}

var workflowStepRetry = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: step-retry
spec:
  entrypoint: step-retry
  templates:
  - name: step-retry
    retryStrategy:
      limit: 1
    steps:
      - - name: whalesay-success
          arguments:
            parameters:
            - name: message
              value: success
          template: whalesay
      - - name: whalesay-failure
          arguments:
            parameters:
            - name: message
              value: failure
          template: whalesay

  - name: whalesay
    inputs:
      parameters:
        - name: message
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay {{inputs.parameters.message}}"]
`

// TestWorkflowParallelismLimit verifies parallelism at a workflow level is honored.
func TestWorkflowStepRetry(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(workflowStepRetry)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(pods.Items))

	//complete the first pod
	makePodsPhase(t, apiv1.PodSucceeded, controller.kubeclientset, wf.ObjectMeta.Namespace)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()

	// fail the second pod
	makePodsPhase(t, apiv1.PodFailed, controller.kubeclientset, wf.ObjectMeta.Namespace)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 3, len(pods.Items))
	assert.Equal(t, "cowsay success", pods.Items[0].Spec.Containers[1].Args[0])
	assert.Equal(t, "cowsay failure", pods.Items[1].Spec.Containers[1].Args[0])

	//verify that after the cowsay failure pod failed, we are retrying cowsay success
	assert.Equal(t, "cowsay success", pods.Items[2].Spec.Containers[1].Args[0])

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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(workflowParallelismLimit)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pods.Items))
	// operate again and make sure we don't schedule any more pods
	makePodsPhase(t, apiv1.PodRunning, controller.kubeclientset, wf.ObjectMeta.Namespace)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	// wfBytes, _ := json.MarshalIndent(wf, "", "  ")
	// log.Printf("%s", wfBytes)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(stepsTemplateParallelismLimit)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pods.Items))

	// operate again and make sure we don't schedule any more pods
	makePodsPhase(t, apiv1.PodRunning, controller.kubeclientset, wf.ObjectMeta.Namespace)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	// wfBytes, _ := json.MarshalIndent(wf, "", "  ")
	// log.Printf("%s", wfBytes)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(dagTemplateParallelismLimit)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pods.Items))

	// operate again and make sure we don't schedule any more pods
	makePodsPhase(t, apiv1.PodRunning, controller.kubeclientset, wf.ObjectMeta.Namespace)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	// wfBytes, _ := json.MarshalIndent(wf, "", "  ")
	// log.Printf("%s", wfBytes)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(nestedParallelism)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 4, len(pods.Items))
}

// TestSidecarResourceLimits verifies resource limits on the sidecar can be set in the controller config
func TestSidecarResourceLimits(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
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
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pod, err := controller.kubeclientset.CoreV1().Pods("").Get("hello-world", metav1.GetOptions{})
	assert.NoError(t, err)
	var waitCtr *apiv1.Container
	for _, ctr := range pod.Spec.Containers {
		if ctr.Name == "wait" {
			waitCtr = &ctr
			break
		}
	}
	if assert.NotNil(t, waitCtr) && assert.NotNil(t, waitCtr.Resources) {
		assert.Len(t, waitCtr.Resources.Limits, 2)
		assert.Len(t, waitCtr.Resources.Requests, 2)
	}
}

// TestSuspendResume tests the suspend and resume feature
func TestSuspendResume(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(stepsTemplateParallelismLimit)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)

	// suspend the workflow
	err = util.SuspendWorkflow(wfcset, wf.ObjectMeta.Name)
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, *wf.Spec.Suspend)

	// operate should not result in no workflows being created since it is suspended
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// resume the workflow and operate again. two pods should be able to be scheduled
	err = util.ResumeWorkflow(wfcset, controller.hydrator, wf.ObjectMeta.Name, "")
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Nil(t, wf.Spec.Suspend)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pods.Items))
}

var suspendTemplateWithDeadline = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: suspend-template
spec:
  entrypoint: suspend
  activeDeadlineSeconds: 0
  templates:
  - name: suspend
    suspend: {}
`

func TestSuspendWithDeadline(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	wf := unmarshalWF(suspendTemplateWithDeadline)
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
	updatedWf, err := wfcset.Get(wf.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	found := false

	for _, node := range updatedWf.Status.Nodes {
		if node.Type == wfv1.NodeTypeSuspend {
			assert.Equal(t, node.Phase, wfv1.NodeFailed)
			assert.Contains(t, node.Message, "Step exceeded its deadline")
			found = true
		}
	}
	assert.True(t, found)

}

var sequence = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: sequence
spec:
  entrypoint: steps
  templates:
  - name: steps
    steps:
      - - name: step1
          template: echo
          arguments:
            parameters:
            - name: msg
              value: "{{item}}"
          withSequence:
            start: "100"
            end: "101"

  - name: echo
    inputs:
      parameters:
      - name: msg
    container:
      image: alpine:latest
      command: [echo, "{{inputs.parameters.msg}}"]
`

func TestSequence(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(sequence)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	updatedWf, err := wfcset.Get(wf.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	found100 := false
	found101 := false
	for _, node := range updatedWf.Status.Nodes {
		if node.DisplayName == "step1(0:100)" {
			assert.Equal(t, "100", node.Inputs.Parameters[0].Value.String())
			found100 = true
		} else if node.DisplayName == "step1(1:101)" {
			assert.Equal(t, "101", node.Inputs.Parameters[0].Value.String())
			found101 = true
		}
	}
	assert.Equal(t, true, found100)
	assert.Equal(t, true, found101)
}

var inputParametersAsJson = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: whalesay
spec:
  entrypoint: steps
  arguments:
    parameters:
    - name: parameter1
      value: value1
  templates:
  - name: steps
    inputs:
      parameters:
      - name: parameter1
      - name: parameter2
        value: template2
    steps:
      - - name: step1
          template: whalesay
          arguments:
            parameters:
            - name: json
              value: "Workflow: {{workflow.parameters}}. Template: {{inputs.parameters}}"

  - name: whalesay
    inputs:
      parameters:
      - name: json
    container:
      image: docker/whalesay:latest
      command: [cowsay]
`

func TestInputParametersAsJson(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(inputParametersAsJson)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	updatedWf, err := wfcset.Get(wf.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	found := false
	for _, node := range updatedWf.Status.Nodes {
		if node.Type == wfv1.NodeTypePod {
			expectedJson := `Workflow: [{"name":"parameter1","value":"value1"}]. Template: [{"name":"parameter1","value":"value1"},{"name":"parameter2","value":"template2"}]`
			assert.Equal(t, expectedJson, node.Inputs.Parameters[0].Value.String())
			found = true
		}
	}
	assert.Equal(t, true, found)
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	wf := unmarshalWF(expandWithItems)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	newSteps, err := woc.expandStep(wf.Spec.Templates[0].Steps[0].Steps[0])
	assert.NoError(t, err)
	assert.Equal(t, 5, len(newSteps))
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
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
            value: "{{item.os}} {{item.version}} JSON({{item}})"
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
      args: ["cowsay \"{{inputs.parameters.message}}\""]
`

func TestExpandWithItemsMap(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(expandWithItemsMap)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	newSteps, err := woc.expandStep(wf.Spec.Templates[0].Steps[0].Steps[0])
	assert.NoError(t, err)
	assert.Equal(t, 3, len(newSteps))
	assert.Equal(t, "debian 9.1 JSON({\"os\":\"debian\",\"version\":9.1})", newSteps[0].Arguments.Parameters[0].Value.String())
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
        arguments:
          parameters:
          - name: param1
            value: value1
    - - name: release
        template: whalesay

  - name: approve
    inputs:
      parameters:
      - name: param1
    suspend: {}

  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
`

func TestSuspendTemplate(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	wf := unmarshalWF(suspendTemplate)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// resume the workflow. verify resume workflow edits nodestatus correctly
	err = util.ResumeWorkflow(wfcset, controller.hydrator, wf.ObjectMeta.Name, "")
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.False(t, util.IsWorkflowSuspended(wf))

	// operate the workflow. it should reach the second step
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pods.Items))
}

func TestSuspendTemplateWithFailedResume(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	wf := unmarshalWF(suspendTemplate)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// resume the workflow. verify resume workflow edits nodestatus correctly
	err = util.StopWorkflow(wfcset, controller.hydrator, wf.ObjectMeta.Name, "inputs.parameters.param1.value=value1", "Step failed!")
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.False(t, util.IsWorkflowSuspended(wf))

	// operate the workflow. it should be failed and not reach the second step
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Phase)
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))
}

func TestSuspendTemplateWithFilteredResume(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	wf := unmarshalWF(suspendTemplate)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// resume the workflow, but with non-matching selector
	err = util.ResumeWorkflow(wfcset, controller.hydrator, wf.ObjectMeta.Name, "inputs.paramaters.param1.value=value2")
	assert.Error(t, err)

	// operate the workflow. nothing should have happened
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))
	assert.True(t, util.IsWorkflowSuspended(wf))

	// resume the workflow, but with matching selector
	err = util.ResumeWorkflow(wfcset, controller.hydrator, wf.ObjectMeta.Name, "inputs.parameters.param1.value=value1")
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.False(t, util.IsWorkflowSuspended(wf))

	// operate the workflow. it should reach the second step
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pods.Items))
}

var suspendResumeAfterTemplate = `
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
    suspend:
      duration: 3

  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]
`

func TestSuspendResumeAfterTemplate(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	wf := unmarshalWF(suspendResumeAfterTemplate)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// wait 4 seconds
	time.Sleep(4 * time.Second)

	// operate the workflow. it should reach the second step
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pods.Items))
}

func TestSuspendResumeAfterTemplateNoWait(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	wf := unmarshalWF(suspendResumeAfterTemplate)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// don't wait

	// operate the workflow. it should have not reached the second step since not enough time passed
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err = controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	wf := unmarshalWF(volumeWithParam)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	pod, err := controller.kubeclientset.CoreV1().Pods("").Get(wf.Name, metav1.GetOptions{})
	assert.NoError(t, err)
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
	testVal := intstr.Parse("test-value")
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
	assert.Equal(t, testVal.String(), woc.globalParams["workflow.outputs.parameters.global-param"])

	// Change the value and verify it is reflected in workflow outputs
	newValue := intstr.Parse("new-value")
	param.Value = &newValue
	woc.addParamToGlobalScope(param)
	assert.Equal(t, 1, len(woc.wf.Status.Outputs.Parameters))
	assert.Equal(t, param.GlobalName, woc.wf.Status.Outputs.Parameters[0].Name)
	assert.Equal(t, newValue, *woc.wf.Status.Outputs.Parameters[0].Value)
	assert.Equal(t, newValue.String(), woc.globalParams["workflow.outputs.parameters.global-param"])

	// Add a new global parameter
	param.GlobalName = "global-param2"
	woc.addParamToGlobalScope(param)
	assert.Equal(t, 2, len(woc.wf.Status.Outputs.Parameters))
	assert.Equal(t, param.GlobalName, woc.wf.Status.Outputs.Parameters[1].Name)
	assert.Equal(t, newValue, *woc.wf.Status.Outputs.Parameters[1].Value)
	assert.Equal(t, newValue.String(), woc.globalParams["workflow.outputs.parameters.global-param2"])

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
	assert.NoError(t, err)
	assert.Equal(t, wf.Status.Phase, wfv1.NodeRunning)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, len(pods.Items), 1)
}

func TestGlobalParamSubstitutionWithArtifact(t *testing.T) {
	wf := test.LoadE2EWorkflow("functional/global-param-sub-with-artifacts.yaml")
	woc := newWoc(*wf)
	woc.operate()
	wf, err := woc.controller.wfclientset.ArgoprojV1alpha1().Workflows("").Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wf.Status.Phase, wfv1.NodeRunning)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
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
	assert.Equal(t, "0", items[0].GetStrVal())
	assert.Equal(t, "9", items[9].GetStrVal())

	seq = wfv1.Sequence{
		Start: "101",
		Count: "10",
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(items))
	assert.Equal(t, "101", items[0].GetStrVal())
	assert.Equal(t, "110", items[9].GetStrVal())

	seq = wfv1.Sequence{
		Start: "50",
		End:   "60",
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 11, len(items))
	assert.Equal(t, "50", items[0].GetStrVal())
	assert.Equal(t, "60", items[10].GetStrVal())

	seq = wfv1.Sequence{
		Start: "60",
		End:   "50",
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 11, len(items))
	assert.Equal(t, "60", items[0].GetStrVal())
	assert.Equal(t, "50", items[10].GetStrVal())

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
	assert.Equal(t, "8", items[0].GetStrVal())

	seq = wfv1.Sequence{
		Format: "testuser%02X",
		Count:  "10",
		Start:  "1",
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(items))
	assert.Equal(t, "testuser01", items[0].GetStrVal())
	assert.Equal(t, "testuser0A", items[9].GetStrVal())
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
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(metadataTemplate)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	pods, err := controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
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
	woc.artifactRepository.S3 = new(config.S3ArtifactRepository)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.True(t, len(pods.Items) > 0, "pod was not created successfully")

	assert.Equal(t, []string{"sh", "-c", "head -n 3 <\"/inputs/text/data\" | tee \"/outputs/text/data\" | wc -l > \"/outputs/actual-lines-count/data\""}, pods.Items[0].Spec.Containers[1].Command)
}

var outputValuePlaceholders = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: output-value-placeholders-wf
spec:
  entrypoint: tell-pod-name
  templates:
  - name: tell-pod-name
    outputs:
      parameters:
      - name: pod-name
        value: "{{pod.name}}"
    container:
      image: busybox
`

func TestResolvePlaceholdersInOutputValues(t *testing.T) {
	wf := unmarshalWF(outputValuePlaceholders)
	woc := newWoc(*wf)
	woc.artifactRepository.S3 = new(config.S3ArtifactRepository)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.True(t, len(pods.Items) > 0, "pod was not created successfully")

	templateString := pods.Items[0].ObjectMeta.Annotations["workflows.argoproj.io/template"]
	var template wfv1.Template
	err = json.Unmarshal([]byte(templateString), &template)
	assert.NoError(t, err)
	parameterValue := template.Outputs.Parameters[0].Value
	assert.NotNil(t, parameterValue)
	assert.Equal(t, "output-value-placeholders-wf", parameterValue.String())
}

var podNameInRetries = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: output-value-placeholders-wf
spec:
  entrypoint: tell-pod-name
  templates:
  - name: tell-pod-name
    retryStrategy:
      limit: 2
    outputs:
      parameters:
      - name: pod-name
        value: "{{pod.name}}"
    container:
      image: busybox
`

func TestResolvePodNameInRetries(t *testing.T) {
	wf := unmarshalWF(podNameInRetries)
	woc := newWoc(*wf)
	woc.artifactRepository.S3 = new(config.S3ArtifactRepository)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.True(t, len(pods.Items) > 0, "pod was not created successfully")

	templateString := pods.Items[0].ObjectMeta.Annotations["workflows.argoproj.io/template"]
	var template wfv1.Template
	err = json.Unmarshal([]byte(templateString), &template)
	assert.NoError(t, err)
	parameterValue := template.Outputs.Parameters[0].Value
	assert.NotNil(t, parameterValue)
	assert.Equal(t, "output-value-placeholders-wf-3033990984", parameterValue.String())
}

var outputStatuses = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: scripts-bash-
spec:
  entrypoint: bash-script-example
  templates:
  - name: bash-script-example
    steps:
    - - name: first
        template: flakey-container
        continueOn:
          failed: true
    - - name: print
        template: print-message
        arguments:
          parameters:
          - name: message
            value: "{{steps.first.status}}"


  - name: flakey-container
    script:
      image: busybox
      command: [sh, -c]
      args: ["exit 0"]

  - name: print-message
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo result was: {{inputs.parameters.message}}"]
`

func TestResolveStatuses(t *testing.T) {

	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	wf := unmarshalWF(outputStatuses)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	jsonValue, err := json.Marshal(&wf.Spec.Templates[0])
	assert.NoError(t, err)

	assert.Contains(t, string(jsonValue), "{{steps.first.status}}")
	assert.NotContains(t, string(jsonValue), "{{steps.print.status}}")
}

var resourceTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: resource-template
spec:
  entrypoint: resource
  templates:
  - name: resource
    resource:
      action: create
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: resource-cm
`

func TestResourceTemplate(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	wf := unmarshalWF(resourceTemplate)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Phase)

	pod, err := controller.kubeclientset.CoreV1().Pods("").Get("resource-template", metav1.GetOptions{})
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	tmplStr := pod.Annotations[common.AnnotationKeyTemplate]
	tmpl := wfv1.Template{}
	err = yaml.Unmarshal([]byte(tmplStr), &tmpl)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	cm := apiv1.ConfigMap{}
	err = yaml.Unmarshal([]byte(tmpl.Resource.Manifest), &cm)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	assert.Equal(t, "resource-cm", cm.Name)
	assert.Empty(t, cm.ObjectMeta.OwnerReferences)
}

var resourceWithOwnerReferenceTemplate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: resource-with-ownerreference-template
spec:
  entrypoint: start
  templates:
  - name: start
    steps:
    - - name: resource-1
        template: resource-1
      - name: resource-2
        template: resource-2
      - name: resource-3
        template: resource-3
  - name: resource-1
    resource:
      action: create
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: resource-cm-1
          ownerReferences:
          - apiVersion: argoproj.io/v1alpha1
            blockOwnerDeletion: true
            kind: Workflow
            name: "manual-ref-name"
            uid: "manual-ref-uid"
  - name: resource-2
    resource:
      action: create
      setOwnerReference: true
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: resource-cm-2
  - name: resource-3
    resource:
      action: create
      setOwnerReference: true
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: resource-cm-3
          ownerReferences:
          - apiVersion: argoproj.io/v1alpha1
            blockOwnerDeletion: true
            kind: Workflow
            name: "manual-ref-name"
            uid: "manual-ref-uid"
`

func TestResourceWithOwnerReferenceTemplate(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	wf := unmarshalWF(resourceWithOwnerReferenceTemplate)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, wf.Status.Phase)

	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}

	objectMetas := map[string]metav1.ObjectMeta{}
	for _, pod := range pods.Items {
		tmplStr := pod.Annotations[common.AnnotationKeyTemplate]
		tmpl := wfv1.Template{}
		err = yaml.Unmarshal([]byte(tmplStr), &tmpl)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		cm := apiv1.ConfigMap{}
		err = yaml.Unmarshal([]byte(tmpl.Resource.Manifest), &cm)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		objectMetas[cm.Name] = cm.ObjectMeta
	}
	if assert.Equal(t, 1, len(objectMetas["resource-cm-1"].OwnerReferences)) {
		assert.Equal(t, "manual-ref-name", objectMetas["resource-cm-1"].OwnerReferences[0].Name)
	}
	if assert.Equal(t, 1, len(objectMetas["resource-cm-2"].OwnerReferences)) {
		assert.Equal(t, "resource-with-ownerreference-template", objectMetas["resource-cm-2"].OwnerReferences[0].Name)
	}
	if assert.Equal(t, 2, len(objectMetas["resource-cm-3"].OwnerReferences)) {
		assert.Equal(t, "manual-ref-name", objectMetas["resource-cm-3"].OwnerReferences[0].Name)
		assert.Equal(t, "resource-with-ownerreference-template", objectMetas["resource-cm-3"].OwnerReferences[1].Name)
	}
}

var artifactRepositoryRef = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifact-repo-config-ref-
  namespace: my-ns
spec:
  entrypoint: whalesay
  artifactRepositoryRef:
    key: minio
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      artifacts:
      - name: message
        path: /tmp/hello_world.txt
`

var artifactRepositoryConfigMapData = `
s3:
  bucket: my-bucket
  keyPrefix: prefix/in/bucket
  endpoint: my-minio-endpoint.default:9000
  insecure: true
  accessKeySecret:
    name: my-minio-cred
    key: accesskey
  secretKeySecret:
    name: my-minio-cred
    key: secretkey
`

func TestArtifactRepositoryRef(t *testing.T) {
	wf := unmarshalWF(artifactRepositoryRef)
	woc := newWoc(*wf)
	_, err := woc.controller.kubeclientset.CoreV1().ConfigMaps(wf.ObjectMeta.Namespace).Create(
		&apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "artifact-repositories",
			},
			Data: map[string]string{
				"minio": artifactRepositoryConfigMapData,
			},
		},
	)
	assert.NoError(t, err)
	woc.operate()
	assert.Equal(t, woc.artifactRepository.S3.Bucket, "my-bucket")
	assert.Equal(t, woc.artifactRepository.S3.Endpoint, "my-minio-endpoint.default:9000")
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.True(t, len(pods.Items) > 0, "pod was not created successfully")
}

var stepScriptTmpl = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: scripts-bash-
spec:
  entrypoint: bash-script-example
  templates:
  - name: bash-script-example
    steps:
    - - name: generate
        template: gen-random-int
    - - name: print
        template: print-message
        arguments:
          parameters:
          - name: message
            value: "{{steps.generate.outputs.result}}"

  - name: gen-random-int
    script:
      image: debian:9.4
      command: [bash]
      source: |
        cat /dev/urandom | od -N2 -An -i | awk -v f=1 -v r=100 '{printf "%i\n", f + r * $1 / 65536}'

  - name: print-message
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo result was: {{inputs.parameters.message}}"]
`

var dagScriptTmpl = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-target-
spec:
  entrypoint: dag-target
  arguments:
    parameters:
    - name: target
      value: E

  templates:
  - name: dag-target
    dag:
      tasks:
      - name: A
        template: echo
        arguments:
          parameters: [{name: message, value: A}]
      - name: B
        template: echo
        arguments:
          parameters: [{name: message, value: B}]
      - name: C
        dependencies: [A]
        template: echo
        arguments:
          parameters: [{name: message, value: "{{tasks.A.outputs.result}}"}]
  - name: echo
    script:
      image: debian:9.4
      command: [bash]
      source: |
        cat /dev/urandom | od -N2 -An -i | awk -v f=1 -v r=100 '{printf "%i\n", f + r * $1 / 65536}'`

func TestStepWFGetNodeName(t *testing.T) {

	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	wf := unmarshalWF(stepScriptTmpl)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	assert.True(t, hasOutputResultRef("generate", &wf.Spec.Templates[0]))
	assert.False(t, hasOutputResultRef("print-message", &wf.Spec.Templates[0]))
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, "generate") {
			assert.True(t, getStepOrDAGTaskName(node.Name) == "generate")
		} else if strings.Contains(node.Name, "print-message") {
			assert.True(t, getStepOrDAGTaskName(node.Name) == "print-message")
		}
	}
}

func TestDAGWFGetNodeName(t *testing.T) {

	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	wf := unmarshalWF(dagScriptTmpl)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	assert.True(t, hasOutputResultRef("A", &wf.Spec.Templates[0]))
	assert.False(t, hasOutputResultRef("B", &wf.Spec.Templates[0]))
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, ".A") {
			assert.True(t, getStepOrDAGTaskName(node.Name) == "A")
		}
		if strings.Contains(node.Name, ".B") {
			assert.True(t, getStepOrDAGTaskName(node.Name) == "B")
		}
	}
}

var withParamAsJsonList = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: expand-with-items
spec:
  entrypoint: expand-with-items
  arguments:
    parameters:
    - name: input
      value: '[[1,2],[3,4],[4,5],[6,7]]'
  templates:
  - name: expand-with-items
    steps:
    - - name: whalesay
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "{{item}}"
        withParam: "{{workflow.parameters.input}}"
  - name: whalesay
    inputs:
      parameters:
      - name: message
    script:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo result was: {{inputs.parameters.message}}"]
`

func TestWithParamAsJsonList(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	wf := unmarshalWF(withParamAsJsonList)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pods, err := controller.kubeclientset.CoreV1().Pods("").List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 4, len(pods.Items))
}

var stepsOnExit = `
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
        onExit: exitContainer
        template: whalesay

  - name: whalesay
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["hello world"]

  - name: exitContainer
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["goodbye world"]
`

func TestStepsOnExit(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	wf := unmarshalWF(stepsOnExit)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	woc.operate()

	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	onExitNodeIsPresent := false
	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

var onExitFailures = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: exit-handlers
spec:
  entrypoint: intentional-fail
  onExit: exit-handler
  templates:
  - name: intentional-fail
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo intentional failure; exit 1"]
  - name: exit-handler
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo send e-mail: {{workflow.name}} {{workflow.status}}. Failed steps {{workflow.failures}}"]
`

func TestStepsOnExitFailures(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	wf := unmarshalWF(onExitFailures)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	woc.operate()

	assert.Contains(t, woc.globalParams[common.GlobalVarWorkflowFailures], `[{\"displayName\":\"exit-handlers\",\"message\":\"Unexpected pod phase for exit-handlers: \",\"templateName\":\"intentional-fail\",\"phase\":\"Error\",\"podName\":\"exit-handlers\"`)
}

var onExitTimeout = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: exit-handlers
spec:
  entrypoint: intentional-fail
  activeDeadlineSeconds: 0
  onExit: exit-handler
  templates:
  - name: intentional-fail
    suspend: {}
  - name: exit-handler
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo send e-mail: {{workflow.name}} {{workflow.status}}."]
`

func TestStepsOnExitTimeout(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	wf := unmarshalWF(onExitTimeout)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	woc.operate()

	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	onExitNodeIsPresent := false
	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, "onExit") && node.Phase == wfv1.NodePending {
			onExitNodeIsPresent = true
			break
		}
	}
	assert.True(t, onExitNodeIsPresent)
}

var invalidSpec = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: invalid-spec
spec:
  entrypoint: 123
`

func TestEventInvalidSpec(t *testing.T) {
	// Test whether a WorkflowFailed event is emitted in case of invalid spec
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(invalidSpec)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	events := getEvents(controller, 2)
	assert.Equal(t, "Normal WorkflowRunning Workflow Running", events[0])
	assert.Equal(t, "Warning WorkflowFailed invalid spec: template name '123' undefined", events[1])
}

func getEvents(controller *WorkflowController, num int) []string {
	c := controller.eventRecorderManager.(*testEventRecorderManager).eventRecorder.Events
	events := make([]string, num)
	for i := 0; i < num; i++ {
		events[i] = <-c
	}
	return events
}

var failLoadArtifactRepoCm = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifact-repo-config-ref-
spec:
  entrypoint: whalesay
  artifactRepositoryRef:
    configMap: artifact-repository
    key: config
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      artifacts:
      - name: message
        path: /tmp/hello_world.txt
`

func TestEventFailArtifactRepoCm(t *testing.T) {
	// Test whether a WorkflowFailed event is emitted in case of failure in loading artifact repository config map
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(failLoadArtifactRepoCm)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	events := getEvents(controller, 2)
	assert.Equal(t, "Normal WorkflowRunning Workflow Running", events[0])
	assert.Equal(t, "Warning WorkflowFailed Failed to load artifact repository configMap: failed to find artifactory ref {,}/artifact-repository#config", events[1])
}

var pdbwf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: artifact-repo-config-ref
spec:
  entrypoint: whalesay
  poddisruptionbudget:
    minavailable: 100%
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello world | tee /tmp/hello_world.txt"]
`

func TestPDBCreation(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(pdbwf)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	pdb, _ := controller.kubeclientset.PolicyV1beta1().PodDisruptionBudgets("").Get(woc.wf.Name, metav1.GetOptions{})
	assert.NotNil(t, pdb)
	assert.Equal(t, pdb.Name, wf.Name)
	woc.markWorkflowSuccess()
	woc.operate()
	pdb, _ = controller.kubeclientset.PolicyV1beta1().PodDisruptionBudgets("").Get(woc.wf.Name, metav1.GetOptions{})
	assert.Nil(t, pdb)
}

func TestStatusConditions(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(pdbwf)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(t, len(woc.wf.Status.Conditions), 0)
	woc.markWorkflowSuccess()
	assert.Equal(t, woc.wf.Status.Conditions[0].Status, metav1.ConditionStatus("True"))
}

var nestedOptionalOutputArtifacts = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifact-passing-
spec:
  entrypoint: artifact-example
  templates:
  - name: artifact-example
    steps:
    - - name: skip-artifact-generation
        template: conditional-whalesay
        arguments:
          parameters:
          - name: proceed
            value: "false"

  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["sleep 1; cowsay hello world | tee /tmp/hello_world.txt"]
    outputs:
      artifacts:
      - name: hello-art
        path: /tmp/hello_world.txt

  - name: conditional-whalesay
    inputs:
      parameters:
      - name: proceed
    steps:
    - - name: whalesay
        template: whalesay
        when: "{{inputs.parameters.proceed}}"
    outputs:
      artifacts:
      - name: hello-art
        from: "{{steps.whalesay.outputs.artifacts.hello-art}}"
        optional: true
`

func TestNestedOptionalOutputArtifacts(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	wf := unmarshalWF(nestedOptionalOutputArtifacts)
	wf, err := wfcset.Create(wf)
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate()
	woc.operate()

	assert.Equal(t, wfv1.NodeSucceeded, woc.wf.Status.Phase)
}

//  TestPodSpecLogForFailedPods tests PodSpec logging configuration
func TestPodSpecLogForFailedPods(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	assert.NotNil(t, controller)
	controller.Config.PodSpecLogStrategy.FailedPod = true
	wf := unmarshalWF(helloWorldWf)
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	woc.operate()
	woc.operate()
	for _, node := range woc.wf.Status.Nodes {
		assert.True(t, woc.shouldPrintPodSpec(node))
	}

}

//  TestPodSpecLogForAllPods tests  PodSpec logging configuration
func TestPodSpecLogForAllPods(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	assert.NotNil(t, controller)
	controller.Config.PodSpecLogStrategy.AllPods = true
	wf := unmarshalWF(nestedOptionalOutputArtifacts)
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	woc.operate()
	woc.operate()
	for _, node := range woc.wf.Status.Nodes {
		assert.True(t, woc.shouldPrintPodSpec(node))
	}
}

var retryNodeOutputs = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: daemon-step-dvbnn
spec:
  arguments: {}
  entrypoint: daemon-example
  templates:
  - arguments: {}
    inputs: {}
    metadata: {}
    name: daemon-example
    outputs: {}
    steps:
    - - arguments: {}
        name: influx
        template: influxdb
  - arguments: {}
    container:
      image: influxdb:1.2
      name: ""
      readinessProbe:
        httpGet:
          path: /ping
          port: 8086
      resources: {}
    daemon: true
    inputs: {}
    metadata: {}
    name: influxdb
    outputs: {}
    retryStrategy:
      limit: 10
status:
  finishedAt: null
  nodes:
    daemon-step-dvbnn:
      children:
      - daemon-step-dvbnn-1159996203
      displayName: daemon-step-dvbnn
      finishedAt: "2020-04-02T16:29:24Z"
      id: daemon-step-dvbnn
      name: daemon-step-dvbnn
      outboundNodes:
      - daemon-step-dvbnn-2254877734
      phase: Succeeded
      startedAt: "2020-04-02T16:29:18Z"
      templateName: daemon-example
      type: Steps
    daemon-step-dvbnn-1159996203:
      boundaryID: daemon-step-dvbnn
      children:
      - daemon-step-dvbnn-3639466923
      displayName: '[0]'
      finishedAt: "2020-04-02T16:29:24Z"
      id: daemon-step-dvbnn-1159996203
      name: daemon-step-dvbnn[0]
      phase: Succeeded
      startedAt: "2020-04-02T16:29:18Z"
      templateName: daemon-example
      type: StepGroup
    daemon-step-dvbnn-2254877734:
      boundaryID: daemon-step-dvbnn
      daemoned: true
      displayName: influx(0)
      finishedAt: "2020-04-02T16:29:24Z"
      id: daemon-step-dvbnn-2254877734
      name: daemon-step-dvbnn[0].influx(0)
      phase: Running
      podIP: 172.17.0.8
      resourcesDuration:
        cpu: 10
        memory: 0
      startedAt: "2020-04-02T16:29:18Z"
      templateName: influxdb
      type: Pod
    daemon-step-dvbnn-3639466923:
      boundaryID: daemon-step-dvbnn
      children:
      - daemon-step-dvbnn-2254877734
      displayName: influx
      finishedAt: "2020-04-02T16:29:24Z"
      id: daemon-step-dvbnn-3639466923
      name: daemon-step-dvbnn[0].influx
      phase: Succeeded
      startedAt: "2020-04-02T16:29:18Z"
      templateName: influxdb
      type: Retry
  phase: Succeeded
  startedAt: "2020-04-02T16:29:18Z"

`

// This tests to see if the outputs of the last child node of a retry node are added correctly to the scope
func TestRetryNodeOutputs(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := unmarshalWF(retryNodeOutputs)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)
	wf, err = wfcset.Get(wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	retryNode := woc.wf.GetNodeByName("daemon-step-dvbnn[0].influx")
	assert.NotNil(t, retryNode)
	fmt.Println(retryNode)
	scope := &wfScope{
		scope: make(map[string]interface{}),
	}
	woc.buildLocalScope(scope, "steps.influx", retryNode)
	assert.Contains(t, scope.scope, "steps.influx.ip")
}

var containerOutputsResult = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: steps-
spec:
  entrypoint: hello-hello-hello
  templates:
  - name: hello-hello-hello
    steps:
    - - name: hello1
        template: whalesay
        arguments:
          parameters: [{name: message, value: "hello1"}]
    - - name: hello2
        template: whalesay
        arguments:
          parameters: [{name: message, value: "{{steps.hello1.outputs.result}}"}]

  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:latest
      command: [echo]
      args: ["{{pod.name}}: {{inputs.parameters.message}}"]
`

func TestContainerOutputsResult(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	wf := unmarshalWF(containerOutputsResult)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)

	assert.True(t, hasOutputResultRef("hello1", &wf.Spec.Templates[0]))
	assert.False(t, hasOutputResultRef("hello2", &wf.Spec.Templates[0]))

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()

	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, "hello1") {
			assert.True(t, getStepOrDAGTaskName(node.Name) == "hello1")
		} else if strings.Contains(node.Name, "hello2") {
			assert.True(t, getStepOrDAGTaskName(node.Name) == "hello2")
		}
	}
}

var nestedStepGroupGlobalParams = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: global-outputs-bg7gl
spec:
  arguments: {}
  entrypoint: generate-globals
  templates:
  - arguments: {}
    inputs: {}
    metadata: {}
    name: generate-globals
    outputs: {}
    steps:
    - - arguments: {}
        name: generate
        template: nested-global-output-generation
  - arguments: {}
    container:
      args:
      - sleep 1; echo -n hello world > /tmp/hello_world.txt
      command:
      - sh
      - -c
      image: alpine:3.7
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: output-generation
    outputs:
      parameters:
      - name: hello-param
        valueFrom:
          path: /tmp/hello_world.txt
  - arguments: {}
    inputs: {}
    metadata: {}
    name: nested-global-output-generation
    outputs:
      parameters:
      - globalName: global-param
        name: hello-param
        valueFrom:
          parameter: '{{steps.generate-output.outputs.parameters.hello-param}}'
    steps:
    - - arguments: {}
        name: generate-output
        template: output-generation
status:
  conditions:
  - status: "True"
    type: Completed
  finishedAt: "2020-04-24T15:55:18Z"
  nodes:
    global-outputs-bg7gl:
      children:
      - global-outputs-bg7gl-1831647575
      displayName: global-outputs-bg7gl
      id: global-outputs-bg7gl
      name: global-outputs-bg7gl
      outboundNodes:
      - global-outputs-bg7gl-1290716463
      phase: Running
      startedAt: "2020-04-24T15:55:11Z"
      templateName: generate-globals
      templateScope: local/global-outputs-bg7gl
      type: Steps
    global-outputs-bg7gl-1290716463:
      boundaryID: global-outputs-bg7gl-2228002836
      displayName: generate-output
      finishedAt: "2020-04-24T15:55:16Z"
      hostNodeName: minikube
      id: global-outputs-bg7gl-1290716463
      name: global-outputs-bg7gl[0].generate[0].generate-output
      outputs:
        parameters:
        - name: hello-param
          value: hello world
          valueFrom:
            path: /tmp/hello_world.txt
      phase: Succeeded
      startedAt: "2020-04-24T15:55:11Z"
      templateName: output-generation
      templateScope: local/global-outputs-bg7gl
      type: Pod
    global-outputs-bg7gl-1831647575:
      boundaryID: global-outputs-bg7gl
      children:
      - global-outputs-bg7gl-2228002836
      displayName: '[0]'
      finishedAt: "2020-04-24T15:55:18Z"
      id: global-outputs-bg7gl-1831647575
      name: global-outputs-bg7gl[0]
      phase: Succeeded
      startedAt: "2020-04-24T15:55:11Z"
      templateName: generate-globals
      templateScope: local/global-outputs-bg7gl
      type: StepGroup
    global-outputs-bg7gl-2228002836:
      boundaryID: global-outputs-bg7gl
      children:
      - global-outputs-bg7gl-3089902334
      displayName: generate
      id: global-outputs-bg7gl-2228002836
      name: global-outputs-bg7gl[0].generate
      phase: Running
      outboundNodes:
      - global-outputs-bg7gl-1290716463
      startedAt: "2020-04-24T15:55:11Z"
      templateName: nested-global-output-generation
      templateScope: local/global-outputs-bg7gl
      type: Steps
    global-outputs-bg7gl-3089902334:
      boundaryID: global-outputs-bg7gl-2228002836
      children:
      - global-outputs-bg7gl-1290716463
      displayName: '[0]'
      finishedAt: "2020-04-24T15:55:18Z"
      id: global-outputs-bg7gl-3089902334
      name: global-outputs-bg7gl[0].generate[0]
      phase: Succeeded
      startedAt: "2020-04-24T15:55:11Z"
      templateName: nested-global-output-generation
      templateScope: local/global-outputs-bg7gl
      type: StepGroup
  startedAt: "2020-04-24T15:55:11Z"
`

func TestNestedStepGroupGlobalParams(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	wf := unmarshalWF(nestedStepGroupGlobalParams)
	wf, err := wfcset.Create(wf)
	assert.NoError(t, err)

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()

	node := woc.wf.Status.Nodes.FindByDisplayName("generate")
	if assert.NotNil(t, node) {
		assert.Equal(t, "hello-param", node.Outputs.Parameters[0].Name)
		assert.Equal(t, "global-param", node.Outputs.Parameters[0].GlobalName)
		assert.Equal(t, "hello world", node.Outputs.Parameters[0].Value.String())
	}

	assert.Equal(t, "hello world", woc.wf.Status.Outputs.Parameters[0].Value.String())
	assert.Equal(t, "global-param", woc.wf.Status.Outputs.Parameters[0].Name)
}

var globalVariablePlaceholders = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: output-value-global-variables-wf
  namespace: testNamespace
spec:
  serviceAccountName: testServiceAccountName
  entrypoint: tell-workflow-global-variables
  templates:
  - name: tell-workflow-global-variables
    outputs:
      parameters:
      - name: namespace
        value: "{{workflow.namespace}}"
      - name: serviceAccountName
        value: "{{workflow.serviceAccountName}}"
    container:
      image: busybox
`

func TestResolvePlaceholdersInGlobalVariables(t *testing.T) {
	wf := unmarshalWF(globalVariablePlaceholders)
	woc := newWoc(*wf)
	woc.artifactRepository.S3 = new(config.S3ArtifactRepository)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
	pods, err := woc.controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(metav1.ListOptions{})
	assert.NoError(t, err)
	assert.True(t, len(pods.Items) > 0, "pod was not created successfully")

	templateString := pods.Items[0].ObjectMeta.Annotations["workflows.argoproj.io/template"]
	var template wfv1.Template
	err = json.Unmarshal([]byte(templateString), &template)
	assert.NoError(t, err)
	namespaceValue := template.Outputs.Parameters[0].Value
	assert.NotNil(t, namespaceValue)
	assert.Equal(t, "testNamespace", namespaceValue.String())
	serviceAccountNameValue := template.Outputs.Parameters[1].Value
	assert.NotNil(t, serviceAccountNameValue)
	assert.Equal(t, "testServiceAccountName", serviceAccountNameValue.String())
}

var maxDurationOnErroredFirstNode = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2020-05-07T17:40:57Z"
  generateName: echo-
  generation: 4
  labels:
    workflows.argoproj.io/phase: Running
  name: echo-wngc4
  namespace: argo
  resourceVersion: "6339"
  selfLink: /apis/argoproj.io/v1alpha1/namespaces/argo/workflows/echo-wngc4
  uid: bed2749b-2971-4172-a61e-455ef02c4379
spec:
  arguments: {}
  entrypoint: echo
  templates:
  - arguments: {}
    container:
      args:
      - sleep 10 && exit 1
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
      retryPolicy: "Always"
      backoff:
        duration: "10"
        factor: 1
        maxDuration: 20m
      limit: 4
status:
  finishedAt: null
  nodes:
    echo-wngc4:
      children:
      - echo-wngc4-1641470511
      displayName: echo-wngc4
      finishedAt: null
      id: echo-wngc4
      name: echo-wngc4
      phase: Running
      startedAt: "2020-05-07T17:40:57Z"
      templateName: echo
      templateScope: local/echo-wngc4
      type: Retry
    echo-wngc4-1641470511:
      displayName: echo-wngc4(0)
      finishedAt: null
      hostNodeName: minikube
      id: echo-wngc4-1641470511
      name: echo-wngc4(0)
      phase: Error
      startedAt: "2020-05-07T17:40:57Z"
      templateName: echo
      templateScope: local/echo-wngc4
      type: Pod
  phase: Running
  startedAt: "2020-05-07T17:40:57Z"
`

// This tests that retryStrategy.backoff.maxDuration works correctly even if the first child node was deleted without a
// proper finishedTime tag.
func TestMaxDurationOnErroredFirstNode(t *testing.T) {
	wf := unmarshalWF(maxDurationOnErroredFirstNode)

	// Simulate node failed just now
	node := wf.Status.Nodes["echo-wngc4-1641470511"]
	node.StartedAt = metav1.Time{Time: time.Now().Add(-1 * time.Second)}
	wf.Status.Nodes["echo-wngc4-1641470511"] = node

	woc := newWoc(*wf)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
}

var backoffExceedsMaxDuration = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: echo-r6v49
spec:
  arguments: {}
  entrypoint: echo
  templates:
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
        duration: "120"
        factor: 1
        maxDuration: "60"
      limit: 4
status:
  nodes:
    echo-r6v49:
      children:
      - echo-r6v49-3721138751
      displayName: echo-r6v49
      id: echo-r6v49
      name: echo-r6v49
      phase: Running
      startedAt: "2020-05-07T18:10:34Z"
      templateName: echo
      templateScope: local/echo-r6v49
      type: Retry
    echo-r6v49-3721138751:
      displayName: echo-r6v49(0)
      finishedAt: "2020-05-07T18:10:35Z"
      hostNodeName: minikube
      id: echo-r6v49-3721138751
      message: failed with exit code 1
      name: echo-r6v49(0)
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
            key: echo-r6v49/echo-r6v49-3721138751/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "1"
      phase: Failed
      resourcesDuration:
        cpu: 1
        memory: 0
      startedAt: "2020-05-07T18:10:34Z"
      templateName: echo
      templateScope: local/echo-r6v49
      type: Pod
  phase: Running
  resourcesDuration:
    cpu: 1
    memory: 0
  startedAt: "2020-05-07T18:10:34Z"
`

// This tests that we don't wait a backoff if it would exceed the maxDuration anyway.
func TestBackoffExceedsMaxDuration(t *testing.T) {
	wf := unmarshalWF(backoffExceedsMaxDuration)

	// Simulate node failed just now
	node := wf.Status.Nodes["echo-r6v49-3721138751"]
	node.StartedAt = metav1.Time{Time: time.Now().Add(-1 * time.Second)}
	node.FinishedAt = metav1.Time{Time: time.Now()}
	wf.Status.Nodes["echo-r6v49-3721138751"] = node

	woc := newWoc(*wf)
	woc.operate()
	assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Phase)
	assert.Equal(t, "Backoff would exceed max duration limit", woc.wf.Status.Nodes["echo-r6v49"].Message)
	assert.Equal(t, "Backoff would exceed max duration limit", woc.wf.Status.Message)
}

var noOnExitWhenSkipped = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-primay-branch-sd6rg
spec:
  arguments: {}
  entrypoint: statis
  templates:
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
    name: pass
    outputs: {}
  - arguments: {}
    container:
      args:
      - exit
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: exit
    outputs: {}
  - arguments: {}
    dag:
      tasks:
      - arguments: {}
        name: A
        template: pass
      - arguments: {}
        dependencies:
        - A
        name: B
        onExit: exit
        template: pass
        when: '{{tasks.A.status}} != Succeeded'
      - arguments: {}
        dependencies:
        - A
        name: C
        template: pass
    inputs: {}
    metadata: {}
    name: statis
    outputs: {}
status:
  nodes:
    dag-primay-branch-sd6rg:
      children:
      - dag-primay-branch-sd6rg-1815625391
      displayName: dag-primay-branch-sd6rg
      id: dag-primay-branch-sd6rg
      name: dag-primay-branch-sd6rg
      outboundNodes:
      - dag-primay-branch-sd6rg-1832403010
      - dag-primay-branch-sd6rg-1849180629
      phase: Running
      startedAt: "2020-05-22T16:44:05Z"
      templateName: statis
      templateScope: local/dag-primay-branch-sd6rg
      type: DAG
    dag-primay-branch-sd6rg-1815625391:
      boundaryID: dag-primay-branch-sd6rg
      children:
      - dag-primay-branch-sd6rg-1832403010
      - dag-primay-branch-sd6rg-1849180629
      displayName: A
      finishedAt: "2020-05-22T16:44:09Z"
      hostNodeName: minikube
      id: dag-primay-branch-sd6rg-1815625391
      name: dag-primay-branch-sd6rg.A
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
            key: dag-primay-branch-sd6rg/dag-primay-branch-sd6rg-1815625391/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration:
        cpu: 3
        memory: 1
      startedAt: "2020-05-22T16:44:05Z"
      templateName: pass
      templateScope: local/dag-primay-branch-sd6rg
      type: Pod
    dag-primay-branch-sd6rg-1832403010:
      boundaryID: dag-primay-branch-sd6rg
      displayName: B
      finishedAt: "2020-05-22T16:44:10Z"
      id: dag-primay-branch-sd6rg-1832403010
      message: when 'Succeeded != Succeeded' evaluated false
      name: dag-primay-branch-sd6rg.B
      phase: Skipped
      startedAt: "2020-05-22T16:44:10Z"
      templateName: pass
      templateScope: local/dag-primay-branch-sd6rg
      type: Skipped
    dag-primay-branch-sd6rg-1849180629:
      boundaryID: dag-primay-branch-sd6rg
      displayName: C
      hostNodeName: minikube
      id: dag-primay-branch-sd6rg-1849180629
      name: dag-primay-branch-sd6rg.C
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
            key: dag-primay-branch-sd6rg/dag-primay-branch-sd6rg-1849180629/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "0"
      phase: Running
      resourcesDuration:
        cpu: 3
        memory: 1
      startedAt: "2020-05-22T16:44:10Z"
      templateName: pass
      templateScope: local/dag-primay-branch-sd6rg
      type: Pod
  phase: Running
  resourcesDuration:
    cpu: 10
    memory: 4
  startedAt: "2020-05-22T16:44:05Z"
`

// This tests that we don't wait a backoff if it would exceed the maxDuration anyway.
func TestNoOnExitWhenSkipped(t *testing.T) {
	wf := unmarshalWF(noOnExitWhenSkipped)

	woc := newWoc(*wf)
	woc.operate()
	assert.Nil(t, woc.wf.GetNodeByName("B.onExit"))
}

func TestGenerateNodeName(t *testing.T) {
	assert.Equal(t, "sleep(10:ten)", generateNodeName("sleep", 10, "ten"))
	item, err := wfv1.ParseItem(`[{"foo": "bar"}]`)
	assert.NoError(t, err)
	assert.Equal(t, `sleep(10:[{"foo":"bar"}])`, generateNodeName("sleep", 10, item))
	assert.NoError(t, err)
	item, err = wfv1.ParseItem("[10]")
	assert.NoError(t, err)
	assert.Equal(t, `sleep(10:[10])`, generateNodeName("sleep", 10, item))
}

// This tests that we don't wait a backoff if it would exceed the maxDuration anyway.
func TestPanicMetric(t *testing.T) {
	wf := unmarshalWF(noOnExitWhenSkipped)
	woc := newWoc(*wf)

	// This should make the call to "operate" panic
	woc.preExecutionNodePhases = nil
	woc.operate()

	metricsChan := make(chan prometheus.Metric)
	go func() {
		woc.controller.metrics.Collect(metricsChan)
		close(metricsChan)
	}()

	seen := false
	for {
		metric, ok := <-metricsChan
		if !ok {
			break
		}
		if strings.Contains(metric.Desc().String(), "OperationPanic") {
			seen = true
			var writtenMetric dto.Metric
			err := metric.Write(&writtenMetric)
			if assert.NoError(t, err) {
				assert.Equal(t, float64(1), *writtenMetric.Counter.Value)
			}
		}
	}
	assert.True(t, seen)
}

// Assert Workflows cannot be run without using workflowTemplateRef in reference mode
func TestControllerReferenceMode(t *testing.T) {
	wf := unmarshalWF(globalVariablePlaceholders)
	cancel, controller := newController()
	defer cancel()
	controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{}
	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingStrict
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(t, wfv1.NodeError, woc.wf.Status.Phase)
	assert.Equal(t, "workflows must use workflowTemplateRef to be executed when the controller is in reference mode", woc.wf.Status.Message)

	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingSecure
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(t, wfv1.NodeError, woc.wf.Status.Phase)
	assert.Equal(t, "workflows must use workflowTemplateRef to be executed when the controller is in reference mode", woc.wf.Status.Message)

	controller.Config.WorkflowRestrictions = nil
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
}

func TestValidReferenceMode(t *testing.T) {
	wf := test.LoadTestWorkflow("testdata/workflow-template-ref.yaml")
	wfTmpl := test.LoadTestWorkflowTemplate("testdata/workflow-template-submittable.yaml")
	cancel, controller := newController(wf, wfTmpl)
	defer cancel()
	controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{}
	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingSecure
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)

	// Change stored Workflow Spec
	woc.wf.Status.StoredWorkflowSpec.Entrypoint = "different"
	woc.operate()
	assert.Equal(t, wfv1.NodeError, woc.wf.Status.Phase)
	assert.Equal(t, "workflowTemplateRef reference may not change during execution when the controller is in reference mode", woc.wf.Status.Message)

	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingStrict
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)

	// Change stored Workflow Spec
	woc.wf.Status.StoredWorkflowSpec.Entrypoint = "different"
	woc.operate()
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Phase)
}

var workflowStatusMetric = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-to-completion-rngcr
spec:
  arguments: {}
  entrypoint: retry-to-completion
  metrics:
    prometheus:
    - counter:
        value: "1"
      gauge: null
      help: Count of step execution by result status
      histogram: null
      labels:
      - key: name
        value: retry
      - key: status
        value: '{{workflow.status}}'
      name: result_counter
      when: ""
  templates:
  - arguments: {}
    container:
      args:
      - import random; import sys; exit_code = random.choice(range(0, 5)); sys.exit(exit_code)
      command:
      - python
      - -c
      image: python
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: retry-to-completion
    outputs: {}
    retryStrategy: {}
status:
  nodes:
    retry-to-completion-rngcr:
      children:
      - retry-to-completion-rngcr-1856960714
      displayName: retry-to-completion-rngcr
      finishedAt: "2020-06-22T20:33:10Z"
      id: retry-to-completion-rngcr
      name: retry-to-completion-rngcr
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
            key: retry-to-completion-rngcr/retry-to-completion-rngcr-4003951493/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
      phase: Succeeded
      startedAt: "2020-06-22T20:32:15Z"
      templateName: retry-to-completion
      templateScope: local/retry-to-completion-rngcr
      type: Retry
    retry-to-completion-rngcr-1856960714:
      displayName: retry-to-completion-rngcr(0)
      finishedAt: "2020-06-22T20:32:25Z"
      hostNodeName: minikube
      id: retry-to-completion-rngcr-1856960714
      message: failed with exit code 3
      name: retry-to-completion-rngcr(0)
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
            key: retry-to-completion-rngcr/retry-to-completion-rngcr-1856960714/main.log
            secretKeySecret:
              key: secretkey
              name: my-minio-cred
        exitCode: "3"
      phase: Failed
      resourcesDuration:
        cpu: 10
        memory: 6
      startedAt: "2020-06-22T20:32:15Z"
      templateName: retry-to-completion
      templateScope: local/retry-to-completion-rngcr
      type: Pod
  phase: Running
  startedAt: "2020-06-22T20:32:15Z"
`

func TestWorkflowStatusMetric(t *testing.T) {
	wf := unmarshalWF(workflowStatusMetric)
	woc := newWoc(*wf)
	woc.operate()
	// Must only be one (completed: true)
	assert.Len(t, woc.wf.Status.Conditions, 1)
}

var propagate = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-backoff
spec:
  entrypoint: retry-backoff
  templates:
  - name: retry-backoff
    retryStrategy:
      limit: 10
      backoff:
        duration: "1"
        factor: 1
        maxDuration: "20"
    container:
      image: alpine
      command: [sh, -c]
      args: ["sleep $(( {{retries}} * 100 )); exit 1"]
`

func TestPropagateMaxDurationProcess(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	assert.NotNil(t, controller)
	wf := unmarshalWF(propagate)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	_, _, err := woc.loadExecutionSpec()
	assert.NoError(t, err)
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.Template{}, "", wfv1.NodeRunning)
	retryLimit := int32(2)
	retries := wfv1.RetryStrategy{
		Limit: &retryLimit,
		Backoff: &wfv1.Backoff{
			Duration:    "0",
			Factor:      1,
			MaxDuration: "20",
		},
	}
	woc.wf.Status.Nodes[woc.wf.NodeID(nodeName)] = *node

	childNode := fmt.Sprintf("child-node-%d", 0)
	woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.Template{}, "", wfv1.NodeFailed)
	woc.addChildNode(nodeName, childNode)

	var opts executeTemplateOpts
	n := woc.wf.GetNodeByName(nodeName)
	_, _, err = woc.processNodeRetries(n, retries, &opts)
	if assert.NoError(t, err) {
		assert.Equal(t, n.StartedAt.Add(20*time.Second).Round(time.Second).String(), opts.executionDeadline.Round(time.Second).String())
	}
}

var resubmitPendingWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: resubmit-pending-wf
  namespace: argo
spec:
  arguments: {}
  entrypoint: resubmit-pending
  templates:
  - arguments: {}
    inputs: {}
    metadata: {}
    name: resubmit-pending
    outputs: {}
    resubmitPendingPods: true
    script:
      command:
      - bash
      image: busybox
      name: ""
      resources:
        limits:
          cpu: "10"
      source: |
        sleep 5
status:
  finishedAt: null
  nodes:
    resubmit-pending-wf:
      displayName: resubmit-pending-wf
      finishedAt: null
      id: resubmit-pending-wf
      message: Pending 156.62ms
      name: resubmit-pending-wf
      phase: Pending
      startedAt: "2020-07-07T19:54:18Z"
      templateName: resubmit-pending
      templateScope: local/resubmit-pending-wf
      type: Pod
  phase: Running
  startedAt: "2020-07-07T19:54:18Z"
`

func TestCheckForbiddenErrorAndResbmitAllowed(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wf := unmarshalWF(resubmitPendingWf)
	woc := newWorkflowOperationCtx(wf, controller)

	forbiddenErr := apierr.NewForbidden(schema.GroupResource{Group: "test", Resource: "test1"}, "test", nil)
	nonForbiddenErr := apierr.NewBadRequest("badrequest")
	t.Run("ForbiddenError", func(t *testing.T) {
		node, err := woc.checkForbiddenErrorAndResbmitAllowed(forbiddenErr, "resubmit-pending-wf", &wf.Spec.Templates[0])
		assert.NotNil(t, node)
		assert.NoError(t, err)
		assert.Equal(t, wfv1.NodePending, node.Phase)
	})
	t.Run("NonForbiddenError", func(t *testing.T) {
		node, err := woc.checkForbiddenErrorAndResbmitAllowed(nonForbiddenErr, "resubmit-pending-wf", &wf.Spec.Templates[0])
		assert.Error(t, err)
		assert.Nil(t, node)
	})

}
