package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/argoproj/pkg/strftime"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	batchfake "k8s.io/client-go/kubernetes/typed/batch/v1/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v3/config"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	intstrutil "github.com/argoproj/argo-workflows/v3/util/intstr"
	"github.com/argoproj/argo-workflows/v3/util/template"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/cache"
	hydratorfake "github.com/argoproj/argo-workflows/v3/workflow/hydrator/fake"
	"github.com/argoproj/argo-workflows/v3/workflow/sync"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
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
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	ctx := context.Background()
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("").Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
}

func Test_wfOperationCtx_reapplyUpdate(t *testing.T) {
	ctx := context.Background()
	t.Run("Success", func(t *testing.T) {
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
		updatedWf, err := woc.reapplyUpdate(ctx, controller.wfclientset.ArgoprojV1alpha1().Workflows(""), nodes)
		if assert.NoError(t, err) && assert.NotNil(t, updatedWf) {
			assert.True(t, woc.controller.hydrator.IsHydrated(updatedWf))
			if assert.Contains(t, updatedWf.Status.Nodes, "foo") {
				assert.Equal(t, "my-foo", updatedWf.Status.Nodes["foo"].Name)
				assert.Equal(t, wfv1.NodeSucceeded, updatedWf.Status.Nodes["foo"].Phase, "phase is merged")
			}
		}
	})
	t.Run("ErrUpdatingCompletedWorkflow", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
			Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowError},
		}
		currWf := wf.DeepCopy()
		currWf.Status.Phase = wfv1.WorkflowSucceeded
		cancel, controller := newController(currWf)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		_, err := woc.reapplyUpdate(ctx, controller.wfclientset.ArgoprojV1alpha1().Workflows(""), wfv1.Nodes{})
		assert.EqualError(t, err, "must never update completed workflows")
	})
	t.Run("ErrUpdatingCompletedNode", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
			Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"my-node": wfv1.NodeStatus{Phase: wfv1.NodeError}}},
		}
		currWf := wf.DeepCopy()
		currWf.Status.Nodes = wfv1.Nodes{"my-node": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}}
		cancel, controller := newController(currWf)
		defer cancel()
		woc := newWorkflowOperationCtx(wf, controller)
		_, err := woc.reapplyUpdate(ctx, controller.wfclientset.ArgoprojV1alpha1().Workflows(""), wf.Status.Nodes)
		assert.EqualError(t, err, "must never update completed node my-node")
	})
}

func TestResourcesDuration(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf
  namespace: my-ns
spec:
  entrypoint: main
  templates:
   - name: main
     dag:
       tasks:
       - name: pod
         template: pod
   - name: pod
     container: 
       image: my-image
`)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.NotEmpty(t, woc.wf.Status.ResourcesDuration, "workflow duration not empty")
	assert.False(t, woc.wf.Status.Nodes.Any(func(node wfv1.NodeStatus) bool {
		return node.ResourcesDuration.IsZero()
	}), "zero node durations empty")
}

func TestGlobalParamDuration(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf
  namespace: my-ns
spec:
  entrypoint: main
  templates:
   - name: main
     dag:
       tasks:
       - name: pod
         template: pod
   - name: pod
     container: 
       image: my-image
`)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, woc.globalParams[common.GlobalVarWorkflowDuration], "0.000000")

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	assert.Greater(t, woc.globalParams[common.GlobalVarWorkflowDuration], "0.000000")
}

func TestEstimatedDuration(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf
  namespace: my-ns
  labels:
    workflows.argoproj.io/workflow-template: my-wftmpl
spec:
  entrypoint: main
  templates:
   - name: main
     dag:
       tasks:
       - name: pod
         template: pod
   - name: pod
     container: 
       image: my-image
`)
	cancel, controller := newController(wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-baseline-wf
  namespace: my-ns
status:
  startedAt: "1970-01-01T00:00:00Z"
  finishedAt: "1970-01-01T00:01:00Z"
  nodes:
    my-baseline-wf:
      startedAt: "1970-01-01T00:00:00Z"
      finishedAt: "1970-01-01T00:01:00Z"
`), wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.EstimatedDuration(1), woc.wf.Status.EstimatedDuration)
	assert.Equal(t, wfv1.EstimatedDuration(1), woc.wf.Status.Nodes[woc.wf.Name].EstimatedDuration)
	assert.Equal(t, wfv1.EstimatedDuration(1), woc.wf.Status.Nodes.FindByDisplayName("pod").EstimatedDuration)
}

func TestDefaultProgress(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf
  namespace: my-ns
spec:
  entrypoint: main
  templates:
   - name: main
     dag:
       tasks:
       - name: pod
         template: pod
   - name: pod
     container: 
       image: my-image
`)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Progress("1/1"), woc.wf.Status.Progress)
	assert.Equal(t, wfv1.Progress("1/1"), woc.wf.Status.Nodes[woc.wf.Name].Progress)
	assert.Equal(t, wfv1.Progress("1/1"), woc.wf.Status.Nodes.FindByDisplayName("pod").Progress)
}

func TestLoggedProgress(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf
  namespace: my-ns
spec:
  entrypoint: main
  templates:
   - name: main
     dag:
       tasks:
       - name: pod
         template: pod
   - name: pod
     container: 
       image: my-image
`)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	makePodsPhase(ctx, woc, apiv1.PodRunning, withProgress("50/100"))
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Progress("50/100"), woc.wf.Status.Progress)
	assert.Equal(t, wfv1.Progress("50/100"), woc.wf.Status.Nodes[woc.wf.Name].Progress)
	pod := woc.wf.Status.Nodes.FindByDisplayName("pod")
	assert.Equal(t, wfv1.Progress("50/100"), pod.Progress)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded, withProgress("100/100"))
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Progress("100/100"), woc.wf.Status.Progress)
	assert.Equal(t, wfv1.Progress("100/100"), woc.wf.Status.Nodes[woc.wf.Name].Progress)
	pod = woc.wf.Status.Nodes.FindByDisplayName("pod")
	assert.Equal(t, wfv1.Progress("100/100"), pod.Progress)
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
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	if assert.Contains(t, woc.globalParams, "workflow.creationTimestamp") {
		assert.NotContains(t, woc.globalParams["workflow.creationTimestamp"], "UTC")
	}
	for char := range strftime.FormatChars {
		assert.Contains(t, woc.globalParams, fmt.Sprintf("%s.%s", "workflow.creationTimestamp", string(char)))
	}
	assert.Contains(t, woc.globalParams, "workflow.creationTimestamp.s")
	assert.Contains(t, woc.globalParams, "workflow.creationTimestamp.RFC3339")

	assert.Contains(t, woc.globalParams, "workflow.duration")
	assert.Contains(t, woc.globalParams, "workflow.name")
	assert.Contains(t, woc.globalParams, "workflow.namespace")
	assert.Contains(t, woc.globalParams, "workflow.parameters")
	assert.Contains(t, woc.globalParams, "workflow.annotations")
	assert.Contains(t, woc.globalParams, "workflow.labels")
	assert.Contains(t, woc.globalParams, "workflow.serviceAccountName")
	assert.Contains(t, woc.globalParams, "workflow.uid")

	// Ensure that the phase label is included after the first operation
	woc.operate(ctx)
	assert.Contains(t, woc.globalParams, "workflow.labels.workflows.argoproj.io/phase")
}

// TestSidecarWithVolume verifies ia sidecar can have a volumeMount reference to both existing or volumeClaimTemplate volumes
func TestSidecarWithVolume(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(sidecarWithVol)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(woc)
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

func makeVolumeGcStrategyTemplate(strategy wfv1.VolumeClaimGCStrategy, phase wfv1.NodePhase) string {
	return fmt.Sprintf(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: workflow-with-volumes
spec:
  entrypoint: workflow-with-volumes
  volumeClaimGC:
    strategy: %s
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
  - name: workflow-with-volumes
    script:
      image: python:alpine3.6
      command: [python]
      volumeMounts:
      - name: claim-vol
        mountPath: /mnt/vol
      - name: existing-vol
        mountPath: /mnt/existing-vol
      source: |
        print("hello world")
status:
  phase: %s
  startedAt: 2020-08-01T15:32:09Z
  nodes:
    workflow-with-volumes:
      id: workflow-with-volumes
      name: workflow-with-volumes
      displayName: workflow-with-volumes
      type: Pod
      templateName: workflow-with-volumes
      templateScope: local/workflow-with-volumes
      startedAt: 2020-08-01T15:32:09Z
      phase: %s
  persistentVolumeClaims:
    - name: claim-vol
      persistentVolumeClaim:
        claimName: workflow-with-volumes-claim-vol
`, strategy, phase, phase)
}

func TestVolumeGCStrategy(t *testing.T) {
	tests := []struct {
		name                     string
		strategy                 wfv1.VolumeClaimGCStrategy
		phase                    wfv1.NodePhase
		expectedVolumesRemaining int
	}{{
		name:                     "failed/OnWorkflowCompletion",
		strategy:                 wfv1.VolumeClaimGCOnCompletion,
		phase:                    wfv1.NodeFailed,
		expectedVolumesRemaining: 0,
	}, {
		name:                     "failed/OnWorkflowSuccess",
		strategy:                 wfv1.VolumeClaimGCOnSuccess,
		phase:                    wfv1.NodeFailed,
		expectedVolumesRemaining: 1,
	}, {
		name:                     "succeeded/OnWorkflowSuccess",
		strategy:                 wfv1.VolumeClaimGCOnSuccess,
		phase:                    wfv1.NodeSucceeded,
		expectedVolumesRemaining: 0,
	}, {
		name:                     "succeeded/OnWorkflowCompletion",
		strategy:                 wfv1.VolumeClaimGCOnCompletion,
		phase:                    wfv1.NodeSucceeded,
		expectedVolumesRemaining: 0,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(makeVolumeGcStrategyTemplate(tt.strategy, tt.phase))
			cancel, controller := newController(wf)
			defer cancel()

			ctx := context.Background()
			wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
			woc := newWorkflowOperationCtx(wf, controller)
			woc.operate(ctx)
			wf, err := wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
			if assert.NoError(t, err) {
				assert.Len(t, wf.Status.PersistentVolumeClaims, tt.expectedVolumesRemaining)
			}
		})
	}
}

// TestProcessNodesWithRetries tests the processNodesWithRetries() method.
func TestProcessNodesWithRetries(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := 0; i < 2; i++ {
		childNode := fmt.Sprintf("child-node-%d", i)
		woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
		woc.addChildNode(nodeName, childNode)
	}

	n := woc.wf.GetNodeByName(nodeName)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodesWithRetries() should return false since
	// there should be no retries at this point.
	n, _, err := woc.processNodeRetries(n, retries, &executeTemplateOpts{})
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
	woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed)
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
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")
	retries.RetryPolicy = wfv1.RetryPolicyAlways
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := 0; i < 2; i++ {
		childNode := fmt.Sprintf("child-node-%d", i)
		woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
		woc.addChildNode(nodeName, childNode)
	}

	n := woc.wf.GetNodeByName(nodeName)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodesWithRetries() should return false since
	// there should be no retries at this point.
	n, _, err := woc.processNodeRetries(n, retries, &executeTemplateOpts{})
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
	woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeError)
	woc.addChildNode(nodeName, childNode)
	n = woc.wf.GetNodeByName(nodeName)
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	assert.Equal(t, n.Phase, wfv1.NodeError)
}

// TestProcessNodesWithRetries tests retrying when RetryOnTransientError is enabled
func TestProcessNodesWithRetriesOnTransientErrors(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")
	retries.RetryPolicy = wfv1.RetryPolicyOnTransientError
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := 0; i < 2; i++ {
		childNode := fmt.Sprintf("child-node-%d", i)
		woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
		woc.addChildNode(nodeName, childNode)
	}

	n := woc.wf.GetNodeByName(nodeName)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodesWithRetries() should return false since
	// there should be no retries at this point.
	n, _, err := woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Mark lastChild as successful.
	woc.markNodePhase(lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, n.Phase, wfv1.NodeSucceeded)

	// Mark the parent node as running again and the lastChild as errored with a message that indicates the error
	// is transient.
	n = woc.markNodePhase(n.Name, wfv1.NodeRunning)
	transientEnvVarKey := "TRANSIENT_ERROR_PATTERN"
	transientErrMsg := "This error is transient"
	woc.markNodePhase(lastChild.Name, wfv1.NodeError, transientErrMsg)
	_ = os.Setenv(transientEnvVarKey, transientErrMsg)
	_, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.NoError(t, err)
	n = woc.wf.GetNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)
	_ = os.Unsetenv(transientEnvVarKey)

	// Add a third node that has errored.
	childNode := "child-node-3"
	woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeError)
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
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")
	retries.Backoff = &wfv1.Backoff{
		Duration:    "10s",
		Factor:      intstrutil.ParsePtr("2"),
		MaxDuration: "10m",
	}
	retries.RetryPolicy = wfv1.RetryPolicyAlways
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	woc.initializeNode("child-node-1", wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
	woc.addChildNode(nodeName, "child-node-1")

	n := woc.wf.GetNodeByName(nodeName)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodesWithRetries() should return false since
	// there should be no retries at this point.
	n, _, err := woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Mark lastChild as successful.
	woc.markNodePhase(lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.Nil(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, n.Phase, wfv1.NodeSucceeded)
}

func TestProcessNodesWithRetriesWithExponentialBackoff(t *testing.T) {
	require := require.New(t)

	cancel, controller := newController()
	defer cancel()
	require.NotNil(controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	require.NotNil(wf)
	woc := newWorkflowOperationCtx(wf, controller)
	require.NotNil(woc)

	// Verify that there are no nodes in the wf status.
	require.Zero(len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")
	retries.RetryPolicy = wfv1.RetryPolicyAlways
	retries.Backoff = &wfv1.Backoff{
		Duration: "5m",
		Factor:   intstrutil.ParsePtr("2"),
	}
	woc.wf.Status.Nodes[nodeID] = *node

	require.Equal(wfv1.NodeRunning, node.Phase)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	require.Nil(lastChild)

	woc.initializeNode("child-node-1", wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed)
	woc.addChildNode(nodeName, "child-node-1")

	n := woc.wf.GetNodeByName(nodeName)

	// Last child has failed. processNodesWithRetries() should return false due to the default backoff.
	var err error
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	require.NoError(err)
	require.Equal(wfv1.NodeRunning, n.Phase)

	// First backoff should be between 295 and 300 seconds.
	backoff, err := parseRetryMessage(n.Message)
	require.NoError(err)
	require.LessOrEqual(backoff, 300)
	require.Less(295, backoff)

	woc.initializeNode("child-node-2", wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeError)
	woc.addChildNode(nodeName, "child-node-2")
	n = woc.wf.GetNodeByName(nodeName)

	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	require.NoError(err)
	require.Equal(wfv1.NodeRunning, n.Phase)

	// Second backoff should be between 595 and 600 seconds.
	backoff, err = parseRetryMessage(n.Message)
	require.NoError(err)
	require.LessOrEqual(backoff, 600)
	require.Less(595, backoff)

	// Mark lastChild as successful.
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	require.NotNil(lastChild)
	woc.markNodePhase(lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	require.NoError(err)
	// The parent node also gets marked as Succeeded.
	require.Equal(wfv1.NodeSucceeded, n.Phase)
}

func parseRetryMessage(message string) (int, error) {
	pattern := regexp.MustCompile(`Backoff for (\d+) minutes (\d+) seconds`)
	matches := pattern.FindStringSubmatch(message)
	if len(matches) != 3 {
		return 0, fmt.Errorf("unexpected message: %v", message)
	}

	minutes, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, err
	}

	totalSeconds := minutes*60 + seconds
	return totalSeconds, nil
}

// TestProcessNodesWithRetries tests retrying when RetryOn.Error is disabled
func TestProcessNodesNoRetryWithError(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")
	retries.RetryPolicy = wfv1.RetryPolicyOnFailure
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := 0; i < 2; i++ {
		childNode := fmt.Sprintf("child-node-%d", i)
		woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
		woc.addChildNode(nodeName, childNode)
	}

	n := woc.wf.GetNodeByName(nodeName)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodesWithRetries() should return false since
	// there should be no retries at this point.
	n, _, err := woc.processNodeRetries(n, retries, &executeTemplateOpts{})
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
  
  entrypoint: retry-backoff
  templates:
  - 
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
        duration: "2"
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
	wf := wfv1.MustUnmarshalWorkflow(backoffMessage)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
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
	wf := wfv1.MustUnmarshalWorkflow(retriesVariableTemplate)
	cancel, controller := newController(wf)
	defer cancel()
	ctx := context.Background()
	iterations := 5
	var woc *wfOperationCtx
	for i := 1; i <= iterations; i++ {
		woc = newWorkflowOperationCtx(wf, controller)
		if i != 1 {
			makePodsPhase(ctx, woc, apiv1.PodFailed)
		}
		woc.operate(ctx)
		wf = woc.wf
	}

	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, iterations)
	expected := []string{}
	actual := []string{}
	for i := 0; i < iterations; i++ {
		actual = append(actual, pods.Items[i].Spec.Containers[1].Args[0])
		expected = append(expected, fmt.Sprintf("cowsay %d", i))
	}
	// ordering not preserved
	assert.ElementsMatch(t, expected, actual)
}

var retriesVariableInPodSpecPatchTemplate = `
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
    podSpecPatch: |
      containers:
        - name: main
          resources:
            limits:
              memory: "{{=(sprig.int(retries) + 1) * 64}}Mi"
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hello"]
`

// TestRetriesVariableInPodSpecPatch makes sure that {{retries}} variable in pod spec patch is correctly
// updated before each retry
func TestRetriesVariableInPodSpecPatch(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(retriesVariableInPodSpecPatchTemplate)
	cancel, controller := newController(wf)
	defer cancel()
	ctx := context.Background()
	iterations := 5
	var woc *wfOperationCtx
	for i := 1; i <= iterations; i++ {
		woc = newWorkflowOperationCtx(wf, controller)
		if i != 1 {
			makePodsPhase(ctx, woc, apiv1.PodFailed)
		}
		woc.operate(ctx)
		wf = woc.wf
	}

	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, iterations)
	expected := []string{}
	actual := []string{}
	for i := 0; i < iterations; i++ {
		actual = append(actual, pods.Items[i].Spec.Containers[1].Resources.Limits.Memory().String())
		expected = append(expected, fmt.Sprintf("%dMi", (i+1)*64))
	}
	// expecting memory limit to increase after each retry: "64Mi", "128Mi", "192Mi", "256Mi", "320Mi"
	// ordering not preserved
	assert.ElementsMatch(t, actual, expected)
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
	wf := wfv1.MustUnmarshalWorkflow(stepsRetriesVariableTemplate)
	cancel, controller := newController(wf)
	defer cancel()
	ctx := context.Background()
	iterations := 5
	var woc *wfOperationCtx
	for i := 1; i <= iterations; i++ {
		woc = newWorkflowOperationCtx(wf, controller)
		if i != 1 {
			makePodsPhase(ctx, woc, apiv1.PodFailed)
		}
		// move to next retry step
		woc.operate(ctx)
		wf = woc.wf
	}

	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, iterations)

	expected := []string{}
	actual := []string{}
	for i := 0; i < iterations; i++ {
		actual = append(actual, pods.Items[i].Spec.Containers[1].Args[0])
		expected = append(expected, fmt.Sprintf("cowsay %d", i))
	}
	// ordering not preserved
	assert.ElementsMatch(t, expected, actual)
}

func TestAssessNodeStatus(t *testing.T) {
	const templateName = "whalesay"
	tests := []struct {
		name   string
		pod    *apiv1.Pod
		daemon bool
		node   *wfv1.NodeStatus
		want   wfv1.NodePhase
	}{{
		name: "pod pending",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodPending,
			},
		},
		node: &wfv1.NodeStatus{TemplateName: templateName},
		want: wfv1.NodePending,
	}, {
		name: "pod succeeded",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodSucceeded,
			},
		},
		node: &wfv1.NodeStatus{TemplateName: templateName},
		want: wfv1.NodeSucceeded,
	}, {
		name: "pod failed - daemoned",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodFailed,
			},
		},
		daemon: true,
		node:   &wfv1.NodeStatus{TemplateName: templateName},
		want:   wfv1.NodeSucceeded,
	}, {
		name: "daemon, pod running, node failed",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodRunning,
			},
		},
		daemon: true,
		node:   &wfv1.NodeStatus{TemplateName: templateName, Phase: wfv1.NodeFailed},
		want:   wfv1.NodeFailed,
	}, {
		name: "daemon, pod running, node succeeded",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodRunning,
			},
		},
		daemon: true,
		node:   &wfv1.NodeStatus{TemplateName: templateName, Phase: wfv1.NodeSucceeded},
		want:   wfv1.NodeSucceeded,
	}, {
		name: "pod failed - not daemoned",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Message: "failed for some reason",
				Phase:   apiv1.PodFailed,
			},
		},
		node: &wfv1.NodeStatus{TemplateName: templateName},
		want: wfv1.NodeFailed,
	}, {
		name: "pod running",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodRunning,
			},
		},
		node: &wfv1.NodeStatus{TemplateName: templateName},
		want: wfv1.NodeRunning,
	}, {
		name: "default",
		pod:  &apiv1.Pod{},
		node: &wfv1.NodeStatus{TemplateName: templateName},
		want: wfv1.NodeError,
	}}

	nonDaemonWf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	daemonWf := wfv1.MustUnmarshalWorkflow(helloDaemonWf)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := nonDaemonWf
			if tt.daemon {
				wf = daemonWf
			}
			assert.Equal(t, tt.daemon, wf.GetTemplateByName(tt.node.TemplateName).IsDaemon(), "check the test case is valid")
			cancel, controller := newController()
			defer cancel()
			woc := newWorkflowOperationCtx(wf, controller)
			got := woc.assessNodeStatus(tt.pod, tt.node)
			assert.Equal(t, tt.want, got.Phase)
		})
	}
}

func getPodTemplate(pod *apiv1.Pod) (*wfv1.Template, error) {
	tmpl := &wfv1.Template{}
	for _, c := range pod.Spec.Containers {
		for _, e := range c.Env {
			if e.Name == common.EnvVarTemplate {
				return tmpl, json.Unmarshal([]byte(e.Value), tmpl)
			}
		}
	}
	return nil, fmt.Errorf("not found")
}

func getPodDeadline(pod *apiv1.Pod) (time.Time, error) {
	for _, c := range pod.Spec.Containers {
		for _, e := range c.Env {
			if e.Name == common.EnvVarDeadline {
				return time.Parse(time.RFC3339, e.Value)
			}
		}
	}
	return time.Time{}, fmt.Errorf("not found")
}

func TestGetPodTemplate(t *testing.T) {
	tests := []struct {
		name string
		pod  *apiv1.Pod
		want *wfv1.Template
	}{{
		name: "missing template",
		pod: &apiv1.Pod{
			Spec: apiv1.PodSpec{
				Containers: []apiv1.Container{
					{
						Env: []apiv1.EnvVar{},
					},
				},
			},
		},
		want: nil,
	}, {
		name: "empty template",
		pod: &apiv1.Pod{
			Spec: apiv1.PodSpec{
				Containers: []apiv1.Container{
					{
						Env: []apiv1.EnvVar{
							{
								Name:  common.EnvVarTemplate,
								Value: "{}",
							},
						},
					},
				},
			},
		},
		want: &wfv1.Template{},
	}, {
		name: "simple template",
		pod: &apiv1.Pod{
			Spec: apiv1.PodSpec{
				Containers: []apiv1.Container{
					{
						Env: []apiv1.EnvVar{
							{
								Name:  common.EnvVarTemplate,
								Value: "{\"name\":\"argosay\"}",
							},
						},
					},
				},
			},
		},
		want: &wfv1.Template{
			Name: "argosay",
		},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := getPodTemplate(tt.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetPodDeadline(t *testing.T) {
	tests := []struct {
		name string
		pod  *apiv1.Pod
		want time.Time
	}{{
		name: "missing deadline",
		pod: &apiv1.Pod{
			Spec: apiv1.PodSpec{
				Containers: []apiv1.Container{
					{
						Env: []apiv1.EnvVar{},
					},
				},
			},
		},
		want: time.Time{},
	}, {
		name: "zero deadline",
		pod: &apiv1.Pod{
			Spec: apiv1.PodSpec{
				Containers: []apiv1.Container{
					{
						Env: []apiv1.EnvVar{
							{
								Name:  common.EnvVarDeadline,
								Value: "0001-01-01T00:00:00Z",
							},
						},
					},
				},
			},
		},
		want: time.Time{},
	}, {
		name: "a deadline",
		pod: &apiv1.Pod{
			Spec: apiv1.PodSpec{
				Containers: []apiv1.Container{
					{
						Env: []apiv1.EnvVar{
							{
								Name:  common.EnvVarDeadline,
								Value: "2021-01-21T01:02:03Z",
							},
						},
					},
				},
			},
		},
		want: time.Date(2021, time.Month(1), 21, 1, 2, 3, 0, time.UTC),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := getPodDeadline(tt.pod)
			assert.Equal(t, tt.want, got)
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
	ctx := context.Background()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow(workflowStepRetry)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.Nil(t, err)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(pods.Items))

	// complete the first pod
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	// fail the second pod
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err = listPods(woc)
	assert.Nil(t, err)
	if assert.Equal(t, 3, len(pods.Items)) {
		assert.Equal(t, "cowsay success", pods.Items[0].Spec.Containers[1].Args[0])
		assert.Equal(t, "cowsay failure", pods.Items[1].Spec.Containers[1].Args[0])

		// verify that after the cowsay failure pod failed, we are retrying cowsay success
		assert.Equal(t, "cowsay success", pods.Items[2].Spec.Containers[1].Args[0])
	}
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(workflowParallelismLimit)
	cancel, controller := newController(wf)
	defer cancel()

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 2)

	makePodsPhase(ctx, woc, apiv1.PodRunning)

	// operate again and make sure we don't schedule any more pods
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	pods, err = listPods(woc)
	assert.NoError(t, err)
	assert.Len(t, pods.Items, 2)
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
	wf := wfv1.MustUnmarshalWorkflow(stepsTemplateParallelismLimit)
	ctx := context.Background()
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)

	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pods.Items))

	// operate again and make sure we don't schedule any more pods
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	// wfBytes, _ := json.MarshalIndent(wf, "", "  ")
	// log.Printf("%s", wfBytes)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err = listPods(woc)
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
	wf := wfv1.MustUnmarshalWorkflow(dagTemplateParallelismLimit)
	cancel, controller := newController(wf)
	defer cancel()
	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pods.Items))

	// operate again and make sure we don't schedule any more pods
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	pods, err = listPods(woc)
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
	ctx := context.Background()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow(nestedParallelism)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("").Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pod, err := getPod(woc, "hello-world")
	assert.NoError(t, err)
	var waitCtr *apiv1.Container
	for _, ctr := range pod.Spec.Containers {
		ctr := ctr
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
	wf := wfv1.MustUnmarshalWorkflow(stepsTemplateParallelismLimit)
	cancel, controller := newController(wf)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// suspend the workflow
	ctx := context.Background()
	err := util.SuspendWorkflow(ctx, wfcset, wf.ObjectMeta.Name)
	assert.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, *wf.Spec.Suspend)

	// operate should not result in no workflows being created since it is suspended
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// resume the workflow and operate again. two pods should be able to be scheduled
	err = util.ResumeWorkflow(ctx, wfcset, controller.hydrator, wf.ObjectMeta.Name, "")
	assert.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Nil(t, wf.Spec.Suspend)
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err = listPods(woc)
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(suspendTemplateWithDeadline)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	updatedWf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
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

	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(sequence)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	updatedWf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
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

	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(inputParametersAsJson)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	updatedWf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(expandWithItems)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	newSteps, err := woc.expandStep(wf.Spec.Templates[0].Steps[0].Steps[0])
	assert.NoError(t, err)
	assert.Equal(t, 5, len(newSteps))
	woc.operate(ctx)
	pods, err := listPods(woc)
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

	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(expandWithItemsMap)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(suspendTemplate)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// resume the workflow. verify resume workflow edits nodestatus correctly
	err = util.ResumeWorkflow(ctx, wfcset, controller.hydrator, wf.ObjectMeta.Name, "")
	assert.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.False(t, util.IsWorkflowSuspended(wf))

	// operate the workflow. it should reach the second step
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err = listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pods.Items))
}

func TestSuspendTemplateWithFailedResume(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(suspendTemplate)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// resume the workflow. verify resume workflow edits nodestatus correctly
	err = util.StopWorkflow(ctx, wfcset, controller.hydrator, wf.ObjectMeta.Name, "inputs.parameters.param1.value=value1", "Step failed!")
	assert.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.False(t, util.IsWorkflowSuspended(wf))

	// operate the workflow. it should be failed and not reach the second step
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	pods, err = listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))
}

func TestSuspendTemplateWithFilteredResume(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(suspendTemplate)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// resume the workflow, but with non-matching selector
	err = util.ResumeWorkflow(ctx, wfcset, controller.hydrator, wf.ObjectMeta.Name, "inputs.paramaters.param1.value=value2")
	assert.Error(t, err)

	// operate the workflow. nothing should have happened
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err = listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))
	assert.True(t, util.IsWorkflowSuspended(wf))

	// resume the workflow, but with matching selector
	err = util.ResumeWorkflow(ctx, wfcset, controller.hydrator, wf.ObjectMeta.Name, "inputs.parameters.param1.value=value1")
	assert.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.False(t, util.IsWorkflowSuspended(wf))

	// operate the workflow. it should reach the second step
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err = listPods(woc)
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(suspendResumeAfterTemplate)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// wait 4 seconds
	time.Sleep(4 * time.Second)

	// operate the workflow. it should reach the second step
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err = listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pods.Items))
}

func TestSuspendResumeAfterTemplateNoWait(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(suspendResumeAfterTemplate)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(pods.Items))

	// don't wait

	// operate the workflow. it should have not reached the second step since not enough time passed
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err = listPods(woc)
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

	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(volumeWithParam)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	pod, err := getPod(woc, wf.Name)
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
	testVal := wfv1.AnyStringPtr("test-value")
	param := wfv1.Parameter{
		Name:  "test-param",
		Value: testVal,
	}
	// Make sure if the param is not global, don't add to scope
	woc.addParamToGlobalScope(param)
	assert.Nil(t, woc.wf.Status.Outputs)

	// Now set it as global. Verify it is added to workflow outputs
	param.GlobalName = "global-param"
	woc.addParamToGlobalScope(param)
	assert.Equal(t, 1, len(woc.wf.Status.Outputs.Parameters))
	assert.Equal(t, param.GlobalName, woc.wf.Status.Outputs.Parameters[0].Name)
	assert.Equal(t, testVal, woc.wf.Status.Outputs.Parameters[0].Value)
	assert.Equal(t, testVal.String(), woc.globalParams["workflow.outputs.parameters.global-param"])

	// Change the value and verify it is reflected in workflow outputs
	newValue := wfv1.AnyStringPtr("new-value")
	param.Value = newValue
	woc.addParamToGlobalScope(param)
	assert.Equal(t, 1, len(woc.wf.Status.Outputs.Parameters))
	assert.Equal(t, param.GlobalName, woc.wf.Status.Outputs.Parameters[0].Name)
	assert.Equal(t, newValue, woc.wf.Status.Outputs.Parameters[0].Value)
	assert.Equal(t, newValue.String(), woc.globalParams["workflow.outputs.parameters.global-param"])

	// Add a new global parameter
	param.GlobalName = "global-param2"
	woc.addParamToGlobalScope(param)
	assert.Equal(t, 2, len(woc.wf.Status.Outputs.Parameters))
	assert.Equal(t, param.GlobalName, woc.wf.Status.Outputs.Parameters[1].Name)
	assert.Equal(t, newValue, woc.wf.Status.Outputs.Parameters[1].Value)
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
	woc.addArtifactToGlobalScope(art)
	assert.Nil(t, woc.wf.Status.Outputs)

	// Now mark it as global. Verify it is added to workflow outputs
	art.GlobalName = "global-art"
	woc.addArtifactToGlobalScope(art)
	assert.Equal(t, 1, len(woc.wf.Status.Outputs.Artifacts))
	assert.Equal(t, art.GlobalName, woc.wf.Status.Outputs.Artifacts[0].Name)
	assert.Equal(t, "some/key", woc.wf.Status.Outputs.Artifacts[0].S3.Key)

	// Change the value and verify update is reflected
	art.S3.Key = "new/key"
	woc.addArtifactToGlobalScope(art)
	assert.Equal(t, 1, len(woc.wf.Status.Outputs.Artifacts))
	assert.Equal(t, art.GlobalName, woc.wf.Status.Outputs.Artifacts[0].Name)
	assert.Equal(t, "new/key", woc.wf.Status.Outputs.Artifacts[0].S3.Key)

	// Add a new global artifact
	art.GlobalName = "global-art2"
	art.S3.Key = "new/new/key"
	woc.addArtifactToGlobalScope(art)
	assert.Equal(t, 2, len(woc.wf.Status.Outputs.Artifacts))
	assert.Equal(t, art.GlobalName, woc.wf.Status.Outputs.Artifacts[1].Name)
	assert.Equal(t, "new/new/key", woc.wf.Status.Outputs.Artifacts[1].S3.Key)
}

func TestParamSubstitutionWithArtifact(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@../../test/e2e/functional/param-sub-with-artifacts.yaml")
	woc := newWoc(*wf)
	ctx := context.Background()
	woc.operate(ctx)
	wf, err := woc.controller.wfclientset.ArgoprojV1alpha1().Workflows("").Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wf.Status.Phase, wfv1.WorkflowRunning)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, len(pods.Items), 1)
}

func TestGlobalParamSubstitutionWithArtifact(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@../../test/e2e/functional/global-param-sub-with-artifacts.yaml")
	woc := newWoc(*wf)
	ctx := context.Background()
	woc.operate(ctx)
	wf, err := woc.controller.wfclientset.ArgoprojV1alpha1().Workflows("").Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wf.Status.Phase, wfv1.WorkflowRunning)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, len(pods.Items), 1)
}

func TestExpandWithSequence(t *testing.T) {
	var seq wfv1.Sequence
	var items []wfv1.Item
	var err error

	seq = wfv1.Sequence{
		Count: intstrutil.ParsePtr("10"),
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(items))
	assert.Equal(t, "0", items[0].GetStrVal())
	assert.Equal(t, "9", items[9].GetStrVal())

	seq = wfv1.Sequence{
		Start: intstrutil.ParsePtr("101"),
		Count: intstrutil.ParsePtr("10"),
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 10, len(items))
	assert.Equal(t, "101", items[0].GetStrVal())
	assert.Equal(t, "110", items[9].GetStrVal())

	seq = wfv1.Sequence{
		Start: intstrutil.ParsePtr("50"),
		End:   intstrutil.ParsePtr("60"),
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 11, len(items))
	assert.Equal(t, "50", items[0].GetStrVal())
	assert.Equal(t, "60", items[10].GetStrVal())

	seq = wfv1.Sequence{
		Start: intstrutil.ParsePtr("60"),
		End:   intstrutil.ParsePtr("50"),
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 11, len(items))
	assert.Equal(t, "60", items[0].GetStrVal())
	assert.Equal(t, "50", items[10].GetStrVal())

	seq = wfv1.Sequence{
		Count: intstrutil.ParsePtr("0"),
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(items))

	seq = wfv1.Sequence{
		Start: intstrutil.ParsePtr("8"),
		End:   intstrutil.ParsePtr("8"),
	}
	items, err = expandSequence(&seq)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, "8", items[0].GetStrVal())

	seq = wfv1.Sequence{
		Format: "testuser%02X",
		Count:  intstrutil.ParsePtr("10"),
		Start:  intstrutil.ParsePtr("1"),
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
	ctx := context.Background()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow(metadataTemplate)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(woc)
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
      image: argoproj/argosay:v2
      command: [sh, -c, 'head -n {{inputs.parameters.lines-count}} <"{{inputs.artifacts.text.path}}" | tee "{{outputs.artifacts.text.path}}" | wc -l > "{{outputs.parameters.actual-lines-count.path}}"']
`

func TestResolveIOPathPlaceholders(t *testing.T) {
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(ioPathPlaceholders)
	woc := newWoc(*wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.True(t, len(pods.Items) > 0, "pod was not created successfully")

	assert.Equal(t, []string{"/var/run/argo/argoexec", "emissary", "--", "sh", "-c", "head -n 3 <\"/inputs/text/data\" | tee \"/outputs/text/data\" | wc -l > \"/outputs/actual-lines-count/data\""}, pods.Items[0].Spec.Containers[1].Command)
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
      image: argoproj/argosay:v2
`

func TestResolvePlaceholdersInOutputValues(t *testing.T) {
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(outputValuePlaceholders)
	woc := newWoc(*wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.True(t, len(pods.Items) > 0, "pod was not created successfully")

	tmpl, err := getPodTemplate(&pods.Items[0])
	assert.NoError(t, err)
	parameterValue := tmpl.Outputs.Parameters[0].Value
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
      image: argoproj/argosay:v2
`

func TestResolvePodNameInRetries(t *testing.T) {
	tests := []struct {
		podNameVersion string
		wantPodName    string
	}{
		{"v1", "output-value-placeholders-wf-3033990984"},
		{"v2", "output-value-placeholders-wf-tell-pod-name-3033990984"},
	}
	for _, tt := range tests {
		_ = os.Setenv("POD_NAMES", tt.podNameVersion)
		ctx := context.Background()
		wf := wfv1.MustUnmarshalWorkflow(podNameInRetries)
		woc := newWoc(*wf)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		pods, err := woc.controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(ctx, metav1.ListOptions{})
		assert.NoError(t, err)
		assert.True(t, len(pods.Items) > 0, "pod was not created successfully")

		template, err := getPodTemplate(&pods.Items[0])
		assert.NoError(t, err)
		parameterValue := template.Outputs.Parameters[0].Value
		assert.NotNil(t, parameterValue)
		assert.Equal(t, tt.wantPodName, parameterValue.String())
	}
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(outputStatuses)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(resourceTemplate)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.WorkflowRunning, wf.Status.Phase)

	pod, err := getPod(woc, "resource-template")
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}
	tmpl, err := getPodTemplate(pod)
	if assert.NoError(t, err) {
		cm := apiv1.ConfigMap{}
		err = yaml.Unmarshal([]byte(tmpl.Resource.Manifest), &cm)
		if !assert.NoError(t, err) {
			t.Fatal(err)
		}
		assert.Equal(t, "resource-cm", cm.Name)
		assert.Empty(t, cm.ObjectMeta.OwnerReferences)
	}
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(resourceWithOwnerReferenceTemplate)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, wfv1.WorkflowRunning, wf.Status.Phase)

	pods, err := listPods(woc)
	if !assert.NoError(t, err) {
		t.Fatal(err)
	}

	objectMetas := map[string]metav1.ObjectMeta{}
	for _, pod := range pods.Items {
		tmpl, err := getPodTemplate(&pod)
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(stepScriptTmpl)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	assert.True(t, hasOutputResultRef("generate", &wf.Spec.Templates[0]))
	assert.False(t, hasOutputResultRef("print-message", &wf.Spec.Templates[0]))
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, "generate") {
			assert.Equal(t, "generate", getStepOrDAGTaskName(node.Name))
		} else if strings.Contains(node.Name, "print-message") {
			assert.Equal(t, "print-message", getStepOrDAGTaskName(node.Name))
		}
	}
}

func TestDAGWFGetNodeName(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(dagScriptTmpl)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	assert.True(t, hasOutputResultRef("A", &wf.Spec.Templates[0]))
	assert.False(t, hasOutputResultRef("B", &wf.Spec.Templates[0]))
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
	assert.NoError(t, err)
	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, ".A") {
			assert.Equal(t, "A", getStepOrDAGTaskName(node.Name))
		}
		if strings.Contains(node.Name, ".B") {
			assert.Equal(t, "B", getStepOrDAGTaskName(node.Name))
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(withParamAsJsonList)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pods, err := listPods(woc)
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
	wf := wfv1.MustUnmarshalWorkflow(stepsOnExit)
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
      args: ["echo send e-mail: {{workflow.name}} {{workflow.status}} {{workflow.duration}}. Failed steps {{workflow.failures}}"]
`

func TestStepsOnExitFailures(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(onExitFailures)
	cancel, controller := newController(wf)
	defer cancel()

	// Test list expansion
	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.Contains(t, woc.globalParams[common.GlobalVarWorkflowFailures], `[{\"displayName\":\"exit-handlers\",\"message\":\"Pod failed\",\"templateName\":\"intentional-fail\",\"phase\":\"Failed\",\"podName\":\"exit-handlers\"`)
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(onExitTimeout)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
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

func TestEventNodeEvents(t *testing.T) {
	for manifest, want := range map[string][]string{
		// Invalid spec
		`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: invalid-spec
spec:
  entrypoint: 123
`: {
			"Warning WorkflowFailed invalid spec: template name '123' undefined",
		},
		// DAG
		`
metadata:
  name: dag-events
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: a
            template: whalesay
    - name: whalesay
      container:
        image: docker/whalesay:latest
`: {
			"Normal WorkflowRunning Workflow Running",
			"Normal WorkflowNodeRunning Running node dag-events",
			"Normal WorkflowNodeRunning Running node dag-events.a",
			"Normal WorkflowNodeSucceeded Succeeded node dag-events.a",
			"Normal WorkflowNodeSucceeded Succeeded node dag-events",
			"Normal WorkflowSucceeded Workflow completed",
		},
		// steps
		`
metadata:
  name: steps-events
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: a
            template: whalesay
    - name: whalesay
      container:
        image: docker/whalesay:latest
`: {
			"Normal WorkflowRunning Workflow Running",
			"Normal WorkflowNodeRunning Running node steps-events",
			"Normal WorkflowNodeRunning Running node steps-events[0]",
			"Normal WorkflowNodeRunning Running node steps-events[0].a",
			"Normal WorkflowNodeSucceeded Succeeded node steps-events[0].a",
			"Normal WorkflowNodeSucceeded Succeeded node steps-events[0]",
			"Normal WorkflowNodeSucceeded Succeeded node steps-events",
			"Normal WorkflowSucceeded Workflow completed",
		},
		// no DAG or steps
		`
metadata:
  name: no-dag-or-steps
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`: {
			"Normal WorkflowRunning Workflow Running",
			"Normal WorkflowNodeRunning Running node no-dag-or-steps",
			"Normal WorkflowNodeSucceeded Succeeded node no-dag-or-steps",
			"Normal WorkflowSucceeded Workflow completed",
		},
	} {
		wf := wfv1.MustUnmarshalWorkflow(manifest)
		ctx := context.Background()
		t.Run(wf.Name, func(t *testing.T) {
			cancel, controller := newController(wf)
			defer cancel()
			woc := newWorkflowOperationCtx(wf, controller)
			woc.operate(ctx)
			makePodsPhase(ctx, woc, apiv1.PodSucceeded)
			woc = newWorkflowOperationCtx(woc.wf, controller)
			woc.operate(ctx)
			assert.ElementsMatch(t, want, getEvents(controller, len(want)))
		})
	}
}

func TestEventNodeEventsAsPod(t *testing.T) {
	for manifest, want := range map[string][]string{
		// Invalid spec
		`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: invalid-spec
spec:
  entrypoint: 123
`: {
			"Warning WorkflowFailed invalid spec: template name '123' undefined",
		},
		// DAG
		`
metadata:
  name: dag-events
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: a
            template: whalesay
    - name: whalesay
      container:
        image: docker/whalesay:latest
`: {
			"Normal WorkflowRunning Workflow Running",
			"Normal WorkflowNodeRunning Running node dag-events",
			"Normal WorkflowNodeRunning Running node dag-events.a",
			"Normal WorkflowNodeSucceeded Succeeded node dag-events.a",
			"Normal WorkflowNodeSucceeded Succeeded node dag-events",
			"Normal WorkflowSucceeded Workflow completed",
		},
		// steps
		`
metadata:
  name: steps-events
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: a
            template: whalesay
    - name: whalesay
      container:
        image: docker/whalesay:latest
`: {
			"Normal WorkflowRunning Workflow Running",
			"Normal WorkflowNodeRunning Running node steps-events",
			"Normal WorkflowNodeRunning Running node steps-events[0]",
			"Normal WorkflowNodeRunning Running node steps-events[0].a",
			"Normal WorkflowNodeSucceeded Succeeded node steps-events[0].a",
			"Normal WorkflowNodeSucceeded Succeeded node steps-events[0]",
			"Normal WorkflowNodeSucceeded Succeeded node steps-events",
			"Normal WorkflowSucceeded Workflow completed",
		},
		// no DAG or steps
		`
metadata:
  name: no-dag-or-steps
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`: {
			"Normal WorkflowRunning Workflow Running",
			"Normal WorkflowNodeRunning Running node no-dag-or-steps",
			"Normal WorkflowNodeSucceeded Succeeded node no-dag-or-steps",
			"Normal WorkflowSucceeded Workflow completed",
		},
	} {
		wf := wfv1.MustUnmarshalWorkflow(manifest)
		ctx := context.Background()
		t.Run(wf.Name, func(t *testing.T) {
			cancel, controller := newController(wf)
			defer cancel()
			controller.Config.NodeEvents = config.NodeEvents{SendAsPod: true}
			woc := newWorkflowOperationCtx(wf, controller)
			createRunningPods(ctx, woc)
			woc.operate(ctx)
			makePodsPhase(ctx, woc, apiv1.PodSucceeded)
			woc = newWorkflowOperationCtx(woc.wf, controller)
			woc.operate(ctx)
			assert.ElementsMatch(t, want, getEvents(controller, len(want)))
		})
	}
}

func getEvents(controller *WorkflowController, num int) []string {
	c := controller.eventRecorderManager.(*testEventRecorderManager).eventRecorder.Events
	events := make([]string, num)
	for i := 0; i < num; i++ {
		events[i] = <-c
	}
	return events
}

func TestGetPodByNode(t *testing.T) {
	workflowText := `
metadata:
  name: dag-events
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: a
            template: whalesay
    - name: whalesay
      container:
        image: docker/whalesay:latest
`
	wf := wfv1.MustUnmarshalWorkflow(workflowText)
	wf.Namespace = "argo"
	ctx := context.Background()
	cancel, controller := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, controller)
	createRunningPods(ctx, woc)
	woc.operate(ctx)
	time.Sleep(time.Second)
	// Parent dag node has no pod
	parentNode := woc.wf.GetNodeByName("dag-events")
	pod, err := woc.getPodByNode(parentNode)
	assert.Nil(t, pod)
	assert.Error(t, err, "Expected node type Pod, got DAG")
	// Pod node should return a pod
	podNode := woc.wf.GetNodeByName("dag-events.a")
	pod, err = woc.getPodByNode(podNode)
	assert.NoError(t, err)
	assert.NotNil(t, pod)
	// Invalid node should not return a pod
	invalidNode := wfv1.NodeStatus{Type: wfv1.NodeTypePod, Name: "doesnt-exist"}
	pod, err = woc.getPodByNode(&invalidNode)
	assert.Nil(t, pod)
	assert.Nil(t, err)
}

var pdbwf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-pdb-wf
spec:
  entrypoint: main
  poddisruptionbudget:
    minavailable: 100%
  templates:
  - name: main
    container:
      image: docker/whalesay:latest
`

func TestPDBCreation(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pdbwf)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	pdb, _ := controller.kubeclientset.PolicyV1beta1().PodDisruptionBudgets("").Get(ctx, woc.wf.Name, metav1.GetOptions{})
	assert.Equal(t, pdb.Name, wf.Name)
	woc.markWorkflowSuccess(ctx)
	_, err := controller.kubeclientset.PolicyV1beta1().PodDisruptionBudgets("").Get(ctx, woc.wf.Name, metav1.GetOptions{})
	assert.EqualError(t, err, "poddisruptionbudgets.policy \"my-pdb-wf\" not found")
}

func TestPDBCreationRaceDelete(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pdbwf)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	err := controller.kubeclientset.PolicyV1beta1().PodDisruptionBudgets("").Delete(ctx, woc.wf.Name, metav1.DeleteOptions{})
	assert.NoError(t, err)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestStatusConditions(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pdbwf)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Empty(t, woc.wf.Status.Conditions)
	woc.markWorkflowSuccess(ctx)
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(nestedOptionalOutputArtifacts)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.Nil(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

//  TestPodSpecLogForFailedPods tests PodSpec logging configuration
func TestPodSpecLogForFailedPods(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	controller.Config.PodSpecLogStrategy.FailedPod = true
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	for _, node := range woc.wf.Status.Nodes {
		assert.True(t, woc.shouldPrintPodSpec(&node))
	}
}

//  TestPodSpecLogForAllPods tests  PodSpec logging configuration
func TestPodSpecLogForAllPods(t *testing.T) {
	cancel, controller := newController()
	defer cancel()

	ctx := context.Background()
	assert.NotNil(t, controller)
	controller.Config.PodSpecLogStrategy.AllPods = true
	wf := wfv1.MustUnmarshalWorkflow(nestedOptionalOutputArtifacts)
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	woc.operate(ctx)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	for _, node := range woc.wf.Status.Nodes {
		assert.True(t, woc.shouldPrintPodSpec(&node))
	}
}

var retryNodeOutputs = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: daemon-step-dvbnn
spec:
  
  entrypoint: daemon-example
  templates:
  - 
    inputs: {}
    metadata: {}
    name: daemon-example
    outputs: {}
    steps:
    - - 
        name: influx
        template: influxdb
  - 
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
	ctx := context.Background()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow(retryNodeOutputs)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.ObjectMeta.Name, metav1.GetOptions{})
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
	assert.Contains(t, scope.scope, "steps.influx.id")
	assert.Contains(t, scope.scope, "steps.influx.startedAt")
	assert.Contains(t, scope.scope, "steps.influx.finishedAt")
}

var workflowWithPVCAndFailingStep = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: wf-with-pvc
spec:
  entrypoint: entrypoint
  templates:
  - name: entrypoint
    steps:
    - - name: succeed
        template: succeed
    - - name: failure
        template: failure
  - name: succeed
    script:
      args: [success]
      command: [cowsay]
      image: docker/whalesay:latest
      volumeMounts:
      - mountPath: /data
        name: data
  - name: failure
    script:
      command: [sh]
      image: alpine
      args: [exit, "1"]
      volumeMounts:
      - mountPath: /data
        name: data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes:
      - ReadWriteOnce
      resources:
        requests:
          storage: 1Gi
status:
  nodes:
    wf-with-pvc:
      name: wf-with-pvc
      phase: Failed
    wf-with-pvc-2390440388:
      name: wf-with-pvc(0)[0].succeed
      phase: Succeeded
    wf-with-pvc-3099954303:
      name: wf-with-pvc(0)[1].failure
      phase: Failed
  persistentVolumeClaims:
  - name: data
    persistentVolumeClaim:
      claimName: wf-with-pvc-data
`

// This test ensures that the PVCs used in the steps are not deleted when
// the workflow fails
func TestDeletePVCDoesNotDeletePVCOnFailedWorkflow(t *testing.T) {
	assert := assert.New(t)

	wf := wfv1.MustUnmarshalWorkflow(workflowWithPVCAndFailingStep)
	cancel, controller := newController(wf)
	defer cancel()

	woc := newWorkflowOperationCtx(wf, controller)

	assert.Len(woc.wf.Status.PersistentVolumeClaims, 1, "1 PVC before operating")

	ctx := context.Background()
	woc.operate(ctx)

	node1 := woc.wf.GetNodeByName("wf-with-pvc(0)[0].succeed")
	node2 := woc.wf.GetNodeByName("wf-with-pvc(0)[1].failure")

	// Node 1 Succeeded
	assert.Equal(node1.Phase, wfv1.NodeSucceeded)
	// Node 2 Failed
	assert.Equal(node2.Phase, wfv1.NodeFailed)
	// Hence, PVCs should stick around
	assert.Len(woc.wf.Status.PersistentVolumeClaims, 1, "PVCs not deleted")
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(containerOutputsResult)
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)

	assert.True(t, hasOutputResultRef("hello1", &wf.Spec.Templates[0]))
	assert.False(t, hasOutputResultRef("hello2", &wf.Spec.Templates[0]))

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, "hello1") {
			assert.Equal(t, "hello1", getStepOrDAGTaskName(node.Name))
		} else if strings.Contains(node.Name, "hello2") {
			assert.Equal(t, "hello2", getStepOrDAGTaskName(node.Name))
		}
	}
}

var nestedStepGroupGlobalParams = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: global-outputs-bg7gl
spec:
  
  entrypoint: generate-globals
  templates:
  - 
    inputs: {}
    metadata: {}
    name: generate-globals
    outputs: {}
    steps:
    - - 
        name: generate
        template: nested-global-output-generation
  - 
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
  - 
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
    - - 
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
      id: global-outputs-bg7gl-1831647575
      name: global-outputs-bg7gl[0]
      phase: Running
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
      id: global-outputs-bg7gl-3089902334
      name: global-outputs-bg7gl[0].generate[0]
      phase: Running
      startedAt: "2020-04-24T15:55:11Z"
      templateName: nested-global-output-generation
      templateScope: local/global-outputs-bg7gl
      type: StepGroup
  startedAt: "2020-04-24T15:55:11Z"
`

func TestNestedStepGroupGlobalParams(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(nestedStepGroupGlobalParams)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.FindByDisplayName("generate")
	if assert.NotNil(t, node) && assert.NotNil(t, node.Outputs) && assert.Len(t, node.Outputs.Parameters, 1) {
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(globalVariablePlaceholders)
	woc := newWoc(*wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.True(t, len(pods.Items) > 0, "pod was not created successfully")

	template, err := getPodTemplate(&pods.Items[0])
	assert.NoError(t, err)
	namespaceValue := template.Outputs.Parameters[0].Value
	assert.NotNil(t, namespaceValue)
	assert.Equal(t, "testNamespace", namespaceValue.String())
	serviceAccountNameValue := template.Outputs.Parameters[1].Value
	assert.NotNil(t, serviceAccountNameValue)
	assert.Equal(t, "testServiceAccountName", serviceAccountNameValue.String())
}

var unsuppliedArgValue = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: wf-with-unsupplied-param-
spec:
  arguments:
    parameters:
    - name: missing
  entrypoint: whalesay
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
    name: whalesay
    outputs: {}
`

func TestUnsuppliedArgValue(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(unsuppliedArgValue)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, woc.wf.Status.Conditions[0].Status, metav1.ConditionStatus("True"))
	assert.Equal(t, woc.wf.Status.Message, "invalid spec: spec.arguments.missing.value is required")
}

var suppliedArgValue = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: wf-with-supplied-param-
spec:
  arguments:
    parameters:
    - name: message
      value: argo
      valueFrom:
        supplied: {}
  entrypoint: whalesay
  templates:
  - container:
      args: ["{{workflow.parameters.message}}"]
      command: ["cowsay"]
      image: docker/whalesay:latest
    name: whalesay
`

// TestSuppliedArgValue ensures that supplied workflow parameters are correctly set in the global parameters
func TestSuppliedArgValue(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(suppliedArgValue)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, "argo", woc.globalParams["workflow.parameters.message"])
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
  
  entrypoint: echo
  templates:
  - 
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
	wf := wfv1.MustUnmarshalWorkflow(maxDurationOnErroredFirstNode)

	// Simulate node failed just now
	node := wf.Status.Nodes["echo-wngc4-1641470511"]
	node.StartedAt = metav1.Time{Time: time.Now().Add(-1 * time.Second)}
	wf.Status.Nodes["echo-wngc4-1641470511"] = node

	ctx := context.Background()
	woc := newWoc(*wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

var backoffExceedsMaxDuration = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: echo-r6v49
spec:
  
  entrypoint: echo
  templates:
  - 
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
	wf := wfv1.MustUnmarshalWorkflow(backoffExceedsMaxDuration)

	// Simulate node failed just now
	node := wf.Status.Nodes["echo-r6v49-3721138751"]
	node.StartedAt = metav1.Time{Time: time.Now().Add(-1 * time.Second)}
	node.FinishedAt = metav1.Time{Time: time.Now()}
	wf.Status.Nodes["echo-r6v49-3721138751"] = node

	ctx := context.Background()
	woc := newWoc(*wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Equal(t, "Backoff would exceed max duration limit", woc.wf.Status.Nodes["echo-r6v49"].Message)
	assert.Equal(t, "Backoff would exceed max duration limit", woc.wf.Status.Message)
}

var noOnExitWhenSkipped = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-primay-branch-sd6rg
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
    name: pass
    outputs: {}
  - 
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
  - 
    dag:
      tasks:
      - 
        name: A
        template: pass
      - 
        dependencies:
        - A
        name: B
        onExit: exit
        template: pass
        when: '{{tasks.A.status}} != Succeeded'
      - 
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
	wf := wfv1.MustUnmarshalWorkflow(noOnExitWhenSkipped)

	ctx := context.Background()
	woc := newWoc(*wf)
	woc.operate(ctx)
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
	wf := wfv1.MustUnmarshalWorkflow(noOnExitWhenSkipped)
	woc := newWoc(*wf)

	// This should make the call to "operate" panic
	ctx := context.Background()
	woc.preExecutionNodePhases = nil
	woc.operate(ctx)

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
	wf := wfv1.MustUnmarshalWorkflow(globalVariablePlaceholders)
	cancel, controller := newController()
	defer cancel()

	ctx := context.Background()
	controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{}
	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingStrict
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Equal(t, "workflows must use workflowTemplateRef to be executed when the controller is in reference mode", woc.wf.Status.Message)

	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingSecure
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Equal(t, "workflows must use workflowTemplateRef to be executed when the controller is in reference mode", woc.wf.Status.Message)

	controller.Config.WorkflowRestrictions = nil
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestValidReferenceMode(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/workflow-template-ref.yaml")
	wfTmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/workflow-template-submittable.yaml")
	cancel, controller := newController(wf, wfTmpl)
	defer cancel()

	ctx := context.Background()
	controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{}
	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingSecure
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// Change stored Workflow Spec
	woc.wf.Status.StoredWorkflowSpec.Entrypoint = "different"
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Equal(t, "WorkflowSpec may not change during execution when the controller is set `templateReferencing: Secure`", woc.wf.Status.Message)

	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingStrict
	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// Change stored Workflow Spec
	woc.wf.Status.StoredWorkflowSpec.Entrypoint = "different"
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
}

var workflowStatusMetric = `
metadata:
  name: retry-to-completion-rngcr
spec:
  
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
  - 
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
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(workflowStatusMetric)
	woc := newWoc(*wf)
	woc.operate(ctx)
	// Must only be two (completed: true), (podRunning: true)
	assert.Len(t, woc.wf.Status.Conditions, 2)
}

func TestWorkflowConditions(t *testing.T) {
	ctx := context.Background()
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf
  namespace: my-ns
spec:
  entrypoint: main
  templates:
  - container:
      image: docker/whalesay:latest
    name: main
`)
	cancel, controller := newController(wf)
	defer cancel()

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Nil(t, woc.wf.Status.Conditions, "zero conditions on first reconciliation")
	makePodsPhase(ctx, woc, apiv1.PodPending)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Conditions{{Type: wfv1.ConditionTypePodRunning, Status: metav1.ConditionFalse}}, woc.wf.Status.Conditions)

	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Conditions{{Type: wfv1.ConditionTypePodRunning, Status: metav1.ConditionTrue}}, woc.wf.Status.Conditions)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Conditions{
		{Type: wfv1.ConditionTypePodRunning, Status: metav1.ConditionFalse},
		{Type: wfv1.ConditionTypeCompleted, Status: metav1.ConditionTrue},
	}, woc.wf.Status.Conditions)
}

var workflowCached = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: memoized-workflow-test
  namespace: default
spec:
  entrypoint: whalesay
  arguments:
    parameters:
    - name: message
      value: hi-there-world
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
    memoize:
      key: "{{inputs.parameters.message}}"
      cache:
        configMap:
          name: whalesay-cache
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["sleep 10; cowsay {{inputs.parameters.message}} > /tmp/hello_world.txt"]
    outputs:
      parameters:
      - name: hello
        valueFrom:
          path: /tmp/hello_world.txt
`

func TestConfigMapCacheLoadOperate(t *testing.T) {
	sampleConfigMapCacheEntry := apiv1.ConfigMap{
		Data: map[string]string{
			"hi-there-world": `{"nodeID":"memoized-simple-workflow-5wj2p","outputs":{"parameters":[{"name":"hello","value":"foobar","valueFrom":{"path":"/tmp/hello_world.txt"}}],"artifacts":[{"name":"main-logs","archiveLogs":true,"s3":{"endpoint":"minio:9000","bucket":"my-bucket","insecure":true,"accessKeySecret":{"name":"my-minio-cred","key":"accesskey"},"secretKeySecret":{"name":"my-minio-cred","key":"secretkey"},"key":"memoized-simple-workflow-5wj2p/memoized-simple-workflow-5wj2p/main.log"}}]},"creationTimestamp":"2020-09-21T18:12:56Z"}`,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "whalesay-cache",
			ResourceVersion: "1630732",
		},
	}
	wf := wfv1.MustUnmarshalWorkflow(workflowCached)
	cancel, controller := newController()
	defer cancel()

	ctx := context.Background()
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapCacheEntry, metav1.CreateOptions{})
	assert.NoError(t, err)

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	if assert.Len(t, woc.wf.Status.Nodes, 1) {
		for _, node := range woc.wf.Status.Nodes {
			assert.NotNil(t, node.Outputs)
			assert.Equal(t, "hello", node.Outputs.Parameters[0].Name)
			assert.Equal(t, "foobar", node.Outputs.Parameters[0].Value.String())
			assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
		}
	}
}

var workflowCachedMaxAge = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: memoized-workflow-test
spec:
  entrypoint: whalesay
  arguments:
    parameters:
    - name: message
      value: hi-there-world
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
    memoize:
      key: "{{inputs.parameters.message}}"
      maxAge: '10s'
      cache:
        configMap:
          name: whalesay-cache
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["sleep 10; cowsay {{inputs.parameters.message}} > /tmp/hello_world.txt"]
    outputs:
      parameters:
      - name: hello
        valueFrom:
          path: /tmp/hello_world.txt
`

func TestConfigMapCacheLoadOperateMaxAge(t *testing.T) {
	getEntryCreatedAtTime := func(time time.Time) apiv1.ConfigMap {
		jsonTime, _ := time.UTC().MarshalJSON()
		return apiv1.ConfigMap{
			Data: map[string]string{
				"hi-there-world": fmt.Sprintf(`{"nodeID":"memoized-simple-workflow-5wj2p","outputs":{"parameters":[{"name":"hello","value":"foobar","valueFrom":{"path":"/tmp/hello_world.txt"}}],"artifacts":[{"name":"main-logs","archiveLogs":true,"s3":{"endpoint":"minio:9000","bucket":"my-bucket","insecure":true,"accessKeySecret":{"name":"my-minio-cred","key":"accesskey"},"secretKeySecret":{"name":"my-minio-cred","key":"secretkey"},"key":"memoized-simple-workflow-5wj2p/memoized-simple-workflow-5wj2p/main.log"}}]},"creationTimestamp":%s}`, string(jsonTime)),
			},
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:            "whalesay-cache",
				ResourceVersion: "1630732",
			},
		}
	}
	wf := wfv1.MustUnmarshalWorkflow(workflowCachedMaxAge)
	cancel, controller := newController()

	ctx := context.Background()
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)

	nonExpiredEntry := getEntryCreatedAtTime(time.Now().Add(-5 * time.Second))
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &nonExpiredEntry, metav1.CreateOptions{})
	assert.NoError(t, err)

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	if assert.Len(t, woc.wf.Status.Nodes, 1) {
		for _, node := range woc.wf.Status.Nodes {
			assert.NotNil(t, node.Outputs)
			assert.Equal(t, "hello", node.Outputs.Parameters[0].Name)
			assert.Equal(t, "foobar", node.Outputs.Parameters[0].Value.String())
			assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
		}
	}

	cancel()
	cancel, controller = newController()
	defer cancel()

	expiredEntry := getEntryCreatedAtTime(time.Now().Add(-15 * time.Second))
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &expiredEntry, metav1.CreateOptions{})
	assert.NoError(t, err)

	woc = newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	if assert.Len(t, woc.wf.Status.Nodes, 1) {
		for _, node := range woc.wf.Status.Nodes {
			assert.Nil(t, node.Outputs)
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
	}
}

func TestConfigMapCacheLoadNilOutputs(t *testing.T) {
	sampleConfigMapCacheEntry := apiv1.ConfigMap{
		Data: map[string]string{
			"hi-there-world": `{"ExpiresAt":"2020-06-18T17:11:05Z","NodeID":"memoize-abx4124-123129321123","Outputs":{}}`,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "whalesay-cache",
			ResourceVersion: "1630732",
		},
	}
	wf := wfv1.MustUnmarshalWorkflow(workflowCached)
	cancel, controller := newController()
	defer cancel()

	ctx := context.Background()
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	assert.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapCacheEntry, metav1.CreateOptions{})
	assert.NoError(t, err)

	woc := newWorkflowOperationCtx(wf, controller)
	fn := func() {
		woc.operate(ctx)
	}
	assert.NotPanics(t, fn)

	if assert.Len(t, woc.wf.Status.Nodes, 1) {
		for _, node := range woc.wf.Status.Nodes {
			assert.NotNil(t, node.Outputs)
			assert.False(t, node.Outputs.HasOutputs())
			assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
		}
	}
}

func TestConfigMapCacheSaveOperate(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(workflowCached)
	cancel, controller := newController(wf)
	defer cancel()

	woc := newWorkflowOperationCtx(wf, controller)
	sampleOutputs := wfv1.Outputs{
		Parameters: []wfv1.Parameter{
			{Name: "hello", Value: wfv1.AnyStringPtr("foobar")},
		},
		ExitCode: pointer.StringPtr("0"),
	}

	ctx := context.Background()
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded, withExitCode(0), withOutputs(wfv1.MustMarshallJSON(sampleOutputs)))
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	cm, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "whalesay-cache", metav1.GetOptions{})
	if assert.NoError(t, err) {
		assert.NotNil(t, cm)
		assert.NotNil(t, cm.Data)
		rawEntry, ok := cm.Data["hi-there-world"]
		assert.True(t, ok)
		var entry cache.Entry
		wfv1.MustUnmarshal(rawEntry, &entry)

		if assert.NotNil(t, entry.Outputs) {
			assert.Equal(t, sampleOutputs, *entry.Outputs)
		}
	}
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
	wf := wfv1.MustUnmarshalWorkflow(propagate)
	assert.NotNil(t, wf)
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotNil(t, woc)
	err := woc.setExecWorkflow(context.Background())
	assert.NoError(t, err)
	assert.Zero(t, len(woc.wf.Status.Nodes))

	// Add the parent node for retries.
	nodeName := "test-node"
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
	retries := wfv1.RetryStrategy{
		Limit: intstrutil.ParsePtr("2"),
		Backoff: &wfv1.Backoff{
			Duration:    "0",
			Factor:      intstrutil.ParsePtr("1"),
			MaxDuration: "20",
		},
	}
	woc.wf.Status.Nodes[woc.wf.NodeID(nodeName)] = *node

	childNode := fmt.Sprintf("child-node-%d", 0)
	woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed)
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
  
  entrypoint: resubmit-pending
  templates:
  - 
    inputs: {}
    metadata: {}
    name: resubmit-pending
    outputs: {}
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
	wf := wfv1.MustUnmarshalWorkflow(resubmitPendingWf)
	woc := newWorkflowOperationCtx(wf, controller)

	forbiddenErr := apierr.NewForbidden(schema.GroupResource{Group: "test", Resource: "test1"}, "test", errors.New("exceeded quota"))
	nonForbiddenErr := apierr.NewBadRequest("badrequest")
	t.Run("ForbiddenError", func(t *testing.T) {
		node, err := woc.requeueIfTransientErr(forbiddenErr, "resubmit-pending-wf")
		assert.NotNil(t, node)
		assert.NoError(t, err)
		assert.Equal(t, wfv1.NodePending, node.Phase)
	})
	t.Run("NonForbiddenError", func(t *testing.T) {
		node, err := woc.requeueIfTransientErr(nonForbiddenErr, "resubmit-pending-wf")
		assert.Error(t, err)
		assert.Nil(t, node)
	})
}

func TestResubmitMemoization(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-wf
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: busybox
status:
  phase: Failed
  nodes:
    my-wf:
      name: my-wf
      phase: Failed
`)
	wf, err := util.FormulateResubmitWorkflow(wf, true)
	if assert.NoError(t, err) {
		cancel, controller := newController(wf)
		defer cancel()

		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		for _, node := range woc.wf.Status.Nodes {
			switch node.TemplateName {
			case "main":
				assert.Equal(t, wfv1.NodePending, node.Phase)
				assert.False(t, node.StartTime().IsZero())
				assert.Equal(t, woc.wf.Labels[common.LabelKeyPreviousWorkflowName], "my-wf")
			case "":
			default:
				assert.Fail(t, "invalid template")
			}
		}
		list, err := listPods(woc)
		if assert.NoError(t, err) {
			assert.Len(t, list.Items, 1)
		}
	}
}

func TestWorkflowOutputs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf
  namespace: my-ns
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: step-1
        template: child
  - name: child
    container:
      image: my-image
    outputs:
      parameters:
      - name: my-param
        valueFrom: 
          path: /my-path
`)
	cancel, controller := newController(wf)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, controller)

	// reconcile
	ctx := context.Background()
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// make all created pods as successful
	makePodsPhase(ctx, woc, apiv1.PodSucceeded, withOutputs(`{"parameters": [{"name": "my-param"}]}`))

	// reconcile
	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

var globalVarsOnExit = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata: 
  name: hello-world-6gphm-8n22g
  namespace: default
spec: 
  arguments: 
    parameters: 
      - 
        name: message
        value: nononono
  workflowTemplateRef: 
    name: hello-world-6gphm
status: 
  nodes: 
    hello-world-6gphm-8n22g: 
      displayName: hello-world-6gphm-8n22g
      finishedAt: "2020-07-14T20:45:28Z"
      hostNodeName: minikube
      id: hello-world-6gphm-8n22g
      inputs: 
        parameters: 
          - 
            name: message
            value: nononono
      name: hello-world-6gphm-8n22g
      outputs: 
        artifacts: 
          - 
            archiveLogs: true
            name: main-logs
            s3: 
              accessKeySecret: 
                key: accesskey
                name: my-minio-cred
              bucket: my-bucket
              endpoint: "minio:9000"
              insecure: true
              key: hello-world-6gphm-8n22g/hello-world-6gphm-8n22g/main.log
              secretKeySecret: 
                key: secretkey
                name: my-minio-cred
        exitCode: "0"
      phase: Succeeded
      resourcesDuration: 
        cpu: 2
        memory: 1
      startedAt: "2020-07-14T20:45:25Z"
      templateRef: 
        name: hello-world-6gphm
        template: whalesay
      templateScope: local/hello-world-6gphm-8n22g
      type: Pod
  phase: Running
  resourcesDuration: 
    cpu: 5
    memory: 2
  startedAt: "2020-07-14T20:45:25Z"
  storedTemplates: 
    namespaced/hello-world-6gphm/whalesay: 
      
      container: 
        args: 
          - "hello {{inputs.parameters.message}}"
        command: 
          - cowsay
        image: "docker/whalesay:latest"
      inputs: 
        parameters: 
          - 
            name: message
      metadata: {}
      name: whalesay
      outputs: {}
  storedWorkflowTemplateSpec: 
    arguments: 
      parameters: 
        - 
          name: message
          value: nononono
    entrypoint: whalesay
    onExit: exitContainer
    templates: 
      - name: whalesay
        container:
          image: "docker/whalesay:latest"
          args: 
            - "hello {{inputs.parameters.message}}"
          command: 
            - cowsay
        inputs: 
          parameters: 
            - name: message
      - name: exitContainer
        container:
          image: docker/whalesay
          args: 
            - "goodbye {{inputs.parameters.message}}"
          command: 
            - cowsay
        inputs: 
          parameters: 
            - name: message
`

var wftmplGlobalVarsOnExit = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: hello-world-6gphm
  namespace: default
spec:
  entrypoint: whalesay
  onExit: exitContainer
  arguments:
    parameters:
    - name: message
      value: "default"
  templates:
  - name: whalesay
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello {{inputs.parameters.message}}"]
  - name: exitContainer
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay
      command: [cowsay]
      args: ["goodbye {{inputs.parameters.message}}"]
`

func TestGlobalVarsOnExit(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(globalVarsOnExit)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(wftmplGlobalVarsOnExit)
	cancel, controller := newController(wf, wftmpl)
	defer cancel()
	woc := newWorkflowOperationCtx(wf, controller)

	ctx := context.Background()
	woc.operate(ctx)

	node := woc.wf.Status.Nodes["hello-world-6gphm-8n22g-3224262006"]
	if assert.NotNil(t, node) && assert.NotNil(t, node.Inputs) && assert.NotEmpty(t, node.Inputs.Parameters) {
		assert.Equal(t, "nononono", node.Inputs.Parameters[0].Value.String())
	}
}

var deadlineWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-9fvnv
  namespace: argo
spec:
  activeDeadlineSeconds: 3
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: approve
        template: approve
  - name: approve
    suspend: {}
  - container:
      args:
      - sleep 50
      command:
      - sh
      - -c
      image: alpine:latest
      resources:
        requests:
          memory: 1Gi
    name: whalesay
status:
  finishedAt: null
  nodes:
    steps-9fvnv:
      children:
      - steps-9fvnv-3514116232
      displayName: steps-9fvnv
      finishedAt: null
      id: steps-9fvnv
      name: steps-9fvnv
      phase: Running
      startedAt: "2020-07-24T16:39:25Z"
      templateName: main
      templateScope: local/steps-9fvnv
      type: Steps

    steps-9fvnv-3514116232:
      boundaryID: steps-9fvnv
      children:
      - steps-9fvnv-3700512507
      displayName: '[0]'
      finishedAt: null
      id: steps-9fvnv-3514116232
      name: steps-9fvnv[0]
      phase: Running
      startedAt: "2020-07-24T16:39:25Z"
      templateName: main
      templateScope: local/steps-9fvnv
      type: StepGroup
    steps-9fvnv-3700512507:
      boundaryID: steps-9fvnv
      displayName: approve
      finishedAt: null
      id: steps-9fvnv-3700512507
      name: steps-9fvnv[0].approve
      phase: Running
      startedAt: "2020-07-24T16:39:25Z"
      templateName: approve
      templateScope: local/steps-9fvnv
      type: Suspend
  phase: Running
  startedAt: "2020-07-24T16:39:25Z"
`

func TestFailSuspendedAndPendingNodesAfterDeadline(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(deadlineWf)
	wf.Status.StartedAt = metav1.Now()
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	t.Run("Before Deadline", func(t *testing.T) {
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	})
	time.Sleep(3 * time.Second)
	t.Run("After Deadline", func(t *testing.T) {
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
		}
	})
}

func TestFailSuspendedAndPendingNodesAfterShutdown(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(deadlineWf)
	wf.Spec.Shutdown = wfv1.ShutdownStrategyStop
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	t.Run("After Shutdown", func(t *testing.T) {
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
		}
	})
}

func Test_processItem(t *testing.T) {
	task := wfv1.DAGTask{
		WithParam: `[{"number": 2, "string": "foo", "list": [0, "1"], "json": {"number": 2, "string": "foo", "list": [0, "1"]}}]`,
	}
	taskBytes, err := json.Marshal(task)
	assert.NoError(t, err)
	var items []wfv1.Item
	wfv1.MustUnmarshal([]byte(task.WithParam), &items)

	var newTask wfv1.DAGTask
	tmpl, _ := template.NewTemplate(string(taskBytes))
	newTaskName, err := processItem(tmpl, "task-name", 0, items[0], &newTask, "")
	if assert.NoError(t, err) {
		assert.Equal(t, `task-name(0:json:{"list":[0,"1"],"number":2,"string":"foo"},list:[0,"1"],number:2,string:foo)`, newTaskName)
	}
}

var stepTimeoutWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-step
spec:
  entrypoint: main
  templates:
  - name: main 
    steps: 
    - - name: step1
        template: whalesay

  - name: whalesay
    timeout: 5s
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

var dagTimeoutWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-dag
spec:
  entrypoint: main
  templates:
  - name: main 
    dag:
      tasks:
      - name: dag1
        template: whalesay
        arguments:
          parameters:
          - name: deadline
            value: 3s
      - name: dag2
        template: whalesay
        arguments:
          parameters:
          - name: deadline
            value: 3s
  - name: whalesay
    inputs:
      parameters:
      - name: deadline
    timeout: "{{inputs.parameters.deadline}}"
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

func TestTemplateTimeoutDuration(t *testing.T) {
	t.Run("Step Template Deadline", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(stepTimeoutWf)
		cancel, controller := newController(wf)
		defer cancel()

		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		time.Sleep(6 * time.Second)
		makePodsPhase(ctx, woc, apiv1.PodPending)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Nodes.FindByDisplayName("step1").Phase)
	})
	t.Run("DAG Template Deadline", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(dagTimeoutWf)
		cancel, controller := newController(wf)
		defer cancel()

		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		time.Sleep(6 * time.Second)
		makePodsPhase(ctx, woc, apiv1.PodPending)
		woc = newWorkflowOperationCtx(woc.wf, controller)
		woc.operate(ctx)

		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Nodes.FindByDisplayName("hello-world-dag").Phase)
	})
	t.Run("Invalid timeout format", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(stepTimeoutWf)
		tmpl := wf.Spec.Templates[1]
		tmpl.Timeout = "23"
		wf.Spec.Templates[1] = tmpl
		cancel, controller := newController(wf)
		defer cancel()

		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		jsonByte, err := json.Marshal(woc.wf)
		assert.NoError(t, err)
		assert.Contains(t, string(jsonByte), "has invalid duration format in timeout")
	})

	t.Run("Invalid timeout in step", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(stepTimeoutWf)
		tmpl := wf.Spec.Templates[0]
		tmpl.Timeout = "23"
		wf.Spec.Templates[0] = tmpl
		cancel, controller := newController(wf)
		defer cancel()

		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		jsonByte, err := json.Marshal(woc.wf)
		assert.NoError(t, err)
		assert.Contains(t, string(jsonByte), "doesn't support timeout field")
	})
}

var wfWithPVC = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: storage-quota-limit
spec:
  entrypoint: wait
  volumeClaimTemplates:                 # define volume, same syntax as k8s Pod spec
    - metadata:
        name: workdir1                     # name of volume claim
      spec:
        accessModes: [ "ReadWriteMany" ]
        resources:
          requests:
            storage: 10Gi
  templates:
  - name: wait
    script:
      image: argoproj/argosay:v2
      args: [echo, ":) Hello Argo!"]
`

func TestStorageQuota(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfWithPVC)

	cancel, controller := newController(wf)
	defer cancel()

	controller.kubeclientset.(*fake.Clientset).BatchV1().(*batchfake.FakeBatchV1).Fake.PrependReactor("create", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierr.NewForbidden(schema.GroupResource{Group: "test", Resource: "test1"}, "test", errors.New("exceeded quota"))
	})

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowPending, woc.wf.Status.Phase)
	assert.Contains(t, woc.wf.Status.Message, "Waiting for a PVC to be created.")

	controller.kubeclientset.(*fake.Clientset).BatchV1().(*batchfake.FakeBatchV1).Fake.PrependReactor("create", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierr.NewBadRequest("BadRequest")
	})

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Contains(t, woc.wf.Status.Message, "BadRequest")
}

var podWithFailed = `
apiVersion: v1
kind: Pod
status:
  conditions:
  - lastProbeTime: 
    lastTransitionTime: '2020-08-27T18:14:19Z'
    status: 'True'
    type: PodScheduled
  containerStatuses:
  - containerID: docker://502dda61a8f05e08d10cffc972d2fb9226e82af7daaacff98e84727bb96f11e6
    image: python:alpine3.6
    imageID: docker-pullable://python@sha256:766a961bf699491995cc29e20958ef11fd63741ff41dcc70ec34355b39d52971
    lastState:
      waiting: {}
    name: main
    ready: false
    restartCount: 0
    started: false
    state:
      waiting: {}
  - containerID: docker://d31f0d56f29b6962ef1493b2df6b7cdb54d48d8b8fa95d7e9c98ddc56f857b35
    image: argoproj/argoexec:v2.9.5
    imageID: docker-pullable://argoproj/argoexec@sha256:989114232892e051c25be323af626149452578d3ebbdc3e9ec7205bba3918d48
    lastState:
      waiting: {}
    name: wait
    ready: false
    restartCount: 0
    started: false
    state:
      waiting: {}
  hostIP: 192.168.65.3
  phase: Failed
  podIP: 10.1.28.244
  podIPs:
  - ip: 10.1.28.244
  qosClass: BestEffort
  startTime: '2020-08-27T18:14:19Z'
`

func TestPodFailureWithContainerWaitingState(t *testing.T) {
	var pod apiv1.Pod
	wfv1.MustUnmarshal(podWithFailed, &pod)
	assert.NotNil(t, pod)
	nodeStatus, msg := newWoc().inferFailedReason(&pod, nil)
	assert.Equal(t, wfv1.NodeError, nodeStatus)
	assert.Contains(t, msg, "Pod failed before")
}

var podWithWaitContainerOOM = `
apiVersion: v1
kind: Pod
status:
  containerStatuses:
  - containerID: containerd://765e8084b1c416d412c8072dca624cab886aae3858d1196b5aaceb7a775ce372
    image: docker.io/argoproj/argosay:v2
    imageID: docker.io/argoproj/argosay@sha256:3d2d553a462cfe3288833a010c1d91454bd05a0e02937f2f82050d68ca57a580
    lastState: {}
    name: main
    ready: false
    restartCount: 0
    started: false
    state:
      terminated:
        containerID: containerd://765e8084b1c416d412c8072dca624cab886aae3858d1196b5aaceb7a775ce372
        exitCode: 0
        finishedAt: "2021-01-22T09:50:17Z"
        reason: Completed
        startedAt: "2021-01-22T09:50:16Z"
  - containerID: containerd://12b93c7a73c7448a3034e63181ca9c8db8dbaf1d7d43dd5ad90c20814a757b51
    image: docker.io/argoproj/argoexec:latest
    imageID: sha256:54331d70b022d9610ba40826b1cfd77cc39b5e5b8a6b6b28a9a73db445a35436
    lastState: {}
    name: wait
    ready: false
    restartCount: 0
    started: false
    state:
      terminated:
        containerID: containerd://12b93c7a73c7448a3034e63181ca9c8db8dbaf1d7d43dd5ad90c20814a757b51
        exitCode: 137
        finishedAt: "2021-01-22T09:50:17Z"
        reason: OOMKilled
        startedAt: "2021-01-22T09:50:16Z"
  hostIP: 172.19.0.2
  phase: Failed
  podIP: 10.42.0.74
  podIPs:
  - ip: 10.42.0.74
  qosClass: Burstable
  startTime: "2021-01-22T09:50:15Z"
`

var podWithMainContainerOOM = `
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: main
    env:
    - name: ARGO_CONTAINER_NAME
      value: main
status:
  containerStatuses:
  - containerID: containerd://3e8c564c13893914ec81a2c105188fa5d34748576b368e709dbc2e71cbf23c5b
    image: docker.io/monitoringartist/docker-killer:latest
    imageID: docker.io/monitoringartist/docker-killer@sha256:85ba7f17a5ef691eb4a3dff7fdab406369085c6ee6e74dc4527db9fe9e448fa1
    lastState: {}
    name: main
    ready: false
    restartCount: 0
    started: false
    state:
      terminated:
        containerID: containerd://3e8c564c13893914ec81a2c105188fa5d34748576b368e709dbc2e71cbf23c5b
        exitCode: 137
        finishedAt: "2021-01-22T10:28:13Z"
        reason: OOMKilled
        startedAt: "2021-01-22T10:28:13Z"
  - containerID: containerd://0efb49b80c593396c5895a8bd062d9174f681d12436824246c273987c466b594
    image: docker.io/argoproj/argoexec:latest
    imageID: sha256:54331d70b022d9610ba40826b1cfd77cc39b5e5b8a6b6b28a9a73db445a35436
    lastState: {}
    name: wait
    ready: false
    restartCount: 0
    started: false
    state:
      terminated:
        containerID: containerd://0efb49b80c593396c5895a8bd062d9174f681d12436824246c273987c466b594
        exitCode: 0
        finishedAt: "2021-01-22T10:28:14Z"
        reason: Completed
        startedAt: "2021-01-22T10:28:12Z"
  hostIP: 172.19.0.2
  phase: Failed
  podIP: 10.42.0.87
  podIPs:
  - ip: 10.42.0.87
  qosClass: Burstable
  startTime: "2021-01-22T10:28:12Z"
`

func TestPodFailureWithContainerOOM(t *testing.T) {
	tests := []struct {
		podDetail string
		phase     wfv1.NodePhase
	}{{
		podDetail: podWithWaitContainerOOM,
		phase:     wfv1.NodeError,
	}, {
		podDetail: podWithMainContainerOOM,
		phase:     wfv1.NodeFailed,
	}}
	var pod apiv1.Pod
	for _, tt := range tests {
		wfv1.MustUnmarshal(tt.podDetail, &pod)
		assert.NotNil(t, pod)
		nodeStatus, msg := newWoc().inferFailedReason(&pod, nil)
		assert.Equal(t, tt.phase, nodeStatus)
		assert.Contains(t, msg, "OOMKilled")
	}
}

func TestResubmitPendingPods(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: my-wf
  namespace: my-ns
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: my-image
`)
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate(wftmplGlobalVarsOnExit)
	cancel, controller := newController(wf, wftmpl)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.True(t, woc.wf.Status.Nodes.Any(func(node wfv1.NodeStatus) bool {
		return node.Phase == wfv1.NodePending
	}))

	time.Sleep(time.Second)
	deletePods(ctx, woc)

	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.True(t, woc.wf.Status.Nodes.Any(func(node wfv1.NodeStatus) bool {
		return node.Phase == wfv1.NodePending
	}))

	time.Sleep(time.Second)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	woc = newWorkflowOperationCtx(woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

var wfRetryWithParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: parameter-aggregation
spec:
  entrypoint: parameter-aggregation
  templates:
  - name: parameter-aggregation
    steps:
    - - name: divide-by-2
        template: divide-by-2
        arguments:
          parameters:
          - name: num
            value: "{{item}}"
        withItems: [1,2,3]
    # Finally, print all numbers processed in the previous step
    - - name: print
        template: whalesay
        arguments:
          parameters:
          - name: message
            value: "{{item}}"
        withParam: "{{steps.divide-by-2.outputs.result}}"

  # divide-by-2 divides a number in half
  - name: divide-by-2
    retryStrategy: 
        limit: 2
        backoff: 
            duration: "1"
            factor: 2
    inputs:
      parameters:
      - name: num
    script:
      image: alpine:latest
      command: [sh, -x]
      source: |
        #!/bin/sh
        echo $(({{inputs.parameters.num}}/2))
  # whalesay prints a number using whalesay
  - name: whalesay
    retryStrategy: 
        limit: 2
        backoff: 
            duration: "1"
            factor: 2
    inputs:
      parameters:
      - name: message
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{inputs.parameters.message}}"]
`

func TestWFWithRetryAndWithParam(t *testing.T) {
	t.Run("IncludeScriptOutputInRetryAndWithParam", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(wfRetryWithParam)
		cancel, controller := newController(wf)
		defer cancel()

		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)
		pods, err := listPods(woc)
		assert.NoError(t, err)
		assert.True(t, len(pods.Items) > 0)
		if assert.Len(t, pods.Items, 3) {
			ctrs := pods.Items[0].Spec.Containers
			assert.Len(t, ctrs, 2)
			envs := ctrs[1].Env
			assert.Len(t, envs, 8)
			assert.Equal(t, apiv1.EnvVar{Name: "ARGO_INCLUDE_SCRIPT_OUTPUT", Value: "true"}, envs[3])
		}
	})
}

var paramAggregation = `
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
      - arguments:
          parameters:
          - name: message
            value: '{{tasks.odd-or-even.outputs.parameters.num}}'
        dependencies:
        - odd-or-even
        name: print-nums
        template: whalesay
      - arguments:
          parameters:
          - name: message
            value: '{{tasks.odd-or-even.outputs.parameters.evenness}}'
        dependencies:
        - odd-or-even
        name: print-evenness
        template: whalesay
    inputs: {}
    metadata: {}
    name: parameter-aggregation
    outputs: {}
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
      image: alpine:latest
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
  - 
    container:
      args:
      - '{{inputs.parameters.message}}'
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
      resources: {}
    inputs:
      parameters:
      - name: message
    metadata: {}
    name: whalesay
    outputs: {}
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
      children:
      - parameter-aggregation-dag-h8b82-2243926302
      - parameter-aggregation-dag-h8b82-3175470584
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
      children:
      - parameter-aggregation-dag-h8b82-2243926302
      - parameter-aggregation-dag-h8b82-3175470584
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

func TestParamAggregation(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(paramAggregation)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	evenNode := woc.wf.Status.Nodes.FindByDisplayName("print-evenness")
	if assert.NotNil(t, evenNode) {
		if assert.Len(t, evenNode.Inputs.Parameters, 1) {
			assert.Equal(t, `["odd","even"]`, evenNode.Inputs.Parameters[0].Value.String())
		}
	}

	numNode := woc.wf.Status.Nodes.FindByDisplayName("print-nums")
	if assert.NotNil(t, numNode) {
		if assert.Len(t, numNode.Inputs.Parameters, 1) {
			assert.Equal(t, `["1","2"]`, numNode.Inputs.Parameters[0].Value.String())
		}
	}
}

func TestPodHasContainerNeedingTermination(t *testing.T) {
	pod := apiv1.Pod{
		Status: apiv1.PodStatus{
			ContainerStatuses: []apiv1.ContainerStatus{
				{
					Name:  common.WaitContainerName,
					State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}},
				},
				{
					Name:  common.MainContainerName,
					State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}},
				},
			}}}
	tmpl := wfv1.Template{}
	assert.True(t, podHasContainerNeedingTermination(&pod, tmpl))

	pod = apiv1.Pod{
		Status: apiv1.PodStatus{
			ContainerStatuses: []apiv1.ContainerStatus{
				{
					Name:  common.WaitContainerName,
					State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
				},
				{
					Name:  common.MainContainerName,
					State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}},
				},
			}}}
	assert.False(t, podHasContainerNeedingTermination(&pod, tmpl))

	pod = apiv1.Pod{
		Status: apiv1.PodStatus{
			ContainerStatuses: []apiv1.ContainerStatus{
				{
					Name:  common.WaitContainerName,
					State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}},
				},
				{
					Name:  common.MainContainerName,
					State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
				},
			}}}
	assert.False(t, podHasContainerNeedingTermination(&pod, tmpl))

	pod = apiv1.Pod{
		Status: apiv1.PodStatus{
			ContainerStatuses: []apiv1.ContainerStatus{
				{
					Name:  common.MainContainerName,
					State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
				},
			}}}
	assert.False(t, podHasContainerNeedingTermination(&pod, tmpl))

	pod = apiv1.Pod{
		Status: apiv1.PodStatus{
			ContainerStatuses: []apiv1.ContainerStatus{
				{
					Name:  common.MainContainerName,
					State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}},
				},
			}}}
	assert.True(t, podHasContainerNeedingTermination(&pod, tmpl))
}

func TestRetryOnDiffHost(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	woc := newWorkflowOperationCtx(wf, controller)
	// Verify that there are no nodes in the wf status.
	assert.Empty(t, woc.wf.Status.Nodes)

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	node := woc.initializeNode(nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)

	hostSelector := "kubernetes.io/hostname"
	retries := wfv1.RetryStrategy{}
	retries.Affinity = &wfv1.RetryAffinity{
		NodeAntiAffinity: &wfv1.RetryNodeAntiAffinity{},
	}

	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, node.Phase, wfv1.NodeRunning)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child node.
	childNode := fmt.Sprintf("child-node-%d", 0)
	woc.initializeNode(childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning)
	woc.addChildNode(nodeName, childNode)

	n := woc.wf.GetNodeByName(nodeName)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	woc.markNodePhase(lastChild.Name, wfv1.NodeFailed)
	_, _, err := woc.processNodeRetries(n, retries, &executeTemplateOpts{})
	assert.NoError(t, err)
	n = woc.wf.GetNodeByName(nodeName)
	assert.Equal(t, n.Phase, wfv1.NodeRunning)

	// Ensure related fields are not set
	assert.Equal(t, lastChild.HostNodeName, "")

	// Set host name
	n = woc.wf.GetNodeByName(nodeName)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	lastChild.HostNodeName = "test-fail-hostname"
	woc.wf.Status.Nodes[lastChild.ID] = *lastChild

	pod := &apiv1.Pod{
		Spec: apiv1.PodSpec{
			Affinity: &apiv1.Affinity{},
		},
	}

	tmpl := &wfv1.Template{}
	tmpl.RetryStrategy = &retries

	RetryOnDifferentHost(nodeID)(*woc.retryStrategy(tmpl), woc.wf.Status.Nodes, pod)
	assert.NotNil(t, pod.Spec.Affinity)

	// Verify if template's Affinity has the right value
	targetNodeSelectorRequirement :=
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
	sourceNodeSelectorRequirement := apiv1.NodeSelectorRequirement{
		Key:      hostSelector,
		Operator: apiv1.NodeSelectorOpNotIn,
		Values:   []string{lastChild.HostNodeName},
	}
	assert.Equal(t, sourceNodeSelectorRequirement, targetNodeSelectorRequirement)
}

var noPodsWhenShutdown = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: whalesay
  shutdown: "Stop"
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`

func TestNoPodsWhenShutdown(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(noPodsWhenShutdown)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.FindByDisplayName("hello-world")
	if assert.NotNil(t, node) {
		assert.Equal(t, wfv1.NodeSkipped, node.Phase)
		assert.Contains(t, node.Message, "workflow shutdown with strategy: Stop")
	}
}

var wfscheVariable = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
  annotations:
    workflows.argoproj.io/scheduled-time: 2006-01-02T15:04:05-07:00
spec:
  entrypoint: whalesay
  shutdown: "Stop"
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{workflows.scheduledTime}}"]
`

func TestWorkflowScheduledTimeVariable(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfscheVariable)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, "2006-01-02T15:04:05-07:00", woc.globalParams[common.GlobalVarWorkflowCronScheduleTime])
}

func TestWorkflowShutdownStrategy(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: whalesay
  namespace: default
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [sh, -c]
      args: ["cowsay hellow"]`)

	wf1 := wf.DeepCopy()
	wf1.Name = "whalesay-1"
	cancel, controller := newController(wf, wf1)
	defer cancel()
	t.Run("StopStrategy", func(t *testing.T) {
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)
		woc.operate(ctx)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		// Simulate the Stop command
		wf1 := woc.wf
		wf1.Spec.Shutdown = wfv1.ShutdownStrategyStop
		woc1 := newWorkflowOperationCtx(wf1, controller)
		woc1.operate(ctx)

		node := woc1.wf.Status.Nodes.FindByDisplayName("whalesay")
		if assert.NotNil(t, node) {
			assert.Contains(t, node.Message, "workflow shutdown with strategy")
			assert.Contains(t, node.Message, "Stop")
		}
	})

	t.Run("TerminateStrategy", func(t *testing.T) {
		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf1, controller)
		woc.operate(ctx)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		// Simulate the Terminate command
		wfOut := woc.wf
		wfOut.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
		woc1 := newWorkflowOperationCtx(wfOut, controller)
		woc1.operate(ctx)
		for _, node := range woc1.wf.Status.Nodes {
			if assert.NotNil(t, node) {
				assert.Contains(t, node.Message, "workflow shutdown with strategy")
				assert.Contains(t, node.Message, "Terminate")
			}
		}
	})
}

const resultVarRefWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: scripts-bash-
spec:
  entrypoint: bash-script-example
  templates:
  - name: bash-script-example
    steps:
    - - name: generate-random
        template: gen-random-int
    - - name: generate-random-1
        template: gen-random-int
    - - name: from
        template: print-message
        arguments:
          parameters:
          - name: message
            value: "{{steps.generate-random.outputs.result}}"
    outputs:
      parameters:
        - name: stepresult
          valueFrom:
            expression: "steps['generate-random-1'].outputs.result"

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

func TestHasOutputResultRef(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(resultVarRefWf)
	assert.True(t, hasOutputResultRef("generate-random", &wf.Spec.Templates[0]))
	assert.True(t, hasOutputResultRef("generate-random-1", &wf.Spec.Templates[0]))
}

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

func TestGetStepOrDAGTaskName(t *testing.T) {
	assert.Equal(t, "generate-artifact", getStepOrDAGTaskName("data-transformation-gjrt8[0].generate-artifact(2:foo/script.py)"))
	assert.Equal(t, "generate-artifact", getStepOrDAGTaskName("data-transformation-gjrt8[0].generate-artifact(2:foo/scrip[t.py)"))
	assert.Equal(t, "generate-artifact", getStepOrDAGTaskName("data-transformation-gjrt8[0].generate-artifact(2:foo/scrip]t.py)"))
	assert.Equal(t, "generate-artifact", getStepOrDAGTaskName("data-transformation-gjrt8[0].generate-artifact(2:foo/scri[p]t.py)"))
	assert.Equal(t, "generate-artifact", getStepOrDAGTaskName("data-transformation-gjrt8[0].generate-artifact(2:foo/script.py)"))
	assert.Equal(t, "step3", getStepOrDAGTaskName("bug-rqq5f[0].fanout[0].fanout1(0:1)(0)[0].fanout2(0:1).step3(0)"))
	assert.Equal(t, "divide-by-2", getStepOrDAGTaskName("parameter-aggregation[0].divide-by-2(0:1)(0)"))
	assert.Equal(t, "hello-mate", getStepOrDAGTaskName("greet-many-tkcld.greet-many(0:1).greet.hello-mate"))
	assert.Equal(t, "hello-mate", getStepOrDAGTaskName("greet.hello-mate"))
	assert.Equal(t, "hello-mate", getStepOrDAGTaskName("hello-mate"))
	assert.Equal(t, "fanout1", getStepOrDAGTaskName("bug-rqq5f[0].fanout[0].fanout1(0:1)(0)[0]"))
}

func TestGenerateOutputResultRegex(t *testing.T) {
	dagTmpl := &wfv1.Template{DAG: &wfv1.DAGTemplate{}}
	ref, expr := generateOutputResultRegex("template-name", dagTmpl)
	assert.Equal(t, `tasks\.template-name\.outputs\.result`, ref)
	assert.Equal(t, `tasks\[['\"]template-name['\"]\]\.outputs.result`, expr)

	stepsTmpl := &wfv1.Template{Steps: []wfv1.ParallelSteps{}}
	ref, expr = generateOutputResultRegex("template-name", stepsTmpl)
	assert.Equal(t, `steps\.template-name\.outputs\.result`, ref)
	assert.Equal(t, `steps\[['\"]template-name['\"]\]\.outputs.result`, expr)
}

const rootRetryStrategyCompletes = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-5bd7v
spec:
  activeDeadlineSeconds: 300
  entrypoint: whalesay
  retryStrategy:
    limit: 1
  templates:
  - container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
    name: whalesay
  ttlStrategy:
    secondsAfterCompletion: 600
status:
  nodes:
    hello-world-5bd7v:
      children:
      - hello-world-5bd7v-643409622
      displayName: hello-world-5bd7v
      finishedAt: "2021-03-23T14:53:45Z"
      id: hello-world-5bd7v
      name: hello-world-5bd7v
      phase: Succeeded
      startedAt: "2021-03-23T14:53:39Z"
      templateName: whalesay
      templateScope: local/hello-world-5bd7v
      type: Retry
    hello-world-5bd7v-643409622:
      displayName: hello-world-5bd7v(0)
      finishedAt: "2021-03-23T14:53:44Z"
      hostNodeName: k3d-k3s-default-server-0
      id: hello-world-5bd7v-643409622
      name: hello-world-5bd7v(0)
      phase: Succeeded
      startedAt: "2021-03-23T14:53:39Z"
      templateName: whalesay
      templateScope: local/hello-world-5bd7v
      type: Pod
  phase: Running
  startedAt: "2021-03-23T14:53:39Z"
`

func TestRootRetryStrategyCompletes(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(rootRetryStrategyCompletes)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

const testGlobalParamSubstitute = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-diamond-8xw8l
spec:
  entrypoint: "whalesay1"
  arguments:
    parameters:
    - name: entrypoint
      value: test
    - name: mutex
      value: mutex1
    - name: message
      value: mutex1
  synchronization:
    mutex:
      name:  "{{workflow.parameters.mutex}}"
  templates:
  - name: whalesay1
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["{{workflow.parameters.message}}"]
`

func TestSubstituteGlobalVariables(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testGlobalParamSubstitute)
	cancel, controller := newController(wf)
	defer cancel()

	// ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	err := woc.setExecWorkflow(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, woc.execWf)
	assert.Equal(t, "mutex1", woc.execWf.Spec.Synchronization.Mutex.Name)
	tempStr, err := json.Marshal(woc.execWf.Spec.Templates)
	assert.NoError(t, err)
	assert.Contains(t, string(tempStr), "{{workflow.parameters.message}}")
}

var wfPending = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  creationTimestamp: "2021-04-05T21:50:18Z"
  name: hello-world-4srt7
  namespace: argo
spec:
  entrypoint: whalesay
  podSpecPatch: |
    terminationGracePeriodSeconds: 3
  templates:
  - container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
      name: ""
    name: whalesay

status:
  artifactRepositoryRef:
    configMap: artifact-repositories
    key: default-v1
    namespace: argo
  finishedAt: null
  nodes:
    hello-world-4srt7:
      displayName: hello-world-4srt7
      finishedAt: null
      id: hello-world-4srt7
      name: hello-world-4srt7
      phase: Pending
      progress: 0/1
      startedAt: "2021-04-05T21:50:18Z"
      templateName: whalesay
      templateScope: local/hello-world-4srt7
      type: Pod
  phase: Running
  progress: 0/1
  startedAt: "2021-04-05T21:50:18Z"
`

func TestWfPendingWithNoPod(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfPending)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(woc)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(pods.Items))

}

var wfPendingWithSync = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world-mpdht
  namespace: argo
spec:
  entrypoint: whalesay
  templates:
  - container:
      args:
      - hello world
      command:
      - cowsay
      image: docker/whalesay:latest
    name: whalesay
    synchronization:
      mutex:
        name: welcome
  ttlStrategy:
    secondsAfterCompletion: 600
status:
  nodes:
    hello-world-mpdht:
      displayName: hello-world-mpdht
      finishedAt: null
      id: hello-world-mpdht
      message: 'Waiting for argo/Mutex/welcome lock. Lock status: 0/1 '
      name: hello-world-mpdht
      phase: Pending
      progress: 0/1
      startedAt: "2021-04-05T22:11:21Z"
      synchronizationStatus:
        waiting: argo/Mutex/welcome
      templateName: whalesay
      templateScope: local/hello-world-mpdht
      type: Pod
  phase: Running
  progress: 0/1
  startedAt: "2021-04-05T22:11:21Z"
  synchronization:
    mutex:
      waiting:
      - holder: argo/hello-world-tmph8/hello-world-tmph8
        mutex: argo/Mutex/welcome
`

func TestMutexWfPendingWithNoPod(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfPendingWithSync)
	cancel, controller := newController(wf)
	defer cancel()
	ctx := context.Background()
	controller.syncManager = sync.NewLockManager(GetSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc)
	_, _, _, err := controller.syncManager.TryAcquire(wf, "test", &wfv1.Synchronization{Mutex: &wfv1.Mutex{Name: "welcome"}})
	assert.NoError(t, err)
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.NodePending, woc.wf.Status.Nodes.FindByDisplayName("hello-world-mpdht").Phase)
}

var wfGlopalArtifactNil = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: global-outputs-ttsfq
  namespace: default
spec:
  entrypoint: generate-globals
  onExit: consume-globals
  templates:
  - name: generate-globals
    steps:
    - - name: generate
        template: global-output
    - - name: consume-globals
        template: consume-globals
  - container:
      args:
      - sleep 1; exit 1
      command:
      - sh
      - -c
      image: alpine:3.7
    name: global-output
    outputs:
      artifacts:
      - globalName: my-global-art
        name: hello-art
        path: /tmp/hello_world.txt
      parameters:
      - globalName: my-global-param
        name: hello-param
        valueFrom:
          path: /tmp/hello_world.txt
  - name: consume-globals
    steps:
    - - name: consume-global-param
        template: consume-global-param
      - arguments:
          artifacts:
          - from: '{{workflow.outputs.artifacts.my-global-art}}'
            name: art
        name: consume-global-art
        template: consume-global-art
  - container:
      args:
      - echo {{inputs.parameters.param}}
      command:
      - sh
      - -c
      image: alpine:3.7
    inputs:
      parameters:
      - name: param
        value: '{{workflow.outputs.parameters.my-global-param}}'
    name: consume-global-param
  - container:
      args:
      - cat /art
      command:
      - sh
      - -c
      image: alpine:3.7
    inputs:
      artifacts:
      - name: art
        path: /art
    name: consume-global-art
`

func TestWFGlobalArtifactNil(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(wfGlopalArtifactNil)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed, func(pod *apiv1.Pod) {
		pod.Annotations[common.AnnotationKeyOutputs] = string("{\"parameters\":[{\"name\":\"hello-param\",\"valueFrom\":{\"path\":\"/tmp/hello_world.txt\"},\"globalName\":\"my-global-param\"}],\"artifacts\":[{\"name\":\"hello-art\",\"path\":\"/tmp/hello_world.txt\",\"globalName\":\"my-global-art\"}]}")
		pod.Status.ContainerStatuses = []apiv1.ContainerStatus{
			{
				Name: "main",
				State: apiv1.ContainerState{
					Terminated: &apiv1.ContainerStateTerminated{
						ExitCode: 1,
						Message:  "",
					},
				},
			},
		}
	})
	woc.operate(ctx)
	assert.NotPanics(t, func() { woc.operate(ctx) })
}

const testDagTwoChildrenWithNonExpectedNodeType = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: ingest-pipeline-cdw-m2fnc
spec:
  arguments:
    parameters:
    - name: job_name
      value: all_the_data
  entrypoint: ingest-task
  templates:
  - dag:
      tasks:
      - dependencies:
        - sent
        name: ing
        template: ingest-collections
      - dependencies:
        - sent
        name: mat
        template: materializations
      - arguments:
          parameters:
          - name: job_name
            value: all_the_data
        name: sent
        template: sentinel
    name: ingest-task
  - container:
      args:
      - sleep 30; date; echo got sentinel for {{inputs.parameters.job_name}}
      command:
      - sh
      - -c
      image: alpine:3.13.5
    inputs:
      parameters:
      - name: job_name
    name: sentinel
  - name: ingest-collections
    steps:
    - - name: get-ingest-collections
        template: get-ingest-collections
  - name: get-ingest-collections
    script:
      command:
      - python
      image: python:alpine3.6
      source: |
        import json
  - name: materializations
    steps:
    - - name: get-materializations
        template: get-materializations
  - name: get-materializations
    script:
      command:
      - python
      image: python:alpine3.6
      name: ""
      resources: {}
      source: |
        import json
status:
  nodes:
    ingest-pipeline-cdw-m2fnc:
      children:
      - ingest-pipeline-cdw-m2fnc-141178578
      displayName: ingest-pipeline-cdw-m2fnc
      id: ingest-pipeline-cdw-m2fnc
      name: ingest-pipeline-cdw-m2fnc
      phase: Running
      startedAt: "2021-06-22T18:51:02Z"
      templateName: ingest-task
      templateScope: local/ingest-pipeline-cdw-m2fnc
      type: DAG
    ingest-pipeline-cdw-m2fnc-141178578:
      boundaryID: ingest-pipeline-cdw-m2fnc
      displayName: sent
      finishedAt: "2021-06-22T18:51:34Z"
      hostNodeName: k3d-k3s-default-server-0
      id: ingest-pipeline-cdw-m2fnc-141178578
      name: ingest-pipeline-cdw-m2fnc.sent
      phase: Succeeded
      startedAt: "2021-06-22T18:51:02Z"
      templateName: sentinel
      templateScope: local/ingest-pipeline-cdw-m2fnc
      type: Pod
  phase: Running
  startedAt: "2021-06-22T18:51:02Z"
`

func TestDagTwoChildrenWithNonExpectedNodeType(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testDagTwoChildrenWithNonExpectedNodeType)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

	sentNode := woc.wf.Status.Nodes.FindByDisplayName("sent")

	//Ensure that both child tasks are labeled as children of the "sent" node
	assert.Len(t, sentNode.Children, 2)
}

const testDagTwoChildrenContainerSet = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: outputs-result-pn6gb
spec:
  entrypoint: main
  templates:
  - dag:
      tasks:
      - name: a
        template: group
      - arguments:
          parameters:
          - name: x
            value: '{{tasks.a.outputs.result}}'
        depends: a
        name: b
        template: verify
    name: main
  - containerSet:
      containers:
      - args:
        - -c
        - |
          print("hi")
        image: python:alpine3.6
        name: main
    name: group
  - inputs:
      parameters:
      - name: x
    name: verify
    script:
      image: python:alpine3.6
      source: |
        assert "{{inputs.parameters.x}}" == "hi"
status:
  phase: Running
  startedAt: "2021-06-24T18:05:35Z"
`

// In this test, a pod originating from a container set should not be its own outbound node. "a" should only have one child
// and "main" should be the outbound node.
func TestDagTwoChildrenContainerSet(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(testDagTwoChildrenContainerSet)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	woc.operate(ctx)

	sentNode := woc.wf.Status.Nodes.FindByDisplayName("a")

	assert.Len(t, sentNode.Children, 1)
}

const operatorRetryExpression = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-script-9z9pv
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: safe-to-retry
        template: safe-to-retry
    - - arguments:
          parameters:
          - name: safe-to-retry
            value: '{{steps.safe-to-retry.outputs.result}}'
        name: retry
        template: retry-script
  - name: safe-to-retry
    script:
      command:
      - python
      image: python:alpine3.6
      source: |
        print("true")
  - inputs:
      parameters:
      - name: safe-to-retry
    name: retry-script
    retryStrategy:
      expression: asInt(lastRetry.exitCode) > 1 && {{inputs.parameters.safe-to-retry}}
        == true
      limit: "3"
    script:
      command:
      - python
      image: python:alpine3.6
      source: |
        import random;
        import sys;
        exit_code = random.choice([1, 2]);
        sys.exit(exit_code)
status:
  nodes:
    retry-script-9z9pv:
      children:
      - retry-script-9z9pv-1740877928
      displayName: retry-script-9z9pv
      id: retry-script-9z9pv
      name: retry-script-9z9pv
      outboundNodes:
      - retry-script-9z9pv-2327053777
      phase: Running
      startedAt: "2021-06-10T22:28:49Z"
      templateName: main
      templateScope: local/retry-script-9z9pv
      type: Steps
    retry-script-9z9pv-734073693:
      boundaryID: retry-script-9z9pv
      children:
      - retry-script-9z9pv-2346402485
      displayName: '[1]'
      id: retry-script-9z9pv-734073693
      name: retry-script-9z9pv[1]
      phase: Running
      startedAt: "2021-06-10T22:28:56Z"
      templateScope: local/retry-script-9z9pv
      type: StepGroup
    retry-script-9z9pv-1740877928:
      boundaryID: retry-script-9z9pv
      children:
      - retry-script-9z9pv-3940097040
      displayName: '[0]'
      finishedAt: "2021-06-10T22:28:56Z"
      id: retry-script-9z9pv-1740877928
      name: retry-script-9z9pv[0]
      phase: Succeeded
      startedAt: "2021-06-10T22:28:49Z"
      templateScope: local/retry-script-9z9pv
      type: StepGroup
    retry-script-9z9pv-2327053777:
      boundaryID: retry-script-9z9pv
      displayName: retry(1)
      finishedAt: "2021-06-10T22:29:10Z"
      id: retry-script-9z9pv-2327053777
      inputs:
        parameters:
        - name: safe-to-retry
          value: "true"
      message: Error (exit code 1)
      name: retry-script-9z9pv[1].retry(1)
      outputs:
        exitCode: "1"
      phase: Failed
      startedAt: "2021-06-10T22:29:04Z"
      templateName: retry-script
      templateScope: local/retry-script-9z9pv
      type: Pod
    retry-script-9z9pv-2346402485:
      boundaryID: retry-script-9z9pv
      children:
      - retry-script-9z9pv-2931195156
      - retry-script-9z9pv-2327053777
      displayName: retry
      id: retry-script-9z9pv-2346402485
      inputs:
        parameters:
        - name: safe-to-retry
          value: "true"
      name: retry-script-9z9pv[1].retry
      phase: Running
      startedAt: "2021-06-10T22:28:56Z"
      templateName: retry-script
      templateScope: local/retry-script-9z9pv
      type: Retry
    retry-script-9z9pv-2931195156:
      boundaryID: retry-script-9z9pv
      displayName: retry(0)
      finishedAt: "2021-06-10T22:29:02Z"
      id: retry-script-9z9pv-2931195156
      inputs:
        parameters:
        - name: safe-to-retry
          value: "true"
      message: Error (exit code 2)
      name: retry-script-9z9pv[1].retry(0)
      outputs:
        exitCode: "2"
      phase: Failed
      startedAt: "2021-06-10T22:28:56Z"
      templateName: retry-script
      templateScope: local/retry-script-9z9pv
      type: Pod
    retry-script-9z9pv-3940097040:
      boundaryID: retry-script-9z9pv
      children:
      - retry-script-9z9pv-734073693
      displayName: safe-to-retry
      finishedAt: "2021-06-10T22:28:55Z"
      id: retry-script-9z9pv-3940097040
      name: retry-script-9z9pv[0].safe-to-retry
      outputs:
        exitCode: "0"
        result: "true"
      phase: Succeeded
      startedAt: "2021-06-10T22:28:49Z"
      templateName: safe-to-retry
      templateScope: local/retry-script-9z9pv
      type: Pod
  phase: Running
  startedAt: "2021-06-10T22:28:49Z"
`

// TestOperatorRetryExpression tests that retryStrategy.when works correctly. In this test, the latest child node has
// just failed with exit code 2. The retryStrategy's when condition specifies that retries must only be done when the
// last exit code is NOT 2. We expect the retryStrategy to fail (even though it has 8 tries remainng).
func TestOperatorRetryExpression(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(operatorRetryExpression)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	retryNode := woc.wf.GetNodeByName("retry-script-9z9pv[1].retry")
	assert.Equal(t, wfv1.NodeFailed, retryNode.Phase)
	assert.Equal(t, 2, len(retryNode.Children))
	assert.Equal(t, "retryStrategy.expression evaluated to false", retryNode.Message)
}

func TestBuildRetryStrategyLocalScope(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(operatorRetryExpression)
	retryNode := wf.GetNodeByName("retry-script-9z9pv[1].retry")

	localScope := buildRetryStrategyLocalScope(retryNode, wf.Status.Nodes)

	assert.Len(t, localScope, 4)
	assert.Equal(t, "1", localScope[common.LocalVarRetries])
	assert.Equal(t, "1", localScope[common.LocalVarRetriesLastExitCode])
	assert.Equal(t, string(wfv1.NodeFailed), localScope[common.LocalVarRetriesLastStatus])
	assert.Equal(t, "6", localScope[common.LocalVarRetriesLastDuration])
}

var exitHandlerWithRetryNodeParam = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: exit-handler-with-param-xbh52
  namespace: argo
spec:
  arguments: {}
  entrypoint: main
  serviceAccountName: argo
  templates:
  - inputs: {}
    metadata: {}
    name: main
    outputs: {}
    steps:
    - - arguments: {}
        hooks:
          exit:
            arguments:
              parameters:
              - name: message
                value: '{{steps.step-1.outputs.parameters.result}}'
            template: exit
        name: step-1
        template: output
  - container:
      args:
      - echo -n hello world > /tmp/hello_world.txt; exit 1
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
    inputs: {}
    metadata: {}
    name: output
    outputs:
      parameters:
      - name: result
        valueFrom:
          default: Foobar
          path: /tmp/hello_world.txt
    retryStrategy:
      backoff:
        duration: 1s
      limit: "1"
      retryPolicy: Always
  - inputs:
      parameters:
      - name: message
        value: GoodValue
    metadata: {}
    name: exit
    outputs: {}
    script:
      args:
      - echo {{inputs.parameters.message}}
      command:
      - sh
      - -c
      image: alpine:latest
      name: ""
      resources: {}
      source: ""
status:
  nodes:
    exit-handler-with-param-xbh52:
      children:
      - exit-handler-with-param-xbh52-3621967439
      displayName: exit-handler-with-param-xbh52
      finishedAt: "2021-10-18T03:28:14Z"
      id: exit-handler-with-param-xbh52
      name: exit-handler-with-param-xbh52
      startedAt: "2021-10-18T03:27:44Z"
      templateName: main
      templateScope: local/exit-handler-with-param-xbh52
      type: Steps
    exit-handler-with-param-xbh52-1429999455:
      boundaryID: exit-handler-with-param-xbh52
      displayName: step-1(1)
      finishedAt: "2021-10-18T03:27:58Z"
      hostNodeName: smile
      id: exit-handler-with-param-xbh52-1429999455
      message: Error (exit code 1)
      name: exit-handler-with-param-xbh52[0].step-1(1)
      outputs:
        exitCode: "1"
        parameters:
        - name: result
          value: hello world
          valueFrom:
            default: Foobar
            path: /tmp/hello_world.txt
      phase: Failed
      progress: 1/1
      resourcesDuration:
        cpu: 5
        memory: 5
      startedAt: "2021-10-18T03:27:54Z"
      templateName: output
      templateScope: local/exit-handler-with-param-xbh52
      type: Pod
    exit-handler-with-param-xbh52-2034140834:
      boundaryID: exit-handler-with-param-xbh52
      displayName: step-1(0)
      finishedAt: "2021-10-18T03:27:48Z"
      hostNodeName: smile
      id: exit-handler-with-param-xbh52-2034140834
      message: Error (exit code 1)
      name: exit-handler-with-param-xbh52[0].step-1(0)
      outputs:
        exitCode: "1"
        parameters:
        - name: result
          value: hello world
          valueFrom:
            default: Foobar
            path: /tmp/hello_world.txt
      phase: Failed
      progress: 1/1
      resourcesDuration:
        cpu: 5
        memory: 5
      startedAt: "2021-10-18T03:27:44Z"
      templateName: output
      templateScope: local/exit-handler-with-param-xbh52
      type: Pod
    exit-handler-with-param-xbh52-3203867295:
      boundaryID: exit-handler-with-param-xbh52
      children:
      - exit-handler-with-param-xbh52-2034140834
      - exit-handler-with-param-xbh52-1429999455
      displayName: step-1
      finishedAt: "2021-10-18T03:28:04Z"
      id: exit-handler-with-param-xbh52-3203867295
      message: No more retries left
      name: exit-handler-with-param-xbh52[0].step-1
      startedAt: "2021-10-18T03:27:44Z"
      templateName: output
      templateScope: local/exit-handler-with-param-xbh52
      type: Retry
    exit-handler-with-param-xbh52-3621967439:
      boundaryID: exit-handler-with-param-xbh52
      children:
      - exit-handler-with-param-xbh52-3203867295
      displayName: '[0]'
      finishedAt: "2021-10-18T03:28:14Z"
      id: exit-handler-with-param-xbh52-3621967439
      message: child 'exit-handler-with-param-xbh52-3203867295' failed
      name: exit-handler-with-param-xbh52[0]
      startedAt: "2021-10-18T03:27:44Z"
      templateScope: local/exit-handler-with-param-xbh52
      type: StepGroup
  startedAt: "2021-10-18T03:27:44Z"
`

func TestExitHandlerWithRetryNodeParam(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(exitHandlerWithRetryNodeParam)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	retryStepNode := woc.wf.GetNodeByName("exit-handler-with-param-xbh52[0].step-1")
	assert.Equal(t, 1, len(retryStepNode.Outputs.Parameters))
	assert.Equal(t, "hello world", retryStepNode.Outputs.Parameters[0].Value.String())
	onExitNode := woc.wf.GetNodeByName("exit-handler-with-param-xbh52[0].step-1.onExit")
	assert.Equal(t, "hello world", onExitNode.Inputs.Parameters[0].Value.String())
}

func TestReOperateCompletedWf(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: my-wf
  namespace: my-ns
spec:
  entrypoint: main
  templates:
   - name: main
     dag:
       tasks:
       - name: pod
         template: pod
   - name: pod
     container: 
       image: my-image
`)
	wf.Status.Phase = wfv1.WorkflowError
	wf.Status.FinishedAt = metav1.Now()
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)
	assert.NotPanics(t, func() { woc.operate(ctx) })
}

func TestSetWFPodNamesAnnotation(t *testing.T) {
	defer func() {
		_ = os.Unsetenv("POD_NAMES")
	}()

	tests := []struct {
		podNameVersion string
	}{
		{"v1"},
		{"v2"},
	}

	for _, tt := range tests {
		_ = os.Setenv("POD_NAMES", tt.podNameVersion)

		wf := wfv1.MustUnmarshalWorkflow(exitHandlerWithRetryNodeParam)
		cancel, controller := newController(wf)
		defer cancel()

		ctx := context.Background()
		woc := newWorkflowOperationCtx(wf, controller)

		woc.operate(ctx)
		annotations := woc.wf.ObjectMeta.GetAnnotations()
		assert.Equal(t, annotations[common.AnnotationKeyPodNameVersion], tt.podNameVersion)
	}
}

var RetryLoopWithOutputParam = `
metadata:
  name: hr-retry-replication
  namespace: argo
  uid: a0edb47a-3e6d-4568-b828-bf0cfcd8e5d5
  resourceVersion: '258409'
  generation: 21
  creationTimestamp: '2022-01-06T00:09:56Z'
  labels:
    app.kubernetes.io/managed-by: Helm
    workflows.argoproj.io/completed: 'true'
    workflows.argoproj.io/phase: Succeeded
  annotations:
    meta.helm.sh/release-name: hr-retry-replication
    meta.helm.sh/release-namespace: argo
    workflows.argoproj.io/pod-name-format: v1
  managedFields:
    - manager: Go-http-client
      operation: Update
      apiVersion: argoproj.io/v1alpha1
      fieldsType: FieldsV1
      fieldsV1:
        'f:spec':
          'f:entrypoint': {}
          'f:templates': {}
    - manager: argo
      operation: Update
      apiVersion: argoproj.io/v1alpha1
      time: '2022-01-06T00:09:56Z'
      fieldsType: FieldsV1
      fieldsV1:
        'f:metadata':
          'f:annotations':
            .: {}
            'f:meta.helm.sh/release-name': {}
            'f:meta.helm.sh/release-namespace': {}
          'f:labels':
            .: {}
            'f:app.kubernetes.io/managed-by': {}
    - manager: workflow-controller
      operation: Update
      apiVersion: argoproj.io/v1alpha1
      time: '2022-01-06T00:10:17Z'
      fieldsType: FieldsV1
      fieldsV1:
        'f:metadata':
          'f:annotations':
            'f:workflows.argoproj.io/pod-name-format': {}
          'f:labels':
            'f:workflows.argoproj.io/completed': {}
            'f:workflows.argoproj.io/phase': {}
        'f:spec': {}
        'f:status': {}
spec:
  templates:
    - name: hr-retry-replication
      inputs: {}
      outputs: {}
      metadata: {}
      dag:
        tasks:
          - name: Create-Json-List
            template: createJsonList
            arguments: {}
          - name: Retry-And-Loop
            template: retryAndLoop
            arguments:
              parameters:
                - name: itemId
                  value: '{{item.Id}}'
            withParam: '{{tasks.Create-Json-List.outputs.parameters.jsonList}}'
            depends: Create-Json-List
          - name: Check-Output
            template: checkOutput
            arguments:
              parameters:
                - name: someInput
                  value: '{{tasks.Retry-And-Loop.outputs.parameters.outputParam}}'
            depends: Retry-And-Loop
    - name: checkOutput
      inputs:
        parameters:
          - name: someInput
      outputs: {}
      metadata: {}
      script:
        name: ''
        image: 'alpine:3.7'
        command:
          - /bin/sh
        resources: {}
        source: |
          echo "The Output is: {{inputs.parameters.someInput}}"
    - name: retryAndLoop
      inputs:
        parameters:
          - name: itemId
      outputs:
        parameters:
          - name: outputParam
            valueFrom:
              path: /outputParam.txt
      metadata: {}
      script:
        name: ''
        image: 'alpine:3.7'
        command:
          - /bin/sh
        resources: {}
        source: |
          echo ItemId: {{inputs.parameters.itemId}}
          if [[ {{ retries }} == 0 ]]; then
            exit 1 # Exit as failed on zeroth retry
          fi
          echo "Successful" > /outputParam.txt
          exit 0 # Else exit successfully
      retryStrategy:
        limit: '2'
    - name: createJsonList
      inputs: {}
      outputs:
        parameters:
          - name: jsonList
            valueFrom:
              path: /jsonList.json
      metadata: {}
      script:
        name: ''
        image: 'alpine:3.7'
        command:
          - /bin/sh
        resources: {}
        source: |
          echo [{\"Id\": \"1\"}, {\"Id\": \"2\"}] > /jsonList.json
  entrypoint: hr-retry-replication
  arguments: {}
  ttlStrategy:
    secondsAfterCompletion: 600
  activeDeadlineSeconds: 300
  podSpecPatch: |
    terminationGracePeriodSeconds: 3
status:
  phase: Succeeded
  startedAt: '2022-01-06T00:09:56Z'
  finishedAt: '2022-01-06T00:10:17Z'
  progress: 6/6
  nodes:
    hr-retry-replication:
      id: hr-retry-replication
      name: hr-retry-replication
      displayName: hr-retry-replication
      type: DAG
      templateName: hr-retry-replication
      templateScope: local/hr-retry-replication
      phase: Succeeded
      startedAt: '2022-01-06T00:09:56Z'
      finishedAt: '2022-01-06T00:10:17Z'
      progress: 6/6
      resourcesDuration:
        cpu: 18
        memory: 8
      children:
        - hr-retry-replication-2229970335
      outboundNodes:
        - hr-retry-replication-2709022465
    hr-retry-replication-1172938528:
      id: hr-retry-replication-1172938528
      name: 'hr-retry-replication.Retry-And-Loop(0:Id:1)(0)'
      displayName: 'Retry-And-Loop(0:Id:1)(0)'
      type: Pod
      templateName: retryAndLoop
      templateScope: local/hr-retry-replication
      phase: Failed
      boundaryID: hr-retry-replication
      message: Error (exit code 1)
      startedAt: '2022-01-06T00:10:00Z'
      finishedAt: '2022-01-06T00:10:05Z'
      progress: 1/1
      resourcesDuration:
        cpu: 2
        memory: 1
      inputs:
        parameters:
          - name: itemId
            value: '1'
      outputs:
        parameters:
          - name: outputParam
            valueFrom:
              path: /outputParam.txt
        artifacts:
          - name: main-logs
            s3:
              key: hr-retry-replication/hr-retry-replication-1172938528/main.log
        exitCode: '1'
      hostNodeName: k3d-k3s-default-server-0
    hr-retry-replication-1480423937:
      id: hr-retry-replication-1480423937
      name: 'hr-retry-replication.Retry-And-Loop(1:Id:2)(1)'
      displayName: 'Retry-And-Loop(1:Id:2)(1)'
      type: Pod
      templateName: retryAndLoop
      templateScope: local/hr-retry-replication
      phase: Succeeded
      boundaryID: hr-retry-replication
      startedAt: '2022-01-06T00:10:05Z'
      finishedAt: '2022-01-06T00:10:12Z'
      progress: 1/1
      resourcesDuration:
        cpu: 4
        memory: 2
      inputs:
        parameters:
          - name: itemId
            value: '2'
      outputs:
        parameters:
          - name: outputParam
            value: Successful
            valueFrom:
              path: /outputParam.txt
        artifacts:
          - name: main-logs
            s3:
              key: hr-retry-replication/hr-retry-replication-1480423937/main.log
        exitCode: '0'
      children:
        - hr-retry-replication-2709022465
      hostNodeName: k3d-k3s-default-server-0
    hr-retry-replication-1488413861:
      id: hr-retry-replication-1488413861
      name: 'hr-retry-replication.Retry-And-Loop(1:Id:2)'
      displayName: 'Retry-And-Loop(1:Id:2)'
      type: Retry
      templateName: retryAndLoop
      templateScope: local/hr-retry-replication
      phase: Succeeded
      boundaryID: hr-retry-replication
      startedAt: '2022-01-06T00:10:00Z'
      finishedAt: '2022-01-06T00:10:12Z'
      progress: 3/3
      resourcesDuration:
        cpu: 9
        memory: 4
      inputs:
        parameters:
          - name: itemId
            value: '2'
      outputs:
        parameters:
          - name: outputParam
            value: Successful
            valueFrom:
              path: /outputParam.txt
        artifacts:
          - name: main-logs
            s3:
              key: hr-retry-replication/hr-retry-replication-1480423937/main.log
        exitCode: '0'
      children:
        - hr-retry-replication-3158332932
        - hr-retry-replication-1480423937
    hr-retry-replication-2229970335:
      id: hr-retry-replication-2229970335
      name: hr-retry-replication.Create-Json-List
      displayName: Create-Json-List
      type: Pod
      templateName: createJsonList
      templateScope: local/hr-retry-replication
      phase: Succeeded
      boundaryID: hr-retry-replication
      startedAt: '2022-01-06T00:09:56Z'
      finishedAt: '2022-01-06T00:10:00Z'
      progress: 1/1
      resourcesDuration:
        cpu: 3
        memory: 1
      outputs:
        parameters:
          - name: jsonList
            value: '[{"Id": "1"}, {"Id": "2"}]'
            valueFrom:
              path: /jsonList.json
        artifacts:
          - name: main-logs
            s3:
              key: hr-retry-replication/hr-retry-replication-2229970335/main.log
        exitCode: '0'
      children:
        - hr-retry-replication-3704116740
      hostNodeName: k3d-k3s-default-server-0
    hr-retry-replication-2709022465:
      id: hr-retry-replication-2709022465
      name: hr-retry-replication.Check-Output
      displayName: Check-Output
      type: Pod
      templateName: checkOutput
      templateScope: local/hr-retry-replication
      phase: Succeeded
      boundaryID: hr-retry-replication
      startedAt: '2022-01-06T00:10:12Z'
      finishedAt: '2022-01-06T00:10:17Z'
      progress: 1/1
      resourcesDuration:
        cpu: 3
        memory: 1
      inputs:
        parameters:
          - name: someInput
            value: '["Successful","Successful"]'
      outputs:
        artifacts:
          - name: main-logs
            s3:
              key: hr-retry-replication/hr-retry-replication-2709022465/main.log
        exitCode: '0'
      hostNodeName: k3d-k3s-default-server-0
    hr-retry-replication-302978553:
      id: hr-retry-replication-302978553
      name: 'hr-retry-replication.Retry-And-Loop(0:Id:1)'
      displayName: 'Retry-And-Loop(0:Id:1)'
      type: Retry
      templateName: retryAndLoop
      templateScope: local/hr-retry-replication
      phase: Succeeded
      boundaryID: hr-retry-replication
      startedAt: '2022-01-06T00:10:00Z'
      finishedAt: '2022-01-06T00:10:12Z'
      progress: 3/3
      resourcesDuration:
        cpu: 9
        memory: 4
      inputs:
        parameters:
          - name: itemId
            value: '1'
      outputs:
        parameters:
          - name: outputParam
            value: Successful
            valueFrom:
              path: /outputParam.txt
        artifacts:
          - name: main-logs
            s3:
              key: hr-retry-replication/hr-retry-replication-4058438733/main.log
        exitCode: '0'
      children:
        - hr-retry-replication-1172938528
        - hr-retry-replication-4058438733
    hr-retry-replication-3158332932:
      id: hr-retry-replication-3158332932
      name: 'hr-retry-replication.Retry-And-Loop(1:Id:2)(0)'
      displayName: 'Retry-And-Loop(1:Id:2)(0)'
      type: Pod
      templateName: retryAndLoop
      templateScope: local/hr-retry-replication
      phase: Failed
      boundaryID: hr-retry-replication
      message: Error (exit code 1)
      startedAt: '2022-01-06T00:10:00Z'
      finishedAt: '2022-01-06T00:10:05Z'
      progress: 1/1
      resourcesDuration:
        cpu: 2
        memory: 1
      inputs:
        parameters:
          - name: itemId
            value: '2'
      outputs:
        parameters:
          - name: outputParam
            valueFrom:
              path: /outputParam.txt
        artifacts:
          - name: main-logs
            s3:
              key: hr-retry-replication/hr-retry-replication-3158332932/main.log
        exitCode: '1'
      hostNodeName: k3d-k3s-default-server-0
    hr-retry-replication-3704116740:
      id: hr-retry-replication-3704116740
      name: hr-retry-replication.Retry-And-Loop
      displayName: Retry-And-Loop
      type: TaskGroup
      templateName: retryAndLoop
      templateScope: local/hr-retry-replication
      phase: Succeeded
      boundaryID: hr-retry-replication
      startedAt: '2022-01-06T00:10:00Z'
      finishedAt: '2022-01-06T00:10:12Z'
      progress: 5/5
      resourcesDuration:
        cpu: 15
        memory: 7
      children:
        - hr-retry-replication-302978553
        - hr-retry-replication-1488413861
    hr-retry-replication-4058438733:
      id: hr-retry-replication-4058438733
      name: 'hr-retry-replication.Retry-And-Loop(0:Id:1)(1)'
      displayName: 'Retry-And-Loop(0:Id:1)(1)'
      type: Pod
      templateName: retryAndLoop
      templateScope: local/hr-retry-replication
      phase: Succeeded
      boundaryID: hr-retry-replication
      startedAt: '2022-01-06T00:10:05Z'
      finishedAt: '2022-01-06T00:10:12Z'
      progress: 1/1
      resourcesDuration:
        cpu: 4
        memory: 2
      inputs:
        parameters:
          - name: itemId
            value: '1'
      outputs:
        parameters:
          - name: outputParam
            value: Successful
            valueFrom:
              path: /outputParam.txt
        artifacts:
          - name: main-logs
            s3:
              key: hr-retry-replication/hr-retry-replication-4058438733/main.log
        exitCode: '0'
      children:
        - hr-retry-replication-2709022465
      hostNodeName: k3d-k3s-default-server-0`

func TestRetryLoopWithOutputParam(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(RetryLoopWithOutputParam)
	cancel, controller := newController(wf)
	defer cancel()

	ctx := context.Background()
	woc := newWorkflowOperationCtx(wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

var workflowShuttingDownWithNodesInPendingAfterReconsiliation = `apiVersion: argoproj.io/v1alpha1
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
      phase: Failed
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

func TestFailSuspendedAndPendingNodesAfterDeadlineOrShutdown(t *testing.T) {
	cancel, controller := newController()
	defer cancel()

	t.Run("Shutdown", func(t *testing.T) {
		workflow := wfv1.MustUnmarshalWorkflow(workflowShuttingDownWithNodesInPendingAfterReconsiliation)
		woc := newWorkflowOperationCtx(workflow, controller)

		woc.execWf.Spec.Shutdown = "Terminate"
		woc.execWf.Spec.ActiveDeadlineSeconds = nil

		step1NodeName := "container-set-termination-demopw5vv-893664226"
		step2NodeName := "container-set-termination-demopw5vv-876886607"

		// update step1 type to NodeTypePod
		//              phase to NodeRunning
		if entry, ok := woc.wf.Status.Nodes[step1NodeName]; ok {
			entry.Type = wfv1.NodeTypePod
			entry.Phase = wfv1.NodeRunning
			woc.wf.Status.Nodes[step1NodeName] = entry
		}

		// update step2 type to NodeTypeSuspend
		//              phase to NodeRunning
		if entry, ok := woc.wf.Status.Nodes[step2NodeName]; ok {
			entry.Type = wfv1.NodeTypeSuspend
			entry.Phase = wfv1.NodeRunning
			woc.wf.Status.Nodes[step2NodeName] = entry
		}

		assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes[step1NodeName].Phase)
		assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes[step2NodeName].Phase)

		woc.failSuspendedAndPendingNodesAfterDeadlineOrShutdown()

		assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes[step1NodeName].Phase)
		assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Nodes[step2NodeName].Phase)
	})

	t.Run("Deadline", func(t *testing.T) {
		workflow := wfv1.MustUnmarshalWorkflow(workflowShuttingDownWithNodesInPendingAfterReconsiliation)
		woc := newWorkflowOperationCtx(workflow, controller)

		woc.execWf.Spec.Shutdown = ""

		deadlineInSeconds := int64(10)
		woc.wf.Status.StartedAt = metav1.NewTime(time.Now().AddDate(0, 0, -1)) // yesterday
		woc.execWf.Spec.ActiveDeadlineSeconds = &deadlineInSeconds
		woc.workflowDeadline = woc.getWorkflowDeadline()

		step1NodeName := "container-set-termination-demopw5vv-893664226"
		step2NodeName := "container-set-termination-demopw5vv-876886607"

		// update step1 phase to NodeRunning
		if entry, ok := woc.wf.Status.Nodes[step1NodeName]; ok {
			entry.Phase = wfv1.NodeRunning
			woc.wf.Status.Nodes[step1NodeName] = entry
		}

		// update step2 phase to NodePending
		if entry, ok := woc.wf.Status.Nodes[step2NodeName]; ok {
			entry.Phase = wfv1.NodePending
			woc.wf.Status.Nodes[step2NodeName] = entry
		}

		assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes[step1NodeName].Phase)
		assert.Equal(t, wfv1.NodePending, woc.wf.Status.Nodes[step2NodeName].Phase)

		woc.failSuspendedAndPendingNodesAfterDeadlineOrShutdown()

		assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes[step1NodeName].Phase)
		assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Nodes[step2NodeName].Phase)
	})
}
