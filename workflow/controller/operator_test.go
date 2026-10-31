package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	apiv1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	batchfake "k8s.io/client-go/kubernetes/typed/batch/v1/fake"
	corefake "k8s.io/client-go/kubernetes/typed/core/v1/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/yaml"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	intstrutil "github.com/argoproj/argo-workflows/v4/util/intstr"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/strftime"
	"github.com/argoproj/argo-workflows/v4/util/variables"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/cache"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/dag"
	hydratorfake "github.com/argoproj/argo-workflows/v4/workflow/hydrator/fake"
	"github.com/argoproj/argo-workflows/v4/workflow/sync"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

// TestOperateWorkflowPanicRecover ensures we can recover from unexpected panics
func TestOperateWorkflowPanicRecover(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fail()
		}
	}()
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	// intentionally set clientset to nil to induce panic
	controller.kubeclientset = nil
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	ctx := logging.TestContext(t.Context())
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("").Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
}

func Test_wfOperationCtx_reapplyUpdate(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
			Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{Name: "my-foo"}}},
		}
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, wf)
		defer cancel()
		controller.hydrator = hydratorfake.Always
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		// fake the behaviour woc.operate()
		require.NoError(t, controller.hydrator.Hydrate(ctx, wf))
		nodes := wfv1.Nodes{"foo": wfv1.NodeStatus{Name: "my-foo", Phase: wfv1.NodeSucceeded}}

		// now force a re-apply update
		updatedWf, err := woc.reapplyUpdate(ctx, controller.wfclientset.ArgoprojV1alpha1().Workflows(""), nodes)
		require.NoError(t, err)
		require.NotNil(t, updatedWf)
		assert.True(t, woc.controller.hydrator.IsHydrated(updatedWf))
		require.Contains(t, updatedWf.Status.Nodes, "foo")
		assert.Equal(t, "my-foo", updatedWf.Status.Nodes["foo"].Name)
		assert.Equal(t, wfv1.NodeSucceeded, updatedWf.Status.Nodes["foo"].Phase, "phase is merged")
	})
	t.Run("ErrUpdatingCompletedWorkflow", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
			Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowError},
		}
		currWf := wf.DeepCopy()
		currWf.Status.Phase = wfv1.WorkflowSucceeded
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, currWf)
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		_, err := woc.reapplyUpdate(ctx, controller.wfclientset.ArgoprojV1alpha1().Workflows(""), wfv1.Nodes{})
		require.EqualError(t, err, "must never update completed workflows")
	})
	t.Run("ErrUpdatingCompletedNode", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wf"},
			Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"my-node": wfv1.NodeStatus{Phase: wfv1.NodeError}}},
		}
		currWf := wf.DeepCopy()
		currWf.Status.Nodes = wfv1.Nodes{"my-node": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded}}
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, currWf)
		defer cancel()
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		_, err := woc.reapplyUpdate(ctx, controller.wfclientset.ArgoprojV1alpha1().Workflows(""), wf.Status.Nodes)
		require.EqualError(t, err, "must never update completed node my-node")
	})
	// https://github.com/argoproj/argo-workflows/issues/16774
	// If the workflow was deleted and recreated under the same name while the controller was
	// mid-operation on it, the UID-guarded Update fails on the UID precondition, which IsConflict
	// treats as an ordinary conflict. reapplyUpdate must detect the UID mismatch on the object it
	// Gets by name and refuse to merge-patch the old instance's status delta onto the new object.
	t.Run("ErrWorkflowRecreatedWithDifferentUID", func(t *testing.T) {
		wf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "old-uid"},
			Status:     wfv1.WorkflowStatus{Nodes: wfv1.Nodes{"foo": wfv1.NodeStatus{Name: "my-foo"}}},
		}
		// currWf simulates the recreated workflow: same name, different UID, no status.
		currWf := &wfv1.Workflow{
			ObjectMeta: metav1.ObjectMeta{Name: "my-wf", UID: "new-uid"},
		}
		ctx := logging.TestContext(t.Context())
		cancel, controller := newController(ctx, currWf)
		defer cancel()
		controller.hydrator = hydratorfake.Always
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		require.NoError(t, controller.hydrator.Hydrate(ctx, wf))
		nodes := wfv1.Nodes{"foo": wfv1.NodeStatus{Name: "my-foo", Phase: wfv1.NodeSucceeded}}

		_, err := woc.reapplyUpdate(ctx, controller.wfclientset.ArgoprojV1alpha1().Workflows(""), nodes)
		require.EqualError(t, err, "workflow has been recreated, not updating(old-uid -> new-uid)")
	})
}

func TestResourcesDuration(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/resources-duration.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.NotEmpty(t, woc.wf.Status.ResourcesDuration, "workflow duration not empty")
	assert.False(t, woc.wf.Status.Nodes.Any(func(node wfv1.NodeStatus) bool {
		return node.ResourcesDuration.IsZero()
	}), "zero node durations empty")
}

func TestGlobalParamDuration(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/global-param-duration.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, "0.000000", woc.globalParams()[varkeys.WorkflowDuration.Template()])

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Greater(t, woc.globalParams()[varkeys.WorkflowDuration.Template()], "0.000000")
}

func TestEstimatedDuration(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/estimated-duration-workflow.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wfv1.MustUnmarshalWorkflow("@testdata/operator/estimated-duration-baseline-workflow.yaml"), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.EstimatedDuration(1), woc.wf.Status.EstimatedDuration)
	assert.Equal(t, wfv1.EstimatedDuration(1), woc.wf.Status.Nodes[woc.wf.Name].EstimatedDuration)
	assert.Equal(t, wfv1.EstimatedDuration(1), woc.wf.Status.Nodes.FindByDisplayName("pod").EstimatedDuration)
}

func TestDefaultProgress(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/default-progress.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Equal(t, "false", woc.wf.Labels[common.LabelKeyCompleted])

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Progress("1/1"), woc.wf.Status.Progress)
	assert.Equal(t, wfv1.Progress("1/1"), woc.wf.Status.Nodes[woc.wf.Name].Progress)
	assert.Equal(t, wfv1.Progress("1/1"), woc.wf.Status.Nodes.FindByDisplayName("pod").Progress)
	assert.Equal(t, "true", woc.wf.Labels[common.LabelKeyCompleted])
}

func TestLoggedProgress(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/logged-progress.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Progress("0/100"), woc.wf.Status.Progress)
	assert.Equal(t, wfv1.Progress("0/100"), woc.wf.Status.Nodes[woc.wf.Name].Progress)
	pod := woc.wf.Status.Nodes.FindByDisplayName("pod")
	assert.Equal(t, wfv1.Progress("0/100"), pod.Progress)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Progress("100/100"), woc.wf.Status.Progress)
	assert.Equal(t, wfv1.Progress("100/100"), woc.wf.Status.Nodes[woc.wf.Name].Progress)
	pod = woc.wf.Status.Nodes.FindByDisplayName("pod")
	assert.Equal(t, wfv1.Progress("100/100"), pod.Progress)
}

func TestGlobalParams(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	require.Contains(t, woc.globalParams(), "workflow.creationTimestamp")
	assert.NotContains(t, woc.globalParams()["workflow.creationTimestamp"], "UTC")
	for char := range strftime.FormatChars {
		assert.Contains(t, woc.globalParams(), fmt.Sprintf("%s.%s", "workflow.creationTimestamp", string(char)))
	}
	assert.Contains(t, woc.globalParams(), "workflow.creationTimestamp.s")
	assert.Contains(t, woc.globalParams(), "workflow.creationTimestamp.RFC3339")

	assert.Contains(t, woc.globalParams(), "workflow.duration")
	assert.Contains(t, woc.globalParams(), "workflow.name")
	assert.Contains(t, woc.globalParams(), "workflow.namespace")
	assert.Contains(t, woc.globalParams(), "workflow.mainEntrypoint")
	assert.Contains(t, woc.globalParams(), "workflow.parameters")
	assert.Contains(t, woc.globalParams(), "workflow.annotations")
	assert.Contains(t, woc.globalParams(), "workflow.labels")
	assert.Contains(t, woc.globalParams(), "workflow.serviceAccountName")
	assert.Contains(t, woc.globalParams(), "workflow.uid")

	// Ensure that the phase label is included after the first operation
	woc.operate(ctx)
	assert.Contains(t, woc.globalParams(), "workflow.labels.workflows.argoproj.io/phase")
}

// TestSidecarWithVolume verifies ia sidecar can have a volumeMount reference to both existing or volumeClaimTemplate volumes
func TestSidecarWithVolume(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/sidecar-with-vol.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.NotEmpty(t, pods.Items, "pod was not created successfully")
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
      image: python:alpine3.23
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
		expectedVolumesRemaining: 1,
	}, {
		name:                     "failed/OnWorkflowSuccess",
		strategy:                 wfv1.VolumeClaimGCOnSuccess,
		phase:                    wfv1.NodeFailed,
		expectedVolumesRemaining: 1,
	}, {
		name:                     "succeeded/OnWorkflowSuccess",
		strategy:                 wfv1.VolumeClaimGCOnSuccess,
		phase:                    wfv1.NodeSucceeded,
		expectedVolumesRemaining: 1,
	}, {
		name:                     "succeeded/OnWorkflowCompletion",
		strategy:                 wfv1.VolumeClaimGCOnCompletion,
		phase:                    wfv1.NodeSucceeded,
		expectedVolumesRemaining: 1,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(makeVolumeGcStrategyTemplate(tt.strategy, tt.phase))
			cancel, controller := newController(logging.TestContext(t.Context()), wf)
			defer cancel()

			ctx := logging.TestContext(t.Context())
			wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)
			wf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Len(t, wf.Status.PersistentVolumeClaims, tt.expectedVolumesRemaining)
		})
	}
}

// TestProcessNodeRetries tests the processNodeRetries() method.
func TestProcessNodeRetries(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Empty(t, woc.wf.Status.Nodes)

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	ctx, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, wfv1.NodeRunning, node.Phase)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := range 2 {
		childNode := fmt.Sprintf("%s(%d)", nodeName, i)
		woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true}, true)
		woc.addChildNode(ctx, nodeName, childNode)
	}

	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodeRetries() should return false since
	// there should be no retries at this point.
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Mark lastChild as successful.
	woc.markNodePhase(ctx, lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, wfv1.NodeSucceeded, n.Phase)

	// Mark the parent node as running again and the lastChild as failed.
	// These are synthetic test rewinds; production never re-enters Running
	// from Succeeded, so we bypass markNodePhase (which would reject the
	// transition under the node phase SM) and write directly to the status.
	forceNodePhaseForTest(woc, n.Name, wfv1.NodeRunning)
	forceNodePhaseForTest(woc, lastChild.Name, wfv1.NodeFailed)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	_, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Add a hook node that has Succeeded
	childHookedNode := "child-node.hooks.running"
	woc.initializeNode(ctx, childHookedNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeSucceeded, &wfv1.NodeFlag{Hooked: true}, true)
	woc.addChildNode(ctx, nodeName, childHookedNode)

	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Add a third node that has failed.
	childNode := fmt.Sprintf("%s(%d)", nodeName, 3)
	woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed, &wfv1.NodeFlag{Retried: true}, true)
	woc.addChildNode(ctx, nodeName, childNode)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, n.Phase)
}

// TestProcessNodeRetriesClearsDaemonedFlag verifies that when a daemon child node
// fails (is no longer daemoned), the retry parent's Daemoned flag is cleared so
// the retry logic can proceed instead of treating the node as "fulfilled".
func TestProcessNodeRetriesClearsDaemonedFlag(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	// Create a retry parent node that is Running + Daemoned (as if its child was previously daemoned)
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	ctx, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	daemon := true
	node.Daemoned = &daemon
	woc.wf.Status.Nodes[nodeID] = *node

	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")

	// Add a child node that was daemoned but has now failed
	childNode := fmt.Sprintf("%s(%d)", nodeName, 0)
	woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed, &wfv1.NodeFlag{Retried: true}, true)
	woc.addChildNode(ctx, nodeName, childNode)

	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	assert.True(t, n.IsDaemoned(), "retry node should start with Daemoned=true")

	// processNodeRetries should clear the Daemoned flag since the child is no longer daemoned
	n, continueExecution, err := woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.True(t, continueExecution, "should continue execution to create a retry")
	assert.False(t, n.IsDaemoned(), "Daemoned flag should be cleared when child is no longer daemoned")
	assert.Equal(t, wfv1.NodeRunning, n.Phase, "retry node should stay Running to allow retry")
}

// TestProcessNodeRetries tests retrying when RetryOn.Error is enabled
func TestProcessNodeRetriesOnErrors(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Empty(t, woc.wf.Status.Nodes)

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	_, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")
	retries.RetryPolicy = wfv1.RetryPolicyAlways
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, wfv1.NodeRunning, node.Phase)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := range 2 {
		childNode := fmt.Sprintf("%s(%d)", nodeName, i)
		woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true}, true)
		woc.addChildNode(ctx, nodeName, childNode)
	}

	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodeRetries() should return false since
	// there should be no retries at this point.
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Mark lastChild as successful.
	woc.markNodePhase(ctx, lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, wfv1.NodeSucceeded, n.Phase)

	// Mark the parent node as running again and the lastChild as errored.
	// Synthetic test rewind; bypass markNodePhase (forbidden by SM).
	forceNodePhaseForTest(woc, n.Name, wfv1.NodeRunning)
	forceNodePhaseForTest(woc, lastChild.Name, wfv1.NodeError)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	_, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Add a third node that has errored.
	childNode := fmt.Sprintf("%s(%d)", nodeName, 3)
	woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeError, &wfv1.NodeFlag{Retried: true}, true)
	woc.addChildNode(ctx, nodeName, childNode)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, n.Phase)
}

// TestProcessNodeRetries tests retrying when RetryOnTransientError is enabled
func TestProcessNodeRetriesOnTransientErrors(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Empty(t, woc.wf.Status.Nodes)

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	_, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")
	retries.RetryPolicy = wfv1.RetryPolicyOnTransientError
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, wfv1.NodeRunning, node.Phase)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := range 2 {
		childNode := fmt.Sprintf("%s(%d)", nodeName, i)
		woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true}, true)
		woc.addChildNode(ctx, nodeName, childNode)
	}

	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodeRetries() should return false since
	// there should be no retries at this point.
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Mark lastChild as successful.
	woc.markNodePhase(ctx, lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, wfv1.NodeSucceeded, n.Phase)

	// Mark the parent node as running again and the lastChild as errored with a message that indicates the error
	// is transient. Synthetic test rewind; bypass markNodePhase (forbidden by SM).
	forceNodePhaseForTest(woc, n.Name, wfv1.NodeRunning)
	transientEnvVarKey := "TRANSIENT_ERROR_PATTERN"
	transientErrMsg := "This error is transient"
	forceNodePhaseForTest(woc, lastChild.Name, wfv1.NodeError)
	lcID := woc.wf.NodeID(lastChild.Name)
	lcNode := woc.wf.Status.Nodes[lcID]
	lcNode.Message = transientErrMsg
	woc.wf.Status.Nodes[lcID] = lcNode
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	t.Setenv(transientEnvVarKey, transientErrMsg)
	_, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Add a third node that has errored.
	childNode := fmt.Sprintf("%s(%d)", nodeName, 3)
	woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeError, &wfv1.NodeFlag{Retried: true}, true)
	woc.addChildNode(ctx, nodeName, childNode)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, n.Phase)
}

func TestProcessNodeRetriesWithBackoff(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Empty(t, woc.wf.Status.Nodes)

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	_, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")
	retries.Backoff = &wfv1.Backoff{
		Duration:    "10s",
		Factor:      intstrutil.ParsePtr("2"),
		MaxDuration: "10m",
	}
	retries.RetryPolicy = wfv1.RetryPolicyAlways
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, wfv1.NodeRunning, node.Phase)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	woc.initializeNode(ctx, nodeName+"(0)", wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true}, true)
	woc.addChildNode(ctx, nodeName, nodeName+"(0)")

	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodeRetries() should return false since
	// there should be no retries at this point.
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Mark lastChild as successful.
	woc.markNodePhase(ctx, lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, wfv1.NodeSucceeded, n.Phase)
}

func TestProcessNodeRetriesWithExponentialBackoff(t *testing.T) {
	require := require.New(t)

	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	require.NotNil(controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	require.NotNil(wf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	require.NotNil(woc)

	// Verify that there are no nodes in the wf status.
	require.Empty(woc.wf.Status.Nodes)

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	_, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("3")
	retries.RetryPolicy = wfv1.RetryPolicyAlways
	retries.Backoff = &wfv1.Backoff{
		Duration: "5m",
		Factor:   intstrutil.ParsePtr("2"),
		Cap:      "11m",
	}
	woc.wf.Status.Nodes[nodeID] = *node

	require.Equal(wfv1.NodeRunning, node.Phase)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	require.Nil(lastChild)

	woc.initializeNode(ctx, nodeName+"(0)", wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed, &wfv1.NodeFlag{Retried: true}, true)
	woc.addChildNode(ctx, nodeName, nodeName+"(0)")

	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(err)

	// Last child has failed. processNodesWithRetries() should return false due to the default backoff.
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(err)
	require.Equal(wfv1.NodeRunning, n.Phase)

	// First backoff should be between 295 and 300 seconds.
	backoff, err := parseRetryMessage(n.Message)
	require.NoError(err)
	require.LessOrEqual(backoff, 300)
	require.Less(295, backoff)

	woc.initializeNode(ctx, nodeName+"(1)", wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeError, &wfv1.NodeFlag{Retried: true}, true)
	woc.addChildNode(ctx, nodeName, nodeName+"(1)")
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(err)

	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(err)
	require.Equal(wfv1.NodeRunning, n.Phase)

	// Second backoff should be between 595 and 600 seconds.
	backoff, err = parseRetryMessage(n.Message)
	require.NoError(err)
	require.LessOrEqual(backoff, 600)
	require.Less(595, backoff)

	woc.initializeNode(ctx, nodeName+"(2)", wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeError, &wfv1.NodeFlag{Retried: true}, true)
	woc.addChildNode(ctx, nodeName, nodeName+"(2)")
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(err)

	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(err)
	require.Equal(wfv1.NodeRunning, n.Phase)

	// Third backoff should be limited to 660 seconds by the Cap.
	backoff, err = parseRetryMessage(n.Message)
	require.NoError(err)
	require.LessOrEqual(backoff, 660)
	require.Less(655, backoff)

	// Mark lastChild as successful. Synthetic test rewind from Error to
	// Succeeded; bypass markNodePhase (forbidden by SM).
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	require.NotNil(lastChild)
	forceNodePhaseForTest(woc, lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(err)
	// The parent node also gets marked as Succeeded.
	require.Equal(wfv1.NodeSucceeded, n.Phase)
}

func TestProcessNodeRetriesBackoffOverflow(t *testing.T) {
	require := require.New(t)

	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	_, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("100")
	retries.RetryPolicy = wfv1.RetryPolicyAlways
	retries.Backoff = &wfv1.Backoff{
		Duration: "5s",
		Factor:   intstrutil.ParsePtr("2"),
	}
	woc.wf.Status.Nodes[nodeID] = *node

	// Add 32 child nodes to trigger overflow (2^31 * 5s would overflow int64)
	for i := range 32 {
		childName := fmt.Sprintf("%s(%d)", nodeName, i)
		woc.initializeNode(ctx, childName, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed, &wfv1.NodeFlag{Retried: true}, true)
		woc.addChildNode(ctx, nodeName, childName)
	}

	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(err)

	// Should not panic or overflow - should cap at max duration
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(err)
	require.Equal(wfv1.NodeRunning, n.Phase)
	require.Contains(n.Message, "Backoff for")
}

// TestProcessNodeRetries tests retrying with Expression
func TestProcessNodeRetriesWithExpression(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Empty(t, woc.wf.Status.Nodes)

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	_, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	retries := wfv1.RetryStrategy{}
	retries.Expression = "false"
	retries.Limit = intstrutil.ParsePtr("2")
	retries.RetryPolicy = wfv1.RetryPolicyAlways
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, wfv1.NodeRunning, node.Phase)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := range 2 {
		childNode := fmt.Sprintf("%s(%d)", nodeName, i)
		woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true}, true)
		woc.addChildNode(ctx, nodeName, childNode)
	}

	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodeRetries() should return false since
	// there should be no retries at this point.
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Mark lastChild Pending.
	woc.markNodePhase(ctx, lastChild.Name, wfv1.NodePending)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Mark lastChild as successful.
	woc.markNodePhase(ctx, lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, wfv1.NodeSucceeded, n.Phase)
	assert.Empty(t, n.Message)

	// Mark the parent node as running again and the lastChild as errored.
	// Synthetic test rewind; bypass markNodePhase (forbidden by SM).
	forceNodePhaseForTest(woc, n.Name, wfv1.NodeRunning)
	forceNodePhaseForTest(woc, lastChild.Name, wfv1.NodeError)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	_, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, n.Phase)
	assert.Equal(t, "retryStrategy.expression evaluated to false", n.Message)

	// Add a third node that has failed. Synthetic test rewind of the
	// retry parent (Error -> Running) bypasses markNodePhase.
	forceNodePhaseForTest(woc, n.Name, wfv1.NodeRunning)
	childNode := fmt.Sprintf("%s(%d)", nodeName, 3)
	woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed, &wfv1.NodeFlag{Retried: true}, true)
	woc.addChildNode(ctx, nodeName, childNode)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, n.Phase)
	assert.Equal(t, "No more retries left", n.Message)
}

func TestProcessNodeRetriesMessageOrder(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Empty(t, woc.wf.Status.Nodes)

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	_, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	retries := wfv1.RetryStrategy{}
	retries.Expression = "false"
	retries.Limit = intstrutil.ParsePtr("1")
	retries.RetryPolicy = wfv1.RetryPolicyAlways
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, wfv1.NodeRunning, node.Phase)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := range 1 {
		childNode := fmt.Sprintf("%s(%d)", nodeName, i)
		woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true}, true)
		woc.addChildNode(ctx, nodeName, childNode)
	}

	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// No retry related message for running node
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// No retry related message for pending node
	woc.markNodePhase(ctx, lastChild.Name, wfv1.NodePending)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)
	assert.Empty(t, n.Message)

	// No retry related message for succeeded node
	woc.markNodePhase(ctx, lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, n.Phase)
	assert.Empty(t, n.Message)

	// workflow mark shutdown, no retry is evaluated. Synthetic test rewinds:
	// the parent must go Succeeded → Running and the child Succeeded →
	// Error / Failed for these scenarios. markNodePhase rejects those
	// transitions under the node phase SM, so use the direct setter.
	woc.wf.Spec.Shutdown = wfv1.ShutdownStrategyStop
	forceNodePhaseForTest(woc, n.Name, wfv1.NodeRunning)
	forceNodePhaseForTest(woc, lastChild.Name, wfv1.NodeError)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	_, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, n.Phase)
	assert.Equal(t, "Stopped with strategy 'Stop'", n.Message)
	woc.wf.Spec.Shutdown = ""

	// Invalid retry policy, shouldn't evaluate expression
	retries.RetryPolicy = "noExist"
	forceNodePhaseForTest(woc, n.Name, wfv1.NodeRunning)
	forceNodePhaseForTest(woc, lastChild.Name, wfv1.NodeError)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	_, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	assert.Equal(t, "noExist is not a valid RetryPolicy", err.Error())

	// Node status doesn't with retrypolicy, shouldn't evaluate expression
	retries.RetryPolicy = wfv1.RetryPolicyOnFailure
	forceNodePhaseForTest(woc, n.Name, wfv1.NodeRunning)
	forceNodePhaseForTest(woc, lastChild.Name, wfv1.NodeError)
	// Clear stale message left over from the prior Shutdown branch so the
	// retry-policy-mismatch path's "no message" expectation holds.
	lcID := woc.wf.NodeID(lastChild.Name)
	lcNode := woc.wf.Status.Nodes[lcID]
	lcNode.Message = ""
	woc.wf.Status.Nodes[lcID] = lcNode
	nID := woc.wf.NodeID(n.Name)
	nNode := woc.wf.Status.Nodes[nID]
	nNode.Message = ""
	woc.wf.Status.Nodes[nID] = nNode
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	_, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, n.Phase)
	assert.Empty(t, n.Message)

	// Node status aligns with retrypolicy, should evaluate expression
	retries.RetryPolicy = wfv1.RetryPolicyOnFailure
	forceNodePhaseForTest(woc, n.Name, wfv1.NodeRunning)
	forceNodePhaseForTest(woc, lastChild.Name, wfv1.NodeFailed)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	_, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, n.Phase)
	assert.Equal(t, "retryStrategy.expression evaluated to false", n.Message)

	// Node status aligns with retrypolicy but reach max retry limit, shouldn't evaluate expression
	forceNodePhaseForTest(woc, n.Name, wfv1.NodeRunning)
	childNode := fmt.Sprintf("%s(%d)", nodeName, 1)
	woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed, &wfv1.NodeFlag{Retried: true}, true)
	woc.addChildNode(ctx, nodeName, childNode)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, n.Phase)
	assert.Equal(t, "No more retries left", n.Message)
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

// TestProcessNodesNoRetryWithError tests retrying when RetryOn.Error is disabled
func TestProcessNodesNoRetryWithError(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	assert.NotNil(t, wf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotNil(t, woc)
	// Verify that there are no nodes in the wf status.
	assert.Empty(t, woc.wf.Status.Nodes)

	// Add the parent node for retries.
	nodeName := "test-node"
	nodeID := woc.wf.NodeID(nodeName)
	_, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true}, true)
	retries := wfv1.RetryStrategy{}
	retries.Limit = intstrutil.ParsePtr("2")
	retries.RetryPolicy = wfv1.RetryPolicyOnFailure
	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, wfv1.NodeRunning, node.Phase)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child nodes.
	for i := range 2 {
		childNode := fmt.Sprintf("%s(%d)", nodeName, i)
		woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true}, true)
		woc.addChildNode(ctx, nodeName, childNode)
	}

	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	// Last child is still running. processNodeRetries() should return false since
	// there should be no retries at this point.
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Mark lastChild as successful.
	woc.markNodePhase(ctx, lastChild.Name, wfv1.NodeSucceeded)
	n, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	// The parent node also gets marked as Succeeded.
	assert.Equal(t, wfv1.NodeSucceeded, n.Phase)

	// Mark the parent node as running again and the lastChild as errored.
	// Parent node should also be errored because retry on error is disabled.
	// Synthetic test rewind; bypass markNodePhase (forbidden by SM).
	forceNodePhaseForTest(woc, n.Name, wfv1.NodeRunning)
	forceNodePhaseForTest(woc, lastChild.Name, wfv1.NodeError)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	_, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, n.Phase)
}

func TestBackoffMessage(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/backoff-message.yaml")
	assert.NotNil(t, wf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotNil(t, woc)
	retryNode, err := woc.wf.GetNodeByName("retry-backoff-s69z6")
	require.NoError(t, err)

	// Simulate backoff of 4 secods
	firstNode := getChildNodeIndex(retryNode, woc.wf.Status.Nodes, 0)
	firstNode.StartedAt = metav1.Time{Time: time.Now().Add(-8 * time.Second)}
	firstNode.FinishedAt = metav1.Time{Time: time.Now().Add(-6 * time.Second)}
	woc.wf.Status.Nodes[firstNode.ID] = *firstNode
	lastNode := getChildNodeIndex(retryNode, woc.wf.Status.Nodes, -1)
	lastNode.StartedAt = metav1.Time{Time: time.Now().Add(-3 * time.Second)}
	lastNode.FinishedAt = metav1.Time{Time: time.Now().Add(-1 * time.Second)}
	woc.wf.Status.Nodes[lastNode.ID] = *lastNode

	newRetryNode, proceed, err := woc.processNodeRetries(ctx, retryNode, *woc.wf.Spec.Templates[0].RetryStrategy, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.False(t, proceed)
	assert.Equal(t, "Backoff for 4 seconds", newRetryNode.Message)

	// Advance time one second
	firstNode.StartedAt = metav1.Time{Time: time.Now().Add(-9 * time.Second)}
	firstNode.FinishedAt = metav1.Time{Time: time.Now().Add(-7 * time.Second)}
	woc.wf.Status.Nodes[firstNode.ID] = *firstNode
	lastNode.StartedAt = metav1.Time{Time: time.Now().Add(-4 * time.Second)}
	lastNode.FinishedAt = metav1.Time{Time: time.Now().Add(-2 * time.Second)}
	woc.wf.Status.Nodes[lastNode.ID] = *lastNode

	newRetryNode, proceed, err = woc.processNodeRetries(ctx, retryNode, *woc.wf.Spec.Templates[0].RetryStrategy, &executeTemplateOpts{})
	require.NoError(t, err)
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

	newRetryNode, proceed, err = woc.processNodeRetries(ctx, retryNode, *woc.wf.Spec.Templates[0].RetryStrategy, &executeTemplateOpts{})
	require.NoError(t, err)
	assert.True(t, proceed)
	// New node is started, message should be clear
	assert.Empty(t, newRetryNode.Message)
}

func TestRetriesVariable(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/retries-variable-template.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	iterations := 5
	var woc *wfOperationCtx
	for i := 1; i <= iterations; i++ {
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		if i != 1 {
			makePodsPhase(ctx, woc, apiv1.PodFailed)
		}
		woc.operate(ctx)
		wf = woc.wf
	}

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, iterations)
	expected := []string{}
	actual := []string{}
	for i := range iterations {
		actual = append(actual, pods.Items[i].Spec.Containers[1].Args[0])
		expected = append(expected, fmt.Sprintf("cowsay %d", i))
	}
	// ordering not preserved
	assert.ElementsMatch(t, expected, actual)
}

// TestRetriesVariableInPodSpecPatch makes sure that {{retries}} variable in pod spec patch is correctly
// updated before each retry
func TestRetriesVariableInPodSpecPatch(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/retries-variable-in-pod-spec-patch-template.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	iterations := 5
	var woc *wfOperationCtx
	for i := 1; i <= iterations; i++ {
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		if i != 1 {
			makePodsPhase(ctx, woc, apiv1.PodFailed)
		}
		woc.operate(ctx)
		wf = woc.wf
	}

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, iterations)
	expected := []string{}
	actual := []string{}
	for i := range iterations {
		actual = append(actual, pods.Items[i].Spec.Containers[1].Resources.Limits.Memory().String())
		expected = append(expected, fmt.Sprintf("%dMi", (i+1)*64))
	}
	// expecting memory limit to increase after each retry: "64Mi", "128Mi", "192Mi", "256Mi", "320Mi"
	// ordering not preserved
	assert.ElementsMatch(t, actual, expected)
}

func TestRetriesVariableWithGlobalVariableInPodSpecPatch(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/retries-variable-with-global-variables-in-pod-spec-patch-template.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	iterations := 5
	var woc *wfOperationCtx
	for i := 1; i <= iterations; i++ {
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		if i != 1 {
			makePodsPhase(ctx, woc, apiv1.PodFailed)
		}
		woc.operate(ctx)
		wf = woc.wf
	}

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, iterations)
	expected := []string{}
	actual := []string{}
	for i := range iterations {
		actual = append(actual, pods.Items[i].Spec.Containers[1].Resources.Limits.Memory().String())
		expected = append(expected, fmt.Sprintf("%dMi", (i+1)*100))
	}
	// expecting memory limit to increase after each retry: "100Mi", "200Mi", "300Mi", "400Mi", "500Mi"
	// ordering not preserved
	assert.ElementsMatch(t, actual, expected)
}

func TestLastRetryVariableInPodSpecPatch(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/lastretry-variable-in-pod-spec-patch-template.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	iterations := 5
	var woc *wfOperationCtx
	for i := 1; i <= iterations; i++ {
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		if i != 1 {
			makePodsPhase(ctx, woc, apiv1.PodFailed, withExitCode(1))
		}
		woc.operate(ctx)
		wf = woc.wf
	}

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, iterations)
	expected := []string{}
	actual := []string{}
	for i := range iterations {
		actual = append(actual, pods.Items[i].Spec.Containers[1].Resources.Limits.Memory().String())
		expected = append(expected, fmt.Sprintf("%dMi", (i+1)*100))
	}
	// expecting memory limit to increase after each retry: "100Mi", "200Mi", "300Mi", "400Mi", "500Mi"
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
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	iterations := 5
	var woc *wfOperationCtx
	for i := 1; i <= iterations; i++ {
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		if i != 1 {
			makePodsPhase(ctx, woc, apiv1.PodFailed)
		}
		woc.operate(ctx)
		wf = woc.wf
	}

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, iterations)

	expected := []string{}
	actual := []string{}
	for i := range iterations {
		actual = append(actual, pods.Items[i].Spec.Containers[1].Args[0])
		expected = append(expected, fmt.Sprintf("cowsay %d", i))
	}
	// ordering not preserved
	assert.ElementsMatch(t, expected, actual)
}

func TestAssessNodeStatus(t *testing.T) {
	const templateName = "whalesay"
	tests := []struct {
		name        string
		pod         *apiv1.Pod
		daemon      bool
		node        *wfv1.NodeStatus
		wantPhase   wfv1.NodePhase
		wantMessage string
		wantNil     bool
	}{{
		name: "pod pending",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodPending,
			},
		},
		node:        &wfv1.NodeStatus{TemplateName: templateName},
		wantPhase:   wfv1.NodePending,
		wantMessage: "",
	}, {
		name: "pod succeeded",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodSucceeded,
			},
		},
		node:        &wfv1.NodeStatus{TemplateName: templateName},
		wantPhase:   wfv1.NodeSucceeded,
		wantMessage: "",
	}, {
		name: "pod failed - daemoned",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodFailed,
			},
		},
		daemon:      true,
		node:        &wfv1.NodeStatus{TemplateName: templateName},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "can't find failed message for pod  namespace ", // daemoned nodes currently don't have a fail message
	}, {
		// node already marked Succeeded by killDaemonedChildren; PodFailed event must be ignored.
		name: "pod failed - daemoned, already marked succeeded by teardown",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodFailed,
			},
		},
		daemon:      true,
		node:        &wfv1.NodeStatus{TemplateName: templateName, Phase: wfv1.NodeSucceeded},
		wantPhase:   wfv1.NodeSucceeded,
		wantMessage: "",
	}, {
		// A live daemon that dies on its own must still be assessed normally: its node is Running
		// and Daemoned, not Succeeded, so the teardown predicate must not swallow the failure --
		// otherwise retryStrategy never sees the attempt fail.
		name: "pod failed - daemoned, died on its own",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Message: "died on its own",
				Phase:   apiv1.PodFailed,
			},
		},
		daemon:      true,
		node:        &wfv1.NodeStatus{TemplateName: templateName, Phase: wfv1.NodeRunning, Daemoned: new(true)},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "died on its own",
	}, {
		// non-daemon node that previously Succeeded must still go through inferFailedReason.
		name: "pod failed - not daemoned, previously succeeded",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Message: "failed for some reason",
				Phase:   apiv1.PodFailed,
			},
		},
		daemon:      false,
		node:        &wfv1.NodeStatus{TemplateName: templateName, Phase: wfv1.NodeSucceeded},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "failed for some reason",
	}, {
		// node already marked Succeeded by killDaemonedChildren; PodSucceeded must not overwrite to NodeFailed.
		name: "pod succeeded - daemoned, already marked succeeded by teardown",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodSucceeded,
			},
		},
		daemon:      true,
		node:        &wfv1.NodeStatus{TemplateName: templateName, Phase: wfv1.NodeSucceeded},
		wantPhase:   wfv1.NodeSucceeded,
		wantMessage: "",
	}, {
		name: "daemon, pod running, node failed",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodRunning,
			},
		},
		daemon:      true,
		node:        &wfv1.NodeStatus{TemplateName: templateName, Phase: wfv1.NodeFailed},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "",
	}, {
		name: "daemon, pod running, node succeeded",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodRunning,
			},
		},
		daemon:      true,
		node:        &wfv1.NodeStatus{TemplateName: templateName, Phase: wfv1.NodeSucceeded},
		wantPhase:   wfv1.NodeSucceeded,
		wantMessage: "",
	}, {
		name: "pod failed - not daemoned",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Message: "failed for some reason",
				Phase:   apiv1.PodFailed,
			},
		},
		node:        &wfv1.NodeStatus{TemplateName: templateName},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "failed for some reason",
	}, {
		name: "pod failed - transition from node pending",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Message: "failed for some reason",
				Phase:   apiv1.PodFailed,
			},
		},
		node:        &wfv1.NodeStatus{TemplateName: templateName, Phase: wfv1.NodePending, Message: "failed for some reason"},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "failed for some reason",
	}, {
		name: "pod failed - init container failed",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				InitContainerStatuses: []apiv1.ContainerStatus{
					{
						Name:  common.InitContainerName,
						State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}},
					},
				},
				ContainerStatuses: []apiv1.ContainerStatus{
					{
						Name:  common.WaitContainerName,
						State: apiv1.ContainerState{Terminated: nil},
					},
					{
						Name:  common.MainContainerName,
						State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 0}},
					},
				},
				Message: "failed since init container failed",
				Phase:   apiv1.PodFailed,
			},
		},
		node:        &wfv1.NodeStatus{TemplateName: templateName},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "failed since init container failed",
	}, {
		name: "pod failed - init container failed but neither wait nor main containers are finished",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				InitContainerStatuses: []apiv1.ContainerStatus{
					{
						Name:  common.InitContainerName,
						State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}},
					},
				},
				ContainerStatuses: []apiv1.ContainerStatus{
					{
						Name:  common.WaitContainerName,
						State: apiv1.ContainerState{Terminated: nil},
					},
					{
						Name:  common.MainContainerName,
						State: apiv1.ContainerState{Terminated: nil},
					},
				},
				Message: "failed since init container failed",
				Phase:   apiv1.PodFailed,
			},
		},
		node:        &wfv1.NodeStatus{TemplateName: templateName},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "failed since init container failed",
	}, {
		name: "pod failed - init container with non-standard init container name failed but neither wait nor main containers are finished",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				InitContainerStatuses: []apiv1.ContainerStatus{
					{
						Name:  common.InitContainerName,
						State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 0}},
					},
					{
						Name:  "random-init-container",
						State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}},
					},
				},
				ContainerStatuses: []apiv1.ContainerStatus{
					{
						Name:  common.WaitContainerName,
						State: apiv1.ContainerState{Terminated: nil},
					},
					{
						Name:  common.MainContainerName,
						State: apiv1.ContainerState{Terminated: nil},
					},
				},
				Message: "failed since init container failed",
				Phase:   apiv1.PodFailed,
			},
		},
		node:        &wfv1.NodeStatus{TemplateName: templateName},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "failed since init container failed",
	}, {
		name: "pod failed - wait container waiting but pod was set failed",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				InitContainerStatuses: []apiv1.ContainerStatus{
					{
						Name:  common.InitContainerName,
						State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 0}},
					},
				},
				ContainerStatuses: []apiv1.ContainerStatus{
					{
						Name:  common.WaitContainerName,
						State: apiv1.ContainerState{Terminated: nil, Waiting: &apiv1.ContainerStateWaiting{Reason: "PodInitializing"}},
					},
					{
						Name:  common.MainContainerName,
						State: apiv1.ContainerState{Terminated: nil},
					},
				},
				Message: "failed since wait contain waiting",
				Phase:   apiv1.PodFailed,
			},
		},
		node:        &wfv1.NodeStatus{TemplateName: templateName},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "failed since wait contain waiting",
	}, {
		name: "pod running",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				Phase: apiv1.PodRunning,
			},
		},
		node:        &wfv1.NodeStatus{TemplateName: templateName},
		wantPhase:   wfv1.NodeRunning,
		wantMessage: "",
	}, {
		name:    "empty phase",
		pod:     &apiv1.Pod{},
		node:    &wfv1.NodeStatus{TemplateName: templateName},
		wantNil: true,
	}, {
		name: "pod failed - main container still running, init container finished",
		pod: &apiv1.Pod{
			Status: apiv1.PodStatus{
				InitContainerStatuses: []apiv1.ContainerStatus{
					{
						Name:  common.InitContainerName,
						State: apiv1.ContainerState{Terminated: nil},
					},
				},
				ContainerStatuses: []apiv1.ContainerStatus{
					{
						Name:  common.WaitContainerName,
						State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{StartedAt: metav1.Time{Time: time.Now()}}},
					},
					{
						Name:  common.MainContainerName,
						State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{StartedAt: metav1.Time{Time: time.Now()}}},
					},
				},
				Message: "Pod Failed",
				Phase:   apiv1.PodFailed,
			},
		},
		node:        &wfv1.NodeStatus{TemplateName: templateName},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "Pod Failed",
	}}

	nonDaemonWf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	daemonWf := wfv1.MustUnmarshalWorkflow(helloDaemonWf)

	ctx := logging.TestContext(t.Context())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := nonDaemonWf
			if tt.daemon {
				wf = daemonWf
			}
			assert.Equal(t, tt.daemon, wf.GetTemplateByName(tt.node.TemplateName).IsDaemon(), "check the test case is valid")
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			got := woc.assessNodeStatus(ctx, tt.pod, tt.node)
			if tt.wantNil {
				assert.Nil(t, got)
				return
			}
			if got == nil {
				got = tt.node // nil means the node is unchanged
			}
			assert.Equal(t, tt.wantPhase, got.Phase)
			assert.Equal(t, tt.wantMessage, got.Message)
		})
	}
}

// TestAssessNodeStatusDaemonContainerSetTeardown verifies that when killDaemonedChildren has
// stopped a daemon container set pod (marking its node and unfinished container children
// Succeeded before terminating the pod), the resulting PodFailed event and the containers'
// kill exit codes neither fail the pod node nor the container child nodes, while a container
// that genuinely failed beforehand keeps its Failed phase.
// See https://github.com/argoproj/argo-workflows/issues/16397.
func TestAssessNodeStatusDaemonContainerSetTeardown(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/daemon-container-set-wf.yaml")
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	const podNodeName = "daemon-container-set"
	podNode := &wfv1.NodeStatus{
		ID:           woc.wf.NodeID(podNodeName),
		Name:         podNodeName,
		Type:         wfv1.NodeTypePod,
		TemplateName: "main",
		// set by killDaemonedChildren before the pod was terminated
		Phase: wfv1.NodeSucceeded,
	}
	woc.wf.Status.Nodes.Set(ctx, podNode.ID, *podNode)
	for name, phase := range map[string]wfv1.NodePhase{
		"step-1": wfv1.NodeSucceeded, // completed by killDaemonedChildren during the teardown
		"step-2": wfv1.NodeFailed,    // genuinely failed before the teardown
	} {
		childName := podNodeName + "." + name
		woc.wf.Status.Nodes.Set(ctx, woc.wf.NodeID(childName), wfv1.NodeStatus{
			ID:           woc.wf.NodeID(childName),
			Name:         childName,
			Type:         wfv1.NodeTypeContainer,
			TemplateName: "main",
			BoundaryID:   podNode.ID,
			Phase:        phase,
		})
	}
	pod := &apiv1.Pod{
		Status: apiv1.PodStatus{
			Phase: apiv1.PodFailed,
			ContainerStatuses: []apiv1.ContainerStatus{
				{Name: "step-1", State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 137, Reason: "Error"}}},
				{Name: "step-2", State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1, Reason: "Error"}}},
			},
		},
	}

	got := woc.assessNodeStatus(ctx, pod, podNode)
	require.NotNil(t, got)
	assert.Equal(t, wfv1.NodeSucceeded, got.Phase, "pod node stays Succeeded")

	step1, err := woc.wf.GetNodeByName(podNodeName + ".step-1")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeSucceeded, step1.Phase, "container completed by the teardown must not be failed by its kill exit code")

	step2, err := woc.wf.GetNodeByName(podNodeName + ".step-2")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, step2.Phase, "container that failed before the teardown stays Failed")
}

// injectPlaceholderTaskResult mirrors what the executor writes at pod start: a WorkflowTaskResult
// labelled report-outputs-completed=false, before any output has been reported. Every pod with an
// aux container has one from the moment it starts, and it is what flips a node's TaskResultSynced
// to false. Without it a teardown test cannot tell whether a daemon node was initialized with
// omitTaskResultSynced, because MarkTaskResultIncomplete leaves a nil TaskResultSynced alone.
func injectPlaceholderTaskResult(ctx context.Context, t *testing.T, woc *wfOperationCtx, nodeID string) {
	t.Helper()
	result := &wfv1.WorkflowTaskResult{
		TypeMeta: metav1.TypeMeta{
			APIVersion: workflow.APIVersion,
			Kind:       workflow.WorkflowTaskResultKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeID,
			Namespace: woc.wf.Namespace,
			Labels: map[string]string{
				common.LabelKeyWorkflow:               woc.wf.Name,
				common.LabelKeyReportOutputsCompleted: "false",
			},
		},
	}
	_, err := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(woc.wf.Namespace).
		Create(ctx, result, metav1.CreateOptions{})
	require.NoError(t, err)
	// The informer is fed asynchronously from the fake clientset; seed the indexer directly so the
	// next operate's taskResultReconciliation sees the placeholder.
	require.NoError(t, woc.controller.taskResultInformer.GetIndexer().Add(result))
}

// TestDaemonTeardownEndToEnd drives the whole teardown sequence through the real code path, rather
// than hand-building node status: operate, bring the daemon pod up Running and Ready, let
// killDaemonedChildren stop it when the step group completes, deliver the PodRunning event that is
// still in flight at that moment, then the PodFailed event that Kubernetes reports because the
// containers were SIGKILLed.
//
// This is the daemon case raised in review on
// https://github.com/argoproj/argo-workflows/pull/16396. Three things must hold at the end:
// the pod node stays Succeeded, killDaemonedChildren's completed container children are not
// dragged to Failed by the kill's exit codes, and no exit code from our own kill is recorded.
//
// The placeholder WorkflowTaskResult is what makes this test able to fail. Without it
// TaskResultSynced never becomes false, the node stays Fulfilled through teardown, and the
// PodRunning branch below never resurrects it -- so initializing the node without
// omitTaskResultSynced would go unnoticed.
func TestDaemonTeardownEndToEnd(t *testing.T) {
	for _, tt := range []struct {
		name       string
		wf         string
		daemonNode string
		ctrNodes   []string
	}{{
		// a container set pod has no container called "main", so getExitCode never matches it
		name:       "container set",
		wf:         "@testdata/operator/daemon-container-set-steps-wf.yaml",
		daemonNode: "daemon-cs-steps[0].daemon",
		ctrNodes:   []string{"step-1", "step-2"},
	}, {
		// a script pod does have a "main" container, so it is the case that can record an exit code
		name:       "script",
		wf:         "@testdata/operator/daemon-script-steps-wf.yaml",
		daemonNode: "daemon-script-steps[0].daemon",
	}} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := wfv1.MustUnmarshalWorkflow(tt.wf)
			cancel, controller := newController(ctx, wf)
			defer cancel()

			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)

			daemonNode, err := woc.wf.GetNodeByName(tt.daemonNode)
			require.NoError(t, err)
			injectPlaceholderTaskResult(ctx, t, woc, daemonNode.ID)

			// Bring the daemon pod up: Running with every container Ready, which is what makes
			// assessNodeStatus mark the node Daemoned.
			pods, err := listPods(ctx, woc)
			require.NoError(t, err)
			require.NotEmpty(t, pods.Items)
			for _, p := range pods.Items {
				pod := p
				pod.Status.Phase = apiv1.PodRunning
				// The fake client does not populate ContainerStatuses, so synthesize what a kubelet reports.
				pod.Status.ContainerStatuses = nil
				for _, c := range pod.Spec.Containers {
					pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, apiv1.ContainerStatus{
						Name:  c.Name,
						Ready: true,
						State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
					})
				}
				updatedPod, updateErr := controller.kubeclientset.CoreV1().Pods(pod.Namespace).Update(ctx, &pod, metav1.UpdateOptions{})
				require.NoError(t, updateErr)
				// operate reads pods from the informer cache, which the fake clientset feeds
				// asynchronously; update the store directly so the next operate sees the new status.
				require.NoError(t, controller.PodController.TestingPodInformer().GetStore().Update(updatedPod))
			}
			woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
			woc.operate(ctx)

			daemonNode, err = woc.wf.GetNodeByName(tt.daemonNode)
			require.NoError(t, err)
			require.True(t, daemonNode.IsDaemoned(), "daemon pod node should be Daemoned before teardown")

			// The step group has completed, so the daemon is stopped. This marks the pod node Succeeded,
			// completes the unfinished container children, and signals the pod.
			woc.killDaemonedChildren(ctx, daemonNode.BoundaryID)

			daemonNode, err = woc.wf.GetNodeByName(tt.daemonNode)
			require.NoError(t, err)
			require.Equal(t, wfv1.NodeSucceeded, daemonNode.Phase, "killDaemonedChildren should mark the pod node Succeeded")
			for _, ctr := range tt.ctrNodes {
				child, childErr := woc.wf.GetNodeByName(tt.daemonNode + "." + ctr)
				require.NoError(t, childErr)
				require.Equal(t, wfv1.NodeSucceeded, child.Phase,
					"killDaemonedChildren should complete container node %s with the pod node", ctr)
			}

			pods, err = listPods(ctx, woc)
			require.NoError(t, err)
			var daemonPod *apiv1.Pod
			for i := range pods.Items {
				if pods.Items[i].Annotations[common.AnnotationKeyNodeID] == daemonNode.ID {
					daemonPod = &pods.Items[i]
				}
			}
			require.NotNil(t, daemonPod, "daemon pod")

			// assessNodeStatus returns nil when nothing changed; mirror podReconciliation's contract
			// and store whatever comes back, so every assertion below reads the stored node.
			assess := func() *wfv1.NodeStatus {
				old, getErr := woc.wf.GetNodeByName(tt.daemonNode)
				require.NoError(t, getErr)
				if updated := woc.assessNodeStatus(ctx, daemonPod, old); updated != nil {
					woc.wf.Status.Nodes.Set(ctx, updated.ID, *updated)
				}
				stored, getErr := woc.wf.GetNodeByName(tt.daemonNode)
				require.NoError(t, getErr)
				return stored
			}

			// The kill is queued, not yet delivered: the pod is still Running when the next reconcile
			// lands. This is where a node whose TaskResultSynced is false gets resurrected to
			// Running+Daemoned, which makes the following PodFailed look like a spontaneous death.
			require.Equal(t, wfv1.NodeSucceeded, assess().Phase, "a stopped daemon must not be resurrected to Running")

			// Kubernetes now reports the pod Failed, its containers SIGKILLed by the teardown.
			daemonPod.Status.Phase = apiv1.PodFailed
			for i := range daemonPod.Status.ContainerStatuses {
				daemonPod.Status.ContainerStatuses[i].State = apiv1.ContainerState{
					Terminated: &apiv1.ContainerStateTerminated{ExitCode: 137, Reason: "Error"},
				}
			}

			storedNode := assess()
			assert.Equal(t, wfv1.NodeSucceeded, storedNode.Phase, "daemon pod node should stay Succeeded")
			var exitCode *string
			if storedNode.Outputs != nil {
				exitCode = storedNode.Outputs.ExitCode
			}
			assert.Nil(t, exitCode, "a daemon stopped by the controller must not record our own kill's exit code")

			// The container children must not be dragged to Failed by the teardown.
			for _, ctr := range tt.ctrNodes {
				child, childErr := woc.wf.GetNodeByName(tt.daemonNode + "." + ctr)
				require.NoError(t, childErr)
				assert.Equal(t, wfv1.NodeSucceeded, child.Phase,
					"container node %s must not be Failed by the intentional teardown", ctr)
			}
		})
	}
}

// TestGetNodeTemplateUsesStoredTemplates verifies that GetNodeTemplate checks storedTemplates
// before falling back to the live CWT informer cache. This prevents assessNodeStatus from
// spuriously marking a node as Error when the CWT is modified after the workflow starts.
func TestGetNodeTemplateUsesStoredTemplates(t *testing.T) {
	// A minimal workflow whose storedTemplates holds a cluster-scoped template.
	// The CWT passed to newController intentionally does NOT contain the template,
	// simulating the template having been removed from the CWT mid-run.
	const wfYAML = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: cwt-templateref-bug
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: hello-task
        templateRef:
          name: cwt-helper
          template: say-hello
          clusterScope: true
status:
  storedTemplates:
    cluster/cwt-helper/say-hello:
      name: say-hello
      script:
        image: alpine:3.18
        command: [sh]
        source: echo hello
`
	// CWT with the template removed (simulates post-creation mutation).
	const cwftYAML = `
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cwt-helper
spec:
  templates: []
`
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(wfYAML)
	cwft := wfv1.MustUnmarshalClusterWorkflowTemplate(cwftYAML)

	cancel, controller := newController(ctx, wf, cwft)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	node := &wfv1.NodeStatus{
		Name:          "cwt-templateref-bug.hello-task",
		TemplateScope: "cluster/cwt-helper",
		TemplateRef: &wfv1.TemplateRef{
			Name:         "cwt-helper",
			Template:     "say-hello",
			ClusterScope: true,
		},
	}

	tmpl, err := woc.GetNodeTemplate(ctx, node)
	require.NoError(t, err, "GetNodeTemplate should return template from storedTemplates, not fail because CWT no longer has it")
	require.NotNil(t, tmpl)
	assert.Equal(t, "say-hello", tmpl.Name)
}

// TestAssessNodeStatusWithTemplateRefUsesStoredTemplates verifies that assessNodeStatus
// correctly resolves the phase from the pod when a cross-CWT templateRef node's template
// has been removed from the CWT after pod creation. The node must not be marked Error.
func TestAssessNodeStatusWithTemplateRefUsesStoredTemplates(t *testing.T) {
	const wfYAML = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: cwt-templateref-bug
  namespace: default
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: hello-task
        templateRef:
          name: cwt-helper
          template: say-hello
          clusterScope: true
status:
  storedTemplates:
    cluster/cwt-helper/say-hello:
      name: say-hello
      script:
        image: alpine:3.18
        command: [sh]
        source: echo hello
`
	const cwftYAML = `
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cwt-helper
spec:
  templates: []
`
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(wfYAML)
	cwft := wfv1.MustUnmarshalClusterWorkflowTemplate(cwftYAML)

	cancel, controller := newController(ctx, wf, cwft)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	pod := &apiv1.Pod{
		Status: apiv1.PodStatus{
			Phase: apiv1.PodSucceeded,
		},
	}
	node := &wfv1.NodeStatus{
		Name:          "cwt-templateref-bug.hello-task",
		TemplateScope: "cluster/cwt-helper",
		TemplateRef: &wfv1.TemplateRef{
			Name:         "cwt-helper",
			Template:     "say-hello",
			ClusterScope: true,
		},
	}

	updated := woc.assessNodeStatus(ctx, pod, node)
	require.NotNil(t, updated, "assessNodeStatus must not return nil when template is in storedTemplates")
	assert.Equal(t, wfv1.NodeSucceeded, updated.Phase, "node should be Succeeded, not Error, when pod succeeded and template is in storedTemplates")
}

func getPodTemplate(pod *apiv1.Pod) (*wfv1.Template, error) {
	tmpl := &wfv1.Template{}
	for _, c := range pod.Spec.InitContainers {
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
				InitContainers: []apiv1.Container{
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
				InitContainers: []apiv1.Container{
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
				InitContainers: []apiv1.Container{
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

// TestWorkflowStepRetry verifies that steps retry will restart from the 0th step
func TestWorkflowStepRetry(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-step-retry.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)

	// complete the first pod
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	nodeID := woc.nodeID(&pods.Items[0])
	woc.wf.Status.MarkTaskResultComplete(ctx, nodeID)
	woc.operate(ctx)

	// fail the second pod
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 3)
	assert.Equal(t, "cowsay success", pods.Items[0].Spec.Containers[1].Args[0])
	assert.Equal(t, "cowsay failure", pods.Items[1].Spec.Containers[1].Args[0])

	// verify that after the cowsay failure pod failed, we are retrying cowsay success
	assert.Equal(t, "cowsay success", pods.Items[2].Spec.Containers[1].Args[0])
}

// TestWorkflowParallelismLimit verifies parallelism at a workflow level is honored.
func TestWorkflowParallelismLimit(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-parallelism-limit.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 2)

	makePodsPhase(ctx, woc, apiv1.PodRunning)

	// operate again and make sure we don't schedule any more pods
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
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
      image: alpine:3.23
      command: [sh, -c, sleep 10]
`

// TestStepsTemplateParallelismLimit verifies parallelism at a steps level is honored.
func TestStepsTemplateParallelismLimit(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow(stepsTemplateParallelismLimit)
	ctx := logging.TestContext(t.Context())
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 2)

	// operate again and make sure we don't schedule any more pods
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	// wfBytes, _ := json.MarshalIndent(wf, "", "  ")
	// log.Printf("%s", wfBytes)
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 2)
}

// TestDAGTemplateParallelismLimit verifies parallelism at a dag level is honored.
func TestDAGTemplateParallelismLimit(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/dag-template-parallelism-limit.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 2)

	// operate again and make sure we don't schedule any more pods
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 2)
}

func TestNestedTemplateParallelismLimit(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/nested-parallelism.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 4)
}

// TestSidecarResourceLimits verifies resource limits on the sidecar can be set in the controller config
func TestSidecarResourceLimits(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
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
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows("").Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pod, err := getPod(ctx, woc, "hello-world")
	require.NoError(t, err)
	var waitCtr *apiv1.Container
	for _, ctr := range pod.Spec.Containers {
		if ctr.Name == "wait" {
			waitCtr = &ctr
			break
		}
	}
	require.NotNil(t, waitCtr)
	require.NotNil(t, waitCtr.Resources)
	assert.Len(t, waitCtr.Resources.Limits, 2)
	assert.Len(t, waitCtr.Resources.Requests, 2)
}

// TestSuspendResume tests the suspend and resume feature
func TestSuspendResume(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(stepsTemplateParallelismLimit)
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// suspend the workflow
	ctx := logging.TestContext(t.Context())
	err := util.SuspendWorkflow(ctx, wfcset, wf.Name)
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.True(t, *wf.Spec.Suspend)

	// operate should not result in no workflows being created since it is suspended
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Empty(t, pods.Items)

	// resume the workflow and operate again. two pods should be able to be scheduled
	err = util.ResumeWorkflow(ctx, wfcset, controller.hydrator, wf.Name, "")
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Nil(t, wf.Spec.Suspend)
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 2)
}

func suspendedShutdownWorkflow() *wfv1.Workflow {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/no-pods-when-shutdown.yaml")
	wf.Spec.Shutdown = ""
	wf.Spec.OnExit = "exit-handler"
	wf.Spec.Templates = append(wf.Spec.Templates, wfv1.Template{
		Name: "exit-handler",
		Container: &apiv1.Container{
			Image:   "docker/whalesay:latest",
			Command: []string{"cowsay"},
			Args:    []string{"goodbye world"},
		},
	})
	return wf
}

func TestTerminateOnSuspendedWorkflow(t *testing.T) {
	t.Run("StartedSuspended", func(t *testing.T) {
		wf := suspendedShutdownWorkflow()
		wf.Spec.Suspend = new(true)
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		assert.Empty(t, woc.wf.Status.Nodes)
		pods, err := listPods(ctx, woc)
		require.NoError(t, err)
		assert.Empty(t, pods.Items)

		wf = woc.wf.DeepCopy()
		wf.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)

		node := woc.wf.Status.Nodes.FindByDisplayName("hello-world")
		require.NotNil(t, node)
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
		assert.Contains(t, node.Message, "workflow shutdown with strategy")
		assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("hello-world.onExit"), "terminate must not run the exit handler")
		pods, err = listPods(ctx, woc)
		require.NoError(t, err)
		assert.Empty(t, pods.Items)

		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Equal(t, "Stopped with strategy 'Terminate'", woc.wf.Status.Message)
		persistedWf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.True(t, *persistedWf.Spec.Suspend)
	})

	t.Run("SuspendedWhileRunning", func(t *testing.T) {
		wf := suspendedShutdownWorkflow()
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		node := woc.wf.Status.Nodes.FindByDisplayName("hello-world")
		require.NotNil(t, node)
		assert.Equal(t, wfv1.NodePending, node.Phase)
		makePodsPhase(ctx, woc, apiv1.PodPending)

		wf = woc.wf.DeepCopy()
		wf.Spec.Suspend = new(true)
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

		wf = woc.wf.DeepCopy()
		wf.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)

		node = woc.wf.Status.Nodes.FindByDisplayName("hello-world")
		require.NotNil(t, node)
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
		assert.Contains(t, node.Message, "workflow shutdown with strategy")
		assert.Nil(t, woc.wf.Status.Nodes.FindByDisplayName("hello-world.onExit"), "terminate must not run the exit handler")

		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Equal(t, "Stopped with strategy 'Terminate'", woc.wf.Status.Message)
		persistedWf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.True(t, *persistedWf.Spec.Suspend)
	})
}

func TestStopOnSuspendedWorkflow(t *testing.T) {
	t.Run("StartedSuspended", func(t *testing.T) {
		wf := suspendedShutdownWorkflow()
		wf.Spec.Suspend = new(true)
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		assert.Empty(t, woc.wf.Status.Nodes)
		pods, err := listPods(ctx, woc)
		require.NoError(t, err)
		assert.Empty(t, pods.Items)

		wf = woc.wf.DeepCopy()
		wf.Spec.Shutdown = wfv1.ShutdownStrategyStop
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)

		node := woc.wf.Status.Nodes.FindByDisplayName("hello-world")
		require.NotNil(t, node)
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
		assert.Contains(t, node.Message, "workflow shutdown with strategy")
		onExitNode := woc.wf.Status.Nodes.FindByDisplayName("hello-world.onExit")
		require.NotNil(t, onExitNode, "stop must run the exit handler")
		assert.Equal(t, wfv1.NodePending, onExitNode.Phase)
		pods, err = listPods(ctx, woc)
		require.NoError(t, err)
		assert.Len(t, pods.Items, 1)

		makePodsPhase(ctx, woc, apiv1.PodSucceeded)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)

		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Equal(t, "Stopped with strategy 'Stop'", woc.wf.Status.Message)
		persistedWf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.True(t, *persistedWf.Spec.Suspend)
	})

	t.Run("SuspendedWhileRunning", func(t *testing.T) {
		wf := suspendedShutdownWorkflow()
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		node := woc.wf.Status.Nodes.FindByDisplayName("hello-world")
		require.NotNil(t, node)
		assert.Equal(t, wfv1.NodePending, node.Phase)
		makePodsPhase(ctx, woc, apiv1.PodPending)

		wf = woc.wf.DeepCopy()
		wf.Spec.Suspend = new(true)
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

		wf = woc.wf.DeepCopy()
		wf.Spec.Shutdown = wfv1.ShutdownStrategyStop
		woc = newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)

		node = woc.wf.Status.Nodes.FindByDisplayName("hello-world")
		require.NotNil(t, node)
		assert.Equal(t, wfv1.NodeFailed, node.Phase)
		assert.Contains(t, node.Message, "workflow shutdown with strategy")
		onExitNode := woc.wf.Status.Nodes.FindByDisplayName("hello-world.onExit")
		require.NotNil(t, onExitNode, "stop must run the exit handler")
		assert.Equal(t, wfv1.NodePending, onExitNode.Phase)

		setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
			if n.NodeFlag != nil && n.NodeFlag.Hooked {
				return apiv1.PodSucceeded
			}
			return ""
		})
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)

		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Equal(t, "Stopped with strategy 'Stop'", woc.wf.Status.Message)
		persistedWf, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
		require.NoError(t, err)
		assert.True(t, *persistedWf.Spec.Suspend)
	})
}

func TestSuspendWithDeadline(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/suspend-template-with-deadline.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	updatedWf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	found := false

	for _, node := range updatedWf.Status.Nodes {
		if node.Type == wfv1.NodeTypeSuspend {
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
			assert.Contains(t, node.Message, "Step exceeded its deadline")
			found = true
		}
	}
	assert.True(t, found)
}

func TestSuspendInputsResolution(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/suspend-template-input-resolution.yaml")
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.FindByDisplayName("suspend-template")

	assert.Equal(t, wfv1.NodeTypeSuspend, node.Type)
	assert.Equal(t, wfv1.NodeRunning, node.Phase)

	assert.Equal(t, "param1", node.Inputs.Parameters[0].Name)
	assert.JSONEq(t, "{\"enum\": [\"one\", \"two\", \"three\"]}", node.Inputs.Parameters[0].Value.String())
	assert.Len(t, node.Inputs.Parameters[0].Enum, 3)
	assert.Equal(t, "one", node.Inputs.Parameters[0].Enum[0].String())
	assert.Equal(t, "two", node.Inputs.Parameters[0].Enum[1].String())
	assert.Equal(t, "three", node.Inputs.Parameters[0].Enum[2].String())

	assert.Equal(t, "param2", node.Inputs.Parameters[1].Name)
	assert.Equal(t, "value2", node.Inputs.Parameters[1].Value.String())
}

func TestSequence(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/sequence.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	updatedWf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	found100 := false
	found101 := false
	for _, node := range updatedWf.Status.Nodes {
		switch node.DisplayName {
		case "step1(0:100)":
			assert.Equal(t, "100", node.Inputs.Parameters[0].Value.String())
			found100 = true
		case "step1(1:101)":
			assert.Equal(t, "101", node.Inputs.Parameters[0].Value.String())
			found101 = true
		}
	}
	assert.True(t, found100)
	assert.True(t, found101)
}

func TestInputParametersAsJson(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/input-parameters-as-json.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	updatedWf, err := wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	found := false
	for _, node := range updatedWf.Status.Nodes {
		if node.Type == wfv1.NodeTypePod {
			expected := `Workflow: [{"name":"parameter1","value":"value1"}]. Template: [{"name":"parameter1","value":"value1"},{"name":"parameter2","value":"template2"}]`
			assert.Equal(t, expected, node.Inputs.Parameters[0].Value.String())
			found = true
		}
	}
	assert.True(t, found)
}

func TestExpandWithItems(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/expand-with-items.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	step := wf.Spec.Templates[0].Steps[0].Steps[0]
	dagTask := wfv1.DAGTask{
		Name:      step.Name,
		Template:  step.Template,
		Arguments: step.Arguments,
		WithItems: step.WithItems,
	}
	expandedTasks, err := dag.ExpandTask(ctx, dagTask, woc.globalParams(), woc)
	require.NoError(t, err)
	assert.Len(t, expandedTasks, 5)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 5)
}

func TestExpandWithItemsMap(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/expand-with-items-map.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	step := wf.Spec.Templates[0].Steps[0].Steps[0]
	dagTask := wfv1.DAGTask{
		Name:      step.Name,
		Template:  step.Template,
		Arguments: step.Arguments,
		WithItems: step.WithItems,
	}
	expandedTasks, err := dag.ExpandTask(ctx, dagTask, woc.globalParams(), woc)
	require.NoError(t, err)
	assert.Len(t, expandedTasks, 3)
	assert.Equal(t, "debian 9.1 JSON({\"os\":\"debian\",\"version\":9.1})", expandedTasks[0].Arguments.Parameters[0].Value.String())
}

func TestSuspendTemplate(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/suspend-template.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Empty(t, pods.Items)

	// resume the workflow. verify resume workflow edits nodestatus correctly
	err = util.ResumeWorkflow(ctx, wfcset, controller.hydrator, wf.Name, "")
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.False(t, util.IsWorkflowSuspended(wf))

	// operate the workflow. it should reach the second step
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

func TestSuspendTemplateWithFailedResume(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/suspend-template.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Empty(t, pods.Items)

	// resume the workflow. verify resume workflow edits nodestatus correctly
	err = util.StopWorkflow(ctx, wfcset, controller.hydrator, wf.Name, "inputs.parameters.param1.value=value1", "Step failed!")
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.False(t, util.IsWorkflowSuspended(wf))

	// operate the workflow. it should be failed and not reach the second step
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Empty(t, pods.Items)
}

func TestSuspendTemplateWithFilteredResume(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/suspend-template.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Empty(t, pods.Items)

	// resume the workflow, but with non-matching selector
	err = util.ResumeWorkflow(ctx, wfcset, controller.hydrator, wf.Name, "inputs.paramaters.param1.value=value2")
	require.Error(t, err)

	// operate the workflow. nothing should have happened
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Empty(t, pods.Items)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// resume the workflow, but with matching selector
	err = util.ResumeWorkflow(ctx, wfcset, controller.hydrator, wf.Name, "inputs.parameters.param1.value=value1")
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.False(t, util.IsWorkflowSuspended(wf))

	// operate the workflow. it should reach the second step
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

func TestSuspendResumeAfterTemplate(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/suspend-resume-after-template.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Empty(t, pods.Items)

	// wait 4 seconds
	time.Sleep(4 * time.Second)

	// operate the workflow. it should reach the second step
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

func TestSuspendResumeAfterTemplateNoWait(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should become in a suspended state after
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/suspend-resume-after-template.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.True(t, util.IsWorkflowSuspended(wf))

	// operate again and verify no pods were scheduled
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Empty(t, pods.Items)

	// don't wait

	// operate the workflow. it should have not reached the second step since not enough time passed
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Empty(t, pods.Items)
}

// TestSuspendTimeoutWithInputDefaults tests that when a suspend node times out,
// output parameters get their default values from matching input parameters
func TestSuspendTimeoutWithInputDefaults(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/suspend-with-input-defaults-template.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	// operate the workflow - it should create the suspend node and immediately timeout (duration: 0s)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)

	// Get the suspend node and verify outputs have defaults from inputs
	waitNode := wf.Status.Nodes.FindByDisplayName("wait")
	require.NotNil(t, waitNode)
	require.NotNil(t, waitNode.Outputs)
	require.Len(t, waitNode.Outputs.Parameters, 2)

	// Verify "input" output parameter got the default "Hello World" from input
	var inputParam, selectParam *wfv1.Parameter
	for i := range waitNode.Outputs.Parameters {
		p := &waitNode.Outputs.Parameters[i]
		switch p.Name {
		case "input":
			inputParam = p
		case "select":
			selectParam = p
		}
	}

	require.NotNil(t, inputParam, "input parameter should exist in outputs")
	require.NotNil(t, inputParam.Value, "input parameter should have a value")
	assert.Equal(t, "Hello World", inputParam.Value.String())

	// Verify "select" output parameter got the default "default" from input
	require.NotNil(t, selectParam, "select parameter should exist in outputs")
	require.NotNil(t, selectParam.Value, "select parameter should have a value")
	assert.Equal(t, "default", selectParam.Value.String())

	// Verify the workflow eventually succeeds (continues to next step)
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

// Tests ability to reference workflow parameters from within top level spec fields (e.g. spec.volumes)
func TestWorkflowSpecParam(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/volume-with-param.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	pod, err := getPod(ctx, woc, wf.Name)
	require.NoError(t, err)
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

func TestWokflowSchedulingConstraintsDAG(t *testing.T) {
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator/workflow-scheduling-constraints-template-dag.yaml")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-scheduling-constraints-dag.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf, wftmpl)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 2)
	for _, pod := range pods.Items {
		assert.Equal(t, "workflows", pod.Spec.NodeSelector["pool"])
		found := false
		value := ""
		for _, toleration := range pod.Spec.Tolerations {
			if toleration.Key == "pool" {
				found = true
				value = toleration.Value
			}
		}
		assert.True(t, found)
		assert.Equal(t, "workflows", value)
		assert.NotNil(t, pod.Spec.Affinity)
		assert.Equal(t, "node_group", pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Key)
		assert.Contains(t, pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values, "argo-workflow")
	}
}

func TestWokflowSchedulingConstraintsSteps(t *testing.T) {
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator/workflow-scheduling-constraints-template-steps.yaml")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-scheduling-constraints-steps.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf, wftmpl)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 2)
	for _, pod := range pods.Items {
		assert.Equal(t, "workflows", pod.Spec.NodeSelector["pool"])
		found := false
		value := ""
		for _, toleration := range pod.Spec.Tolerations {
			if toleration.Key == "pool" {
				found = true
				value = toleration.Value
			}
		}
		assert.True(t, found)
		assert.Equal(t, "workflows", value)
		assert.NotNil(t, pod.Spec.Affinity)
		assert.Equal(t, "node_group", pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Key)
		assert.Contains(t, pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values, "argo-workflow")
	}
}

func TestAddGlobalParamToScope(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	testVal := wfv1.AnyStringPtr("test-value")
	param := wfv1.Parameter{
		Name:  "test-param",
		Value: testVal,
	}
	// Make sure if the param is not global, don't add to scope
	woc.addParamToGlobalScope(ctx, param)
	assert.Nil(t, woc.wf.Status.Outputs)

	// Now set it as global. Verify it is added to workflow outputs
	param.GlobalName = "global-param"
	woc.addParamToGlobalScope(ctx, param)
	assert.Len(t, woc.wf.Status.Outputs.Parameters, 1)
	assert.Equal(t, param.GlobalName, woc.wf.Status.Outputs.Parameters[0].Name)
	assert.Equal(t, testVal, woc.wf.Status.Outputs.Parameters[0].Value)
	assert.Equal(t, testVal.String(), woc.globalParams()["workflow.outputs.parameters.global-param"])

	// Change the value and verify it is reflected in workflow outputs
	newValue := wfv1.AnyStringPtr("new-value")
	param.Value = newValue
	woc.addParamToGlobalScope(ctx, param)
	assert.Len(t, woc.wf.Status.Outputs.Parameters, 1)
	assert.Equal(t, param.GlobalName, woc.wf.Status.Outputs.Parameters[0].Name)
	assert.Equal(t, newValue, woc.wf.Status.Outputs.Parameters[0].Value)
	assert.Equal(t, newValue.String(), woc.globalParams()["workflow.outputs.parameters.global-param"])

	// Add a new global parameter
	param.GlobalName = "global-param2"
	woc.addParamToGlobalScope(ctx, param)
	assert.Len(t, woc.wf.Status.Outputs.Parameters, 2)
	assert.Equal(t, param.GlobalName, woc.wf.Status.Outputs.Parameters[1].Name)
	assert.Equal(t, newValue, woc.wf.Status.Outputs.Parameters[1].Value)
	assert.Equal(t, newValue.String(), woc.globalParams()["workflow.outputs.parameters.global-param2"])
}

func TestAddGlobalArtifactToScope(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
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
	woc.addArtifactToGlobalScope(ctx, art)
	assert.Nil(t, woc.wf.Status.Outputs)

	// Now mark it as global. Verify it is added to workflow outputs
	art.GlobalName = "global-art"
	woc.addArtifactToGlobalScope(ctx, art)
	assert.Len(t, woc.wf.Status.Outputs.Artifacts, 1)
	assert.Equal(t, art.GlobalName, woc.wf.Status.Outputs.Artifacts[0].Name)
	assert.Equal(t, "some/key", woc.wf.Status.Outputs.Artifacts[0].S3.Key)

	// Change the value and verify update is reflected
	art.S3.Key = "new/key"
	woc.addArtifactToGlobalScope(ctx, art)
	assert.Len(t, woc.wf.Status.Outputs.Artifacts, 1)
	assert.Equal(t, art.GlobalName, woc.wf.Status.Outputs.Artifacts[0].Name)
	assert.Equal(t, "new/key", woc.wf.Status.Outputs.Artifacts[0].S3.Key)

	// Add a new global artifact
	art.GlobalName = "global-art2"
	art.S3.Key = "new/new/key"
	woc.addArtifactToGlobalScope(ctx, art)
	assert.Len(t, woc.wf.Status.Outputs.Artifacts, 2)
	assert.Equal(t, art.GlobalName, woc.wf.Status.Outputs.Artifacts[1].Name)
	assert.Equal(t, "new/new/key", woc.wf.Status.Outputs.Artifacts[1].S3.Key)
}

func TestParamSubstitutionWithArtifact(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@../../test/e2e/functional/param-sub-with-artifacts.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	wf, err := woc.controller.wfclientset.ArgoprojV1alpha1().Workflows("").Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.WorkflowRunning, wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

func TestGlobalParamSubstitutionWithArtifact(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@../../test/e2e/functional/global-param-sub-with-artifacts.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	wf, err := woc.controller.wfclientset.ArgoprojV1alpha1().Workflows("").Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.WorkflowRunning, wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

func TestMetadataPassing(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/metadata-template.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.NotEmpty(t, pods.Items, "pod was not created successfully")

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

func TestResolveIOPathPlaceholders(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/io-path-placeholders.yaml")
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.NotEmpty(t, pods.Items, "pod was not created successfully")

	assert.Equal(t, append(append([]string{"/var/run/argo/argoexec", "emissary"}, woc.getExecutorLogOpts(ctx)...),
		"--", "sh", "-c", "head -n 3 <\"/inputs/text/data\" | tee \"/outputs/text/data\" | wc -l > \"/outputs/actual-lines-count/data\"",
	), pods.Items[0].Spec.Containers[1].Command)
}

func TestResolvePlaceholdersInOutputValues(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/output-value-placeholders.yaml")
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.NotEmpty(t, pods.Items, "pod was not created successfully")

	tmpl, err := getPodTemplate(&pods.Items[0])
	require.NoError(t, err)
	parameterValue := tmpl.Outputs.Parameters[0].Value
	assert.NotNil(t, parameterValue)
	assert.Equal(t, "output-value-placeholders-wf", parameterValue.String())
}

func TestResolvePodNameInRetries(t *testing.T) {
	tests := []struct {
		podNameVersion string
		wantPodName    string
	}{
		{"v1", "output-value-placeholders-wf-3033990984"},
		{"v2", "output-value-placeholders-wf-tell-pod-name-3033990984"},
	}
	for _, tt := range tests {
		t.Setenv("POD_NAMES", tt.podNameVersion)
		ctx := logging.TestContext(t.Context())
		wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/pod-name-in-retries.yaml")
		woc := newWoc(ctx, *wf)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		pods, err := woc.controller.kubeclientset.CoreV1().Pods(wf.ObjectMeta.Namespace).List(ctx, metav1.ListOptions{})
		require.NoError(t, err)
		assert.NotEmpty(t, pods.Items, "pod was not created successfully")

		template, err := getPodTemplate(&pods.Items[0])
		require.NoError(t, err)
		parameterValue := template.Outputs.Parameters[0].Value
		assert.NotNil(t, parameterValue)
		assert.Equal(t, tt.wantPodName, parameterValue.String())
	}
}

func TestResolveStatuses(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/output-statuses.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	jsonValue, err := json.Marshal(&wf.Spec.Templates[0])
	require.NoError(t, err)

	assert.Contains(t, string(jsonValue), "{{steps.first.status}}")
	assert.NotContains(t, string(jsonValue), "{{steps.print.status}}")
}

func TestResourceTemplate(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/resource-template.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.WorkflowRunning, wf.Status.Phase)

	pod, err := getPod(ctx, woc, "resource-template")
	require.NoError(t, err)
	tmpl, err := getPodTemplate(pod)
	require.NoError(t, err)
	cm := apiv1.ConfigMap{}
	err = yaml.Unmarshal([]byte(tmpl.Resource.Manifest), &cm)
	require.NoError(t, err)
	assert.Equal(t, "resource-cm", cm.Name)
	assert.Empty(t, cm.OwnerReferences)
}

func TestResourceWithOwnerReferenceTemplate(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/resource-with-owner-reference-template.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wfv1.WorkflowRunning, wf.Status.Phase)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	objectMetas := map[string]metav1.ObjectMeta{}
	for _, pod := range pods.Items {
		tmpl, err := getPodTemplate(&pod)
		require.NoError(t, err)
		cm := apiv1.ConfigMap{}
		err = yaml.Unmarshal([]byte(tmpl.Resource.Manifest), &cm)
		require.NoError(t, err)
		objectMetas[cm.Name] = cm.ObjectMeta
	}
	require.Len(t, objectMetas["resource-cm-1"].OwnerReferences, 1)
	assert.Equal(t, "manual-ref-name", objectMetas["resource-cm-1"].OwnerReferences[0].Name)

	require.Len(t, objectMetas["resource-cm-2"].OwnerReferences, 1)
	assert.Equal(t, "resource-with-ownerreference-template", objectMetas["resource-cm-2"].OwnerReferences[0].Name)

	require.Len(t, objectMetas["resource-cm-3"].OwnerReferences, 2)
	assert.Equal(t, "manual-ref-name", objectMetas["resource-cm-3"].OwnerReferences[0].Name)
	assert.Equal(t, "resource-with-ownerreference-template", objectMetas["resource-cm-3"].OwnerReferences[1].Name)
}

func TestStepWFGetNodeName(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/step-script-tmpl.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.True(t, woc.hasOutputResultRef(ctx, "generate", &wf.Spec.Templates[0]))
	assert.False(t, woc.hasOutputResultRef(ctx, "print-message", &wf.Spec.Templates[0]))
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, "generate") {
			assert.Equal(t, "generate", getStepOrDAGTaskName(node.Name))
		} else if strings.Contains(node.Name, "print-message") {
			assert.Equal(t, "print-message", getStepOrDAGTaskName(node.Name))
		}
	}
}

func TestDAGWFGetNodeName(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/dag-script-tmpl.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.True(t, woc.hasOutputResultRef(ctx, "A", &wf.Spec.Templates[0]))
	assert.False(t, woc.hasOutputResultRef(ctx, "B", &wf.Spec.Templates[0]))
	woc.operate(ctx)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, ".A") {
			assert.Equal(t, "A", getStepOrDAGTaskName(node.Name))
		}
		if strings.Contains(node.Name, ".B") {
			assert.Equal(t, "B", getStepOrDAGTaskName(node.Name))
		}
	}
}

func TestWithParamAsJsonList(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/with-param-as-json-list.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 4)
}

func TestStepsOnExit(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/steps-on-exit.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.FindByDisplayName("leafA.onExit")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	node = woc.wf.Status.Nodes.FindByDisplayName("leafB.onExit")
	assert.Nil(t, node)
}

func TestStepsOnExitFailures(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/on-exit-failures.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	// Test list expansion
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Contains(t, woc.globalParams()[varkeys.WorkflowFailures.Template()], `[{\"displayName\":\"exit-handlers\",\"message\":\"Pod failed\",\"templateName\":\"intentional-fail\",\"phase\":\"Failed\",\"podName\":\"exit-handlers\"`)
	node := woc.wf.Status.Nodes.FindByDisplayName("exit-handlers")
	assert.NotNil(t, node)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
	assert.Nil(t, node.NodeFlag)
}

func TestStepsOnExitTimeout(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/on-exit-timeout.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.FindByDisplayName("exit-handlers.onExit")
	require.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, node.Phase)
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
		ctx := logging.TestContext(t.Context())
		t.Run(wf.Name, func(t *testing.T) {
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)
			makePodsPhase(ctx, woc, apiv1.PodSucceeded)
			woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
			woc.operate(ctx)
			assert.ElementsMatch(t, want, getEventsWithoutAnnotations(controller, len(want)))
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
		ctx := logging.TestContext(t.Context())
		t.Run(wf.Name, func(t *testing.T) {
			cancel, controller := newController(ctx, wf)
			defer cancel()
			controller.Config.NodeEvents = config.NodeEvents{SendAsPod: true}
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			createRunningPods(ctx, woc)
			woc.operate(ctx)
			makePodsPhase(ctx, woc, apiv1.PodSucceeded)
			woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
			woc.operate(ctx)
			assert.ElementsMatch(t, want, getEventsWithoutAnnotations(controller, len(want)))
		})
	}
}

func getEventsWithoutAnnotations(controller *WorkflowController, num int) []string {
	c := controller.eventRecorderManager.(*testEventRecorderManager).eventRecorder.Events
	events := make([]string, num)
	for i := range num {
		event := <-c
		events[i] = truncateAnnotationsFromEvent(event)
	}
	return events
}

func truncateAnnotationsFromEvent(event string) string {
	before, _, ok := strings.Cut(event, " map[")
	if ok {
		return before
	}
	return event
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
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	createRunningPods(ctx, woc)
	woc.operate(ctx)
	time.Sleep(time.Second)
	// Parent dag node has no pod
	parentNode, err := woc.wf.GetNodeByName("dag-events")
	require.NoError(t, err)
	pod, err := woc.getPodByNode(parentNode)
	assert.Nil(t, pod)
	require.Error(t, err, "Expected node type Pod, got DAG")
	// Pod node should return a pod
	podNode, err := woc.wf.GetNodeByName("dag-events.a")
	require.NoError(t, err)
	pod, err = woc.getPodByNode(podNode)
	require.NoError(t, err)
	assert.NotNil(t, pod)
	// Invalid node should not return a pod
	invalidNode := wfv1.NodeStatus{Type: wfv1.NodeTypePod, Name: "doesnt-exist"}
	pod, err = woc.getPodByNode(&invalidNode)
	assert.Nil(t, pod)
	require.NoError(t, err)
}

func TestPDBCreation(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/pdbwf.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pdb, _ := controller.kubeclientset.PolicyV1().PodDisruptionBudgets("").Get(ctx, woc.wf.Name, metav1.GetOptions{})
	assert.Equal(t, pdb.Name, wf.Name)
	ctx = woc.markWorkflowSuccess(ctx)
	_, err := controller.kubeclientset.PolicyV1().PodDisruptionBudgets("").Get(ctx, woc.wf.Name, metav1.GetOptions{})
	require.EqualError(t, err, "poddisruptionbudgets.policy \"my-pdb-wf\" not found")

	// Test when PDB already exists
	newPDB := policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:   wf.Name,
			Labels: map[string]string{common.LabelKeyWorkflow: wf.Name},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(wf, wfv1.SchemeGroupVersion.WithKind(workflow.WorkflowKind)),
			},
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &intstr.IntOrString{Type: intstr.Int, IntVal: 3, StrVal: "3"},
		},
	}
	_, err = controller.kubeclientset.PolicyV1().PodDisruptionBudgets(wf.Namespace).Create(ctx, &newPDB, metav1.CreateOptions{})
	require.NoError(t, err)

	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pdb, _ = controller.kubeclientset.PolicyV1().PodDisruptionBudgets("").Get(ctx, woc.wf.Name, metav1.GetOptions{})
	assert.Equal(t, pdb.Name, wf.Name)
}

func TestPDBCreationRaceDelete(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/pdbwf.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	err := controller.kubeclientset.PolicyV1().PodDisruptionBudgets("").Delete(ctx, woc.wf.Name, metav1.DeleteOptions{})
	require.NoError(t, err)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestStatusConditions(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/pdbwf.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Empty(t, woc.wf.Status.Conditions)
	_ = woc.markWorkflowSuccess(ctx)
	assert.Equal(t, woc.wf.Status.Conditions[0].Status, metav1.ConditionStatus("True"))
}

func TestNestedOptionalOutputArtifacts(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// Test list expansion
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/nested-optional-output-artifacts.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestPodSpecLogForFailedPods tests PodSpec logging configuration
func TestPodSpecLogForFailedPods(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	controller.Config.PodSpecLogStrategy.FailedPod = true
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	for _, node := range woc.wf.Status.Nodes {
		assert.True(t, woc.shouldPrintPodSpec(&node))
	}
}

// TestPodSpecLogForAllPods tests  PodSpec logging configuration
func TestPodSpecLogForAllPods(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	assert.NotNil(t, controller)
	controller.Config.PodSpecLogStrategy.AllPods = true
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/nested-optional-output-artifacts.yaml")
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotNil(t, woc)
	woc.operate(ctx)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	for _, node := range woc.wf.Status.Nodes {
		assert.True(t, woc.shouldPrintPodSpec(&node))
	}
}

// This tests to see if the outputs of the last child node of a retry node are added correctly to the scope
func TestRetryNodeOutputs(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/retry-node-outputs.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	retryNode, err := woc.wf.GetNodeByName("daemon-step-dvbnn[0].influx")
	require.NoError(t, err)
	assert.NotNil(t, retryNode)
	fmt.Println(retryNode)
	scope := &wfScope{
		scope: variables.NewScope(),
	}
	woc.buildLocalScope(scope, varkeys.StepsNodeRef, "influx", retryNode)
	assert.Contains(t, scope.scope.AsAnyMap(), "steps.influx.ip")
	assert.Contains(t, scope.scope.AsAnyMap(), "steps.influx.id")
	assert.Contains(t, scope.scope.AsAnyMap(), "steps.influx.startedAt")
	assert.Contains(t, scope.scope.AsAnyMap(), "steps.influx.finishedAt")
	assert.Contains(t, scope.scope.AsAnyMap(), "steps.influx.hostNodeName")
}

// This test ensures that the PVCs used in the steps are not deleted when
// the workflow fails
func TestDeletePVCDoesNotDeletePVCOnFailedWorkflow(t *testing.T) {
	assert := assert.New(t)

	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-with-pvc-and-failing-step.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	assert.Len(woc.wf.Status.PersistentVolumeClaims, 1, "1 PVC before operating")

	woc.operate(ctx)

	node1, err := woc.wf.GetNodeByName("wf-with-pvc(0)[0].succeed")
	require.NoError(t, err)
	node2, err := woc.wf.GetNodeByName("wf-with-pvc(0)[1].failure")
	require.NoError(t, err)

	// Node 1 Succeeded
	assert.Equal(wfv1.NodeSucceeded, node1.Phase)
	// Node 2 Failed
	assert.Equal(wfv1.NodeFailed, node2.Phase)
	// Hence, PVCs should stick around
	assert.Len(woc.wf.Status.PersistentVolumeClaims, 1, "PVCs not deleted")
}

func TestContainerOutputsResult(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")

	// operate the workflow. it should create a pod.
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/container-outputs-result.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.True(t, woc.hasOutputResultRef(ctx, "hello1", &wf.Spec.Templates[0]))
	assert.False(t, woc.hasOutputResultRef(ctx, "hello2", &wf.Spec.Templates[0]))
	woc.operate(ctx)

	for _, node := range wf.Status.Nodes {
		if strings.Contains(node.Name, "hello1") {
			assert.Equal(t, "hello1", getStepOrDAGTaskName(node.Name))
		} else if strings.Contains(node.Name, "hello2") {
			assert.Equal(t, "hello2", getStepOrDAGTaskName(node.Name))
		}
	}
}

func TestNestedStepGroupGlobalParams(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/nested-step-group-global-params.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.FindByDisplayName("generate")
	require.NotNil(t, node)
	require.NotNil(t, node.Outputs)
	require.Len(t, node.Outputs.Parameters, 1)
	assert.Equal(t, "hello-param", node.Outputs.Parameters[0].Name)
	assert.Equal(t, "global-param", node.Outputs.Parameters[0].GlobalName)
	assert.Equal(t, "hello world", node.Outputs.Parameters[0].Value.String())

	assert.Equal(t, "hello world", woc.wf.Status.Outputs.Parameters[0].Value.String())
	assert.Equal(t, "global-param", woc.wf.Status.Outputs.Parameters[0].Name)
}

func TestResolvePlaceholdersInGlobalVariables(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/global-variable-placeholders.yaml")
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.NotEmpty(t, pods.Items, "pod was not created successfully")

	template, err := getPodTemplate(&pods.Items[0])
	require.NoError(t, err)
	namespaceValue := template.Outputs.Parameters[0].Value
	assert.NotNil(t, namespaceValue)
	assert.Equal(t, "testNamespace", namespaceValue.String())
	serviceAccountNameValue := template.Outputs.Parameters[1].Value
	assert.NotNil(t, serviceAccountNameValue)
	assert.Equal(t, "testServiceAccountName", serviceAccountNameValue.String())
}

func TestUnsuppliedArgValue(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/unsupplied-arg-value.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, woc.wf.Status.Conditions[0].Status, metav1.ConditionStatus("True"))
	assert.Equal(t, "invalid spec: spec.arguments.missing.value or spec.arguments.missing.valueFrom is required", woc.wf.Status.Message)
}

// TestSuppliedArgValue ensures that supplied workflow parameters are correctly set in the global parameters
func TestSuppliedArgValue(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/supplied-arg-value.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, "argo", woc.globalParams()["workflow.parameters.message"])
}

// This tests that retryStrategy.backoff.maxDuration works correctly even if the first child node was deleted without a
// proper finishedTime tag.
func TestMaxDurationOnErroredFirstNode(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/max-duration-on-errored-first-node.yaml")

	// Simulate node failed just now
	node := wf.Status.Nodes["echo-wngc4-1641470511"]
	node.StartedAt = metav1.Time{Time: time.Now().Add(-1 * time.Second)}
	wf.Status.Nodes["echo-wngc4-1641470511"] = node

	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

// This tests that we don't wait a backoff if it would exceed the maxDuration anyway.
func TestBackoffExceedsMaxDuration(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/backoff-exceeds-max-duration.yaml")

	// Simulate node failed just now
	node := wf.Status.Nodes["echo-r6v49-3721138751"]
	node.StartedAt = metav1.Time{Time: time.Now().Add(-1 * time.Second)}
	node.FinishedAt = metav1.Time{Time: time.Now()}
	wf.Status.Nodes["echo-r6v49-3721138751"] = node

	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Equal(t, "Backoff would exceed max duration limit", woc.wf.Status.Nodes["echo-r6v49"].Message)
	assert.Equal(t, "Backoff would exceed max duration limit", woc.wf.Status.Message)
}

// This tests that we don't wait a backoff if it would exceed the maxDuration anyway.
func TestNoOnExitWhenSkipped(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/no-on-exit-when-skipped.yaml")

	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	_, err := woc.wf.GetNodeByName("B.onExit")
	require.Error(t, err)
}

// This tests that we don't wait a backoff if it would exceed the maxDuration anyway.
func TestPanicMetric(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/no-on-exit-when-skipped.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)

	// This should make the call to "operate" panic
	woc.preExecutionNodeStatuses = nil
	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("cause", "OperationPanic"))
	val, err := testExporter.GetInt64CounterValue(ctx, "error_count", &attribs)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
}

// Assert Workflows cannot be run without using workflowTemplateRef in reference mode
func TestControllerReferenceMode(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/global-variable-placeholders.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{}
	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingStrict
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Equal(t, "workflows must use workflowTemplateRef to be executed when the controller is in reference mode", woc.wf.Status.Message)

	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingSecure
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Equal(t, "workflows must use workflowTemplateRef to be executed when the controller is in reference mode", woc.wf.Status.Message)

	controller.Config.WorkflowRestrictions = nil
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
}

func TestValidReferenceMode(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/workflow-template-ref.yaml")
	wfTmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/workflow-template-submittable.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf, wfTmpl)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{}
	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingSecure
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// Change stored Workflow Spec
	woc.wf.Status.StoredWorkflowSpec.Entrypoint = "different"
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Equal(t, "WorkflowSpec may not change during execution when the controller is set `templateReferencing: Secure`", woc.wf.Status.Message)

	controller.Config.WorkflowRestrictions.TemplateReferencing = config.TemplateReferencingStrict
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// Change stored Workflow Spec
	woc.wf.Status.StoredWorkflowSpec.Entrypoint = "different"
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
}

func TestReferenceModeBlocksDisallowedFields(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/workflow-template-ref.yaml")
	wfTmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/workflow-template-submittable.yaml")

	t.Run("Strict rejects podSpecPatch", func(t *testing.T) {
		wfWithPatch := wf.DeepCopy()
		wfWithPatch.Spec.PodSpecPatch = `{"containers":[{"name":"main","securityContext":{"privileged":true}}]}`
		cancel, controller := newController(logging.TestContext(t.Context()), wfWithPatch, wfTmpl)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{
			TemplateReferencing: config.TemplateReferencingStrict,
		}
		woc := newWorkflowOperationCtx(ctx, wfWithPatch, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
		assert.Contains(t, woc.wf.Status.Message, "PodSpecPatch")
		assert.Contains(t, woc.wf.Status.Message, "not permitted")
	})

	t.Run("Secure rejects podSpecPatch", func(t *testing.T) {
		wfWithPatch := wf.DeepCopy()
		wfWithPatch.Spec.PodSpecPatch = `{"containers":[{"name":"main","securityContext":{"runAsUser":0}}]}`
		cancel, controller := newController(logging.TestContext(t.Context()), wfWithPatch, wfTmpl)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{
			TemplateReferencing: config.TemplateReferencingSecure,
		}
		woc := newWorkflowOperationCtx(ctx, wfWithPatch, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
		assert.Contains(t, woc.wf.Status.Message, "PodSpecPatch")
		assert.Contains(t, woc.wf.Status.Message, "not permitted")
	})

	t.Run("Strict rejects ServiceAccountName", func(t *testing.T) {
		wfCopy := wf.DeepCopy()
		wfCopy.Spec.ServiceAccountName = "admin"
		cancel, controller := newController(logging.TestContext(t.Context()), wfCopy, wfTmpl)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{
			TemplateReferencing: config.TemplateReferencingStrict,
		}
		woc := newWorkflowOperationCtx(ctx, wfCopy, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
		assert.Contains(t, woc.wf.Status.Message, "ServiceAccountName")
	})

	t.Run("Strict rejects SecurityContext", func(t *testing.T) {
		wfCopy := wf.DeepCopy()
		wfCopy.Spec.SecurityContext = &apiv1.PodSecurityContext{}
		cancel, controller := newController(logging.TestContext(t.Context()), wfCopy, wfTmpl)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{
			TemplateReferencing: config.TemplateReferencingStrict,
		}
		woc := newWorkflowOperationCtx(ctx, wfCopy, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
		assert.Contains(t, woc.wf.Status.Message, "SecurityContext")
	})

	t.Run("Strict rejects Templates", func(t *testing.T) {
		wfCopy := wf.DeepCopy()
		wfCopy.Spec.Templates = []wfv1.Template{{Name: "injected"}}
		cancel, controller := newController(logging.TestContext(t.Context()), wfCopy, wfTmpl)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{
			TemplateReferencing: config.TemplateReferencingStrict,
		}
		woc := newWorkflowOperationCtx(ctx, wfCopy, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
		assert.Contains(t, woc.wf.Status.Message, "Templates")
	})

	t.Run("Strict rejects Volumes", func(t *testing.T) {
		wfCopy := wf.DeepCopy()
		wfCopy.Spec.Volumes = []apiv1.Volume{{Name: "secret-vol"}}
		cancel, controller := newController(logging.TestContext(t.Context()), wfCopy, wfTmpl)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{
			TemplateReferencing: config.TemplateReferencingStrict,
		}
		woc := newWorkflowOperationCtx(ctx, wfCopy, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
		assert.Contains(t, woc.wf.Status.Message, "Volumes")
	})

	t.Run("Strict rejects HostNetwork", func(t *testing.T) {
		wfCopy := wf.DeepCopy()
		hostNet := true
		wfCopy.Spec.HostNetwork = &hostNet
		cancel, controller := newController(logging.TestContext(t.Context()), wfCopy, wfTmpl)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{
			TemplateReferencing: config.TemplateReferencingStrict,
		}
		woc := newWorkflowOperationCtx(ctx, wfCopy, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
		assert.Contains(t, woc.wf.Status.Message, "HostNetwork")
	})

	t.Run("No restrictions allows podSpecPatch", func(t *testing.T) {
		wfWithPatch := wf.DeepCopy()
		wfWithPatch.Spec.PodSpecPatch = `{"containers":[{"name":"main","resources":{"limits":{"cpu":"1"}}}]}`
		cancel, controller := newController(logging.TestContext(t.Context()), wfWithPatch, wfTmpl)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		controller.Config.WorkflowRestrictions = nil
		woc := newWorkflowOperationCtx(ctx, wfWithPatch, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	})

	t.Run("Allowed fields pass in Strict mode", func(t *testing.T) {
		wfCopy := wf.DeepCopy()
		// Set allowed fields only - entrypoint must match one defined in the template
		wfCopy.Spec.Entrypoint = "whalesay-template"
		wfCopy.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
		cancel, controller := newController(logging.TestContext(t.Context()), wfCopy, wfTmpl)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{
			TemplateReferencing: config.TemplateReferencingStrict,
		}
		woc := newWorkflowOperationCtx(ctx, wfCopy, controller)
		woc.operate(ctx)
		// Shutdown=Terminate means it won't reach Running, but it shouldn't be Error
		assert.NotEqual(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	})

	t.Run("Without disallowed fields still works in Strict mode", func(t *testing.T) {
		cancel, controller := newController(logging.TestContext(t.Context()), wf, wfTmpl)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		controller.Config.WorkflowRestrictions = &config.WorkflowRestrictions{
			TemplateReferencing: config.TemplateReferencingStrict,
		}
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	})
}

func TestWorkflowStatusMetric(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-status-metric.yaml")
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	// Must only be two (completed: true), (podRunning: true)
	assert.Len(t, woc.wf.Status.Conditions, 2)
}

func TestWorkflowConditions(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-conditions.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Nil(t, woc.wf.Status.Conditions, "zero conditions on first reconciliation")
	makePodsPhase(ctx, woc, apiv1.PodPending)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Conditions{{Type: wfv1.ConditionTypePodRunning, Status: metav1.ConditionFalse}}, woc.wf.Status.Conditions)

	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Conditions{{Type: wfv1.ConditionTypePodRunning, Status: metav1.ConditionTrue}}, woc.wf.Status.Conditions)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.Conditions{
		{Type: wfv1.ConditionTypePodRunning, Status: metav1.ConditionFalse},
		{Type: wfv1.ConditionTypeCompleted, Status: metav1.ConditionTrue},
	}, woc.wf.Status.Conditions)
}

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
			Labels: map[string]string{
				common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache,
			},
		},
	}
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-cached.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapCacheEntry, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	require.Len(t, woc.wf.Status.Nodes, 1)
	for _, node := range woc.wf.Status.Nodes {
		assert.NotNil(t, node.Outputs)
		assert.Equal(t, "hello", node.Outputs.Parameters[0].Name)
		assert.Equal(t, "foobar", node.Outputs.Parameters[0].Value.String())
		assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	}
}

func TestConfigMapCacheLoadOperateNoOutputs(t *testing.T) {
	sampleConfigMapCacheEntry := apiv1.ConfigMap{
		Data: map[string]string{
			"hi-there-world": `{"nodeID":"memoized-simple-workflow-5wj2p","outputs":null,"creationTimestamp":"2020-09-21T18:12:56Z"}`,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "whalesay-cache",
			ResourceVersion: "1630732",
			Labels: map[string]string{
				common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache,
			},
		},
	}
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-cached-no-outputs.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapCacheEntry, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	require.Len(t, woc.wf.Status.Nodes, 1)
	for _, node := range woc.wf.Status.Nodes {
		assert.Nil(t, node.Outputs)
		assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	}
}

func TestGetOutboundNodesFromCacheHitSteps(t *testing.T) {
	myConfigMapCacheEntry := apiv1.ConfigMap{
		Data: map[string]string{
			"memoized-bug-steps-0": `{"nodeID":"memoized-bug-wqbj4-3475368823","outputs":null,"creationTimestamp":"2020-09-21T18:12:56Z","lastHitTimestamp":"2024-03-11T05:59:58Z"}`,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-config",
			ResourceVersion: "80004",
			Labels: map[string]string{
				common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache,
			},
		},
	}

	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-with-memoized-in-steps.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &myConfigMapCacheEntry, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)

	hitCache := 0
	for _, node := range woc.wf.Status.Nodes {
		if node.DisplayName == "hello-steps" {
			hitCache++
			assert.NotNil(t, node.MemoizationStatus)
			assert.True(t, node.MemoizationStatus.Hit)
			assert.Len(t, node.Children, 1)
		}
	}
	assert.Equal(t, 1, hitCache)
}

func TestGetOutboundNodesFromCacheHitDAG(t *testing.T) {
	myConfigMapCacheEntry := apiv1.ConfigMap{
		Data: map[string]string{
			"memoized-bug-dag-0": `{"nodeID":"memoized-bug-wqbj4-3475368823","outputs":null,"creationTimestamp":"2020-09-21T18:12:56Z","lastHitTimestamp":"2024-03-11T05:59:58Z"}`,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "my-config",
			ResourceVersion: "80004",
			Labels: map[string]string{
				common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache,
			},
		},
	}

	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-with-memoized-in-dag.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &myConfigMapCacheEntry, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)

	hitCache := 0
	for _, node := range woc.wf.Status.Nodes {
		if node.DisplayName == "hello-dag" {
			hitCache++
			assert.NotNil(t, node.MemoizationStatus)
			assert.True(t, node.MemoizationStatus.Hit)
			assert.Len(t, node.Children, 1)
		}
	}
	assert.Equal(t, 1, hitCache)
}

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
				Labels: map[string]string{
					common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache,
				},
			},
		}
	}
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-cached-max-age.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()))

	ctx := logging.TestContext(t.Context())
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	nonExpiredEntry := getEntryCreatedAtTime(time.Now().Add(-5 * time.Second))
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &nonExpiredEntry, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	require.Len(t, woc.wf.Status.Nodes, 1)
	for _, node := range woc.wf.Status.Nodes {
		assert.NotNil(t, node.Outputs)
		assert.Equal(t, "hello", node.Outputs.Parameters[0].Name)
		assert.Equal(t, "foobar", node.Outputs.Parameters[0].Value.String())
		assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	}

	cancel()
	cancel, controller = newController(logging.TestContext(t.Context()))
	defer cancel()

	expiredEntry := getEntryCreatedAtTime(time.Now().Add(-15 * time.Second))
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &expiredEntry, metav1.CreateOptions{})
	require.NoError(t, err)

	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	require.Len(t, woc.wf.Status.Nodes, 1)
	for _, node := range woc.wf.Status.Nodes {
		assert.Nil(t, node.Outputs)
		assert.Equal(t, wfv1.NodePending, node.Phase)
	}
}

func TestStepConfigMapCacheCreateWhenHaveRetryStrategy(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-step-cached-with-retry-strategy.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	cm, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "whalesay-cache", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Contains(t, cm.Labels, common.LabelKeyConfigMapType)
	assert.Equal(t, common.LabelValueTypeConfigMapCache, cm.Labels[common.LabelKeyConfigMapType])
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestDAGConfigMapCacheCreateWhenHaveRetryStrategy(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-dag-cached-with-retry-strategy.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	cm, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "memoization-test-cache", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Contains(t, cm.Labels, common.LabelKeyConfigMapType)
	assert.Equal(t, common.LabelValueTypeConfigMapCache, cm.Labels[common.LabelKeyConfigMapType])
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestConfigMapCacheLoadNoLabels(t *testing.T) {
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
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-cached.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapCacheEntry, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	fn := func() {
		woc.operate(ctx)
	}
	assert.NotPanics(t, fn)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)

	require.Len(t, woc.wf.Status.Nodes, 1)
	for _, node := range woc.wf.Status.Nodes {
		assert.Nil(t, node.Outputs)
		assert.Equal(t, wfv1.NodeError, node.Phase)
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
			Labels: map[string]string{
				common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache,
			},
		},
	}
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-cached.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()

	ctx := logging.TestContext(t.Context())
	_, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.ObjectMeta.Namespace).Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	_, err = controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapCacheEntry, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	fn := func() {
		woc.operate(ctx)
	}
	assert.NotPanics(t, fn)

	require.Len(t, woc.wf.Status.Nodes, 1)
	for _, node := range woc.wf.Status.Nodes {
		assert.NotNil(t, node.Outputs)
		assert.False(t, node.Outputs.HasOutputs())
		assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	}
}

func TestConfigMapCacheSaveOperate(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-cached.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	sampleOutputs := wfv1.Outputs{
		Parameters: []wfv1.Parameter{
			{Name: "hello", Value: wfv1.AnyStringPtr("foobar")},
		},
		ExitCode: new("0"),
	}

	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded, withExitCode(0), withOutputs(ctx, sampleOutputs))
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	cm, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "whalesay-cache", metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotNil(t, cm)
	assert.NotNil(t, cm.Data)
	rawEntry, ok := cm.Data["hi-there-world"]
	assert.True(t, ok)
	var entry cache.Entry
	wfv1.MustUnmarshal(rawEntry, &entry)

	require.NotNil(t, entry.Outputs)
	assert.Equal(t, sampleOutputs, *entry.Outputs)
}

func TestPropagateMaxDurationProcess(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	assert.NotNil(t, controller)
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/propagate.yaml")
	assert.NotNil(t, wf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotNil(t, woc)
	_, err := woc.setExecWorkflow(ctx)
	require.NoError(t, err)
	assert.Empty(t, woc.wf.Status.Nodes)

	// Add the parent node for retries.
	nodeName := "test-node"
	_, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	retries := wfv1.RetryStrategy{
		Limit: intstrutil.ParsePtr("2"),
		Backoff: &wfv1.Backoff{
			Duration:    "0",
			Factor:      intstrutil.ParsePtr("1"),
			MaxDuration: "20",
		},
	}
	woc.wf.Status.Nodes[woc.wf.NodeID(nodeName)] = *node

	childNode := fmt.Sprintf("%s(%d)", nodeName, 0)
	woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeFailed, &wfv1.NodeFlag{Retried: true}, true)
	woc.addChildNode(ctx, nodeName, childNode)

	var opts executeTemplateOpts
	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	_, _, err = woc.processNodeRetries(ctx, n, retries, &opts)
	require.NoError(t, err)
	assert.Equal(t, n.StartedAt.Add(20*time.Second).Round(time.Second).String(), opts.executionDeadline.Round(time.Second).String())
}

func TestCheckForbiddenErrorAndResbmitAllowed(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/resubmit-pending-wf.yaml")
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	forbiddenErr := apierr.NewForbidden(schema.GroupResource{Group: "test", Resource: "test1"}, "test", errors.New("exceeded quota"))
	nonForbiddenErr := apierr.NewBadRequest("badrequest")
	t.Run("ForbiddenError", func(t *testing.T) {
		node, err := woc.requeueIfTransientErr(ctx, forbiddenErr, "resubmit-pending-wf")
		assert.NotNil(t, node)
		require.NoError(t, err)
		assert.Equal(t, wfv1.NodePending, node.Phase)
	})
	t.Run("NonForbiddenError", func(t *testing.T) {
		node, err := woc.requeueIfTransientErr(ctx, nonForbiddenErr, "resubmit-pending-wf")
		require.Error(t, err)
		assert.Nil(t, node)
	})
}

func TestResubmitMemoization(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/resubmit-memoization.yaml")
	ctx := logging.TestContext(t.Context())
	wf, err := util.FormulateResubmitWorkflow(ctx, wf, true, nil)
	require.NoError(t, err)
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	for _, node := range woc.wf.Status.Nodes {
		switch node.TemplateName {
		case "main":
			assert.Equal(t, wfv1.NodePending, node.Phase)
			assert.False(t, node.StartTime().IsZero())
			assert.Equal(t, "my-wf", woc.wf.Labels[common.LabelKeyPreviousWorkflowName])
		case "":
		default:
			assert.Fail(t, "invalid template")
		}
	}
	list, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, list.Items, 1)
}

func TestResubmitParamsOverride(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/resubmit-params-override.yaml")
	ctx := logging.TestContext(t.Context())
	wf, err := util.FormulateResubmitWorkflow(ctx, wf, true, []string{"message=modified"})
	require.NoError(t, err)
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, "modified", wf.Spec.Arguments.Parameters[0].Value.String())
}

func TestRetryParamsOverride(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/retry-params-override.yaml")

	ctx := logging.TestContext(t.Context())
	wf, _, err := util.FormulateRetryWorkflow(ctx, wf, false, "", []string{"message=modified"})
	require.NoError(t, err)
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, "modified", wf.Spec.Arguments.Parameters[0].Value.String())
}

func TestWorkflowOutputs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-outputs.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// make all created pods as successful
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	// reconcile
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestGlobalVarsOnExit(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/global-vars-on-exit.yaml")
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator/wftmpl-global-vars-on-exit.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf, wftmpl)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	node := woc.wf.Status.Nodes["hello-world-6gphm-8n22g-3224262006"]
	require.NotNil(t, node)
	require.NotNil(t, node.Inputs)
	require.NotEmpty(t, node.Inputs.Parameters)
	assert.Equal(t, "nononono", node.Inputs.Parameters[0].Value.String())
}

func TestFailSuspendedAndPendingNodesAfterDeadline(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/deadline-wf.yaml")
	wf.Status.StartedAt = metav1.Now()
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	t.Run("Before Deadline", func(t *testing.T) {
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	})
	time.Sleep(3 * time.Second)
	t.Run("After Deadline", func(t *testing.T) {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
		}
	})
}

func TestFailSuspendedAndPendingNodesAfterShutdown(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/deadline-wf.yaml")
	wf.Spec.Shutdown = wfv1.ShutdownStrategyStop
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	t.Run("After Shutdown", func(t *testing.T) {
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodeFailed, node.Phase)
		}
	})
}

var pendingTimeoutWf = `
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
          - name: sleep_time
            value: 3s
  - name: whalesay
    inputs:
      parameters:
      - name: sleep_time
    pendingTimeout: 2s
    timeout: 4s
    container:
      image: debian:9.5-slim
      command: [sleep, "{{inputs.parameters.sleep_time}}"]
`

func TestTemplateTimeoutDuration(t *testing.T) {
	t.Run("Step Template Deadline", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/step-timeout-wf.yaml")
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		time.Sleep(6 * time.Second)
		makePodsPhase(ctx, woc, apiv1.PodPending)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Nodes.FindByDisplayName("step1").Phase)
	})
	t.Run("DAG Template Deadline", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/dag-timeout-wf.yaml")
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		time.Sleep(6 * time.Second)
		makePodsPhase(ctx, woc, apiv1.PodPending)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)

		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Nodes.FindByDisplayName("hello-world-dag").Phase)
	})
	t.Run("Invalid timeout format", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/step-timeout-wf.yaml")
		tmpl := wf.Spec.Templates[1]
		tmpl.Timeout = "23"
		wf.Spec.Templates[1] = tmpl
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		jsonByte, err := json.Marshal(woc.wf)
		require.NoError(t, err)
		assert.Contains(t, string(jsonByte), "has invalid duration format in timeout")
	})

	t.Run("Invalid timeout in step", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/step-timeout-wf.yaml")
		tmpl := wf.Spec.Templates[0]
		tmpl.Timeout = "23"
		wf.Spec.Templates[0] = tmpl
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		jsonByte, err := json.Marshal(woc.wf)
		require.NoError(t, err)
		assert.Contains(t, string(jsonByte), "doesn't support timeout field")
	})
	t.Run("PendingTimeout in step", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/step-timeout-wf.yaml")
		tmpl := wf.Spec.Templates[0]
		tmpl.PendingTimeout = "23s"
		wf.Spec.Templates[0] = tmpl
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		jsonByte, err := json.Marshal(woc.wf)
		require.NoError(t, err)
		assert.Contains(t, string(jsonByte), "doesn't support pendingTimeout field")
	})
	t.Run("PendingTimeout", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(pendingTimeoutWf)
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
		makePodsPhase(ctx, woc, apiv1.PodPending)
		time.Sleep(6 * time.Second)
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)

		assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
		assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Nodes.FindByDisplayName("hello-world-dag").Phase)

		// the timed-out pending pod is queued for deletion
		assert.True(t, controller.PodController.TestingProcessNextItem(ctx))
		pods, err := listPods(ctx, woc)
		require.NoError(t, err)
		assert.Empty(t, pods.Items)
	})
}

func TestStorageQuota(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wf-with-pvc.yaml")

	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	controller.kubeclientset.(*fake.Clientset).BatchV1().(*batchfake.FakeBatchV1).PrependReactor("create", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierr.NewForbidden(schema.GroupResource{Group: "test", Resource: "test1"}, "test", errors.New("exceeded quota"))
	})

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowPending, woc.wf.Status.Phase)
	assert.Contains(t, woc.wf.Status.Message, "Waiting for a PVC to be created.")

	controller.kubeclientset.(*fake.Clientset).BatchV1().(*batchfake.FakeBatchV1).PrependReactor("create", "persistentvolumeclaims", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierr.NewBadRequest("BadRequest")
	})

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.Contains(t, woc.wf.Status.Message, "BadRequest")
}

func TestPodFailureWithContainerWaitingState(t *testing.T) {
	var pod apiv1.Pod
	wfv1.MustUnmarshal("@testdata/operator/pod-with-failed.yaml", &pod)
	assert.NotNil(t, pod)
	ctx := logging.TestContext(t.Context())
	nodeStatus, msg := newWoc(ctx).inferFailedReason(ctx, &pod, nil)
	assert.Equal(t, wfv1.NodeError, nodeStatus)
	assert.Equal(t, "Pod failed before main container starts due to ContainerCreating: Container is creating", msg)
}

// Test that when containers are in Running state (not Terminated) on a Failed pod,
// we correctly return Failed instead of incorrectly returning Succeeded.
// This can happen during node eviction or other pod-level failures.
func TestPodFailureWithRunningContainers(t *testing.T) {
	var pod apiv1.Pod
	wfv1.MustUnmarshal("@testdata/operator/pod-failed-with-running-containers.yaml", &pod)
	assert.NotNil(t, pod)
	ctx := logging.TestContext(t.Context())
	// Pod has a message, so it should return that
	nodeStatus, msg := newWoc(ctx).inferFailedReason(ctx, &pod, nil)
	assert.Equal(t, wfv1.NodeFailed, nodeStatus)
	assert.Equal(t, "The node was low on resource: memory.", msg)
}

// Test that when containers don't have terminated state and there's no pod message,
// we correctly return Failed with information about which containers couldn't be confirmed.
func TestPodFailureWithRunningContainersNoMessage(t *testing.T) {
	var pod apiv1.Pod
	wfv1.MustUnmarshal("@testdata/operator/pod-failed-with-running-containers-no-message.yaml", &pod)
	assert.NotNil(t, pod)
	ctx := logging.TestContext(t.Context())
	nodeStatus, msg := newWoc(ctx).inferFailedReason(ctx, &pod, nil)
	assert.Equal(t, wfv1.NodeFailed, nodeStatus)
	assert.Equal(t, "pod failed: neither main nor wait container completed successfully", msg)
}

// Test that sidecar SIGKILL still results in success when main and wait succeeded
func TestPodFailureWithSidecarSigkill(t *testing.T) {
	var pod apiv1.Pod
	wfv1.MustUnmarshal("@testdata/operator/pod-failed-with-sidecar-sigkill.yaml", &pod)
	assert.NotNil(t, pod)
	ctx := logging.TestContext(t.Context())
	nodeStatus, msg := newWoc(ctx).inferFailedReason(ctx, &pod, nil)
	// Main and wait succeeded, sidecar was SIGKILL'd - this should be success
	assert.Equal(t, wfv1.NodeSucceeded, nodeStatus)
	assert.Empty(t, msg)
}

func TestPodFailureWithContainerOOM(t *testing.T) {
	tests := []struct {
		podDetail string
		phase     wfv1.NodePhase
	}{{
		podDetail: "@testdata/operator/pod-with-wait-container-oom.yaml",
		phase:     wfv1.NodeError,
	}, {
		podDetail: "@testdata/operator/pod-with-main-container-oom.yaml",
		phase:     wfv1.NodeFailed,
	}}
	var pod apiv1.Pod
	for _, tt := range tests {
		wfv1.MustUnmarshal(tt.podDetail, &pod)
		assert.NotNil(t, pod)
		ctx := logging.TestContext(t.Context())
		nodeStatus, msg := newWoc(ctx).inferFailedReason(ctx, &pod, nil)
		assert.Equal(t, tt.phase, nodeStatus)
		assert.Contains(t, msg, "OOMKilled")
	}
}

// TestInferFailedReason_SupervisorPreMainFailure exercises the init-less mode failure path
// where supervisor pre-main fails: supervisor writes the failed marker and
// exits non-zero, emissary in main observes the marker and exits with 65
// (sysexits.h EX_DATAERR). inferFailedReason must surface the supervisor's
// real message rather than main's placeholder exit-65; this is why
// SupervisorContainerName sorts ahead of main in the order() helper.
func TestInferFailedReason_SupervisorPreMainFailure(t *testing.T) {
	var pod apiv1.Pod
	wfv1.MustUnmarshal("@testdata/operator/pod-with-supervisor-pre-main-failure.yaml", &pod)
	ctx := logging.TestContext(t.Context())
	nodeStatus, msg := newWoc(ctx).inferFailedReason(ctx, &pod, nil)
	// Supervisor's NodeError must win over main's exit-65 NodeFailed; the
	// emitted message must name the supervisor and quote its real reason.
	assert.Equal(t, wfv1.NodeError, nodeStatus)
	assert.Contains(t, msg, "supervisor")
	assert.Contains(t, msg, "connection refused")
	// Precedence, not a merge: main's placeholder exit-65 must not leak through.
	assert.NotContains(t, msg, "exit code 65")
}

// When the supervisor exits cleanly (post-main success) but main fails with
// a real user error, main's failure must still surface — supervisor's order=0
// ranking only kicks in when supervisor itself failed.
func TestInferFailedReason_SupervisorSuccessMainFailure(t *testing.T) {
	var pod apiv1.Pod
	wfv1.MustUnmarshal("@testdata/operator/pod-with-supervisor-ok-and-main-fail.yaml", &pod)
	ctx := logging.TestContext(t.Context())
	nodeStatus, msg := newWoc(ctx).inferFailedReason(ctx, &pod, nil)
	assert.Equal(t, wfv1.NodeFailed, nodeStatus)
	assert.Contains(t, msg, "main")
	assert.Contains(t, msg, "user command failed")
	// A clean (post-main) supervisor must not appear in the user-failure message.
	assert.NotContains(t, msg, "supervisor")
}

func TestResubmitPendingPods(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/resubmit-pending-pods.yaml")
	wftmpl := wfv1.MustUnmarshalWorkflowTemplate("@testdata/operator/wftmpl-global-vars-on-exit.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf, wftmpl)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.True(t, woc.wf.Status.Nodes.Any(func(node wfv1.NodeStatus) bool {
		return node.Phase == wfv1.NodePending
	}))

	time.Sleep(time.Second)
	deletePods(ctx, woc)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.True(t, woc.wf.Status.Nodes.Any(func(node wfv1.NodeStatus) bool {
		return node.Phase == wfv1.NodePending
	}))

	time.Sleep(time.Second)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestWFWithRetryAndWithParam(t *testing.T) {
	t.Run("IncludeScriptOutputInRetryAndWithParam", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wf-retry-with-param.yaml")
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()

		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		pods, err := listPods(ctx, woc)
		require.NoError(t, err)
		assert.NotEmpty(t, pods.Items)
		require.Len(t, pods.Items, 3)
		ctrs := pods.Items[0].Spec.Containers
		assert.Len(t, ctrs, 2)
		envs := ctrs[1].Env
		assert.Len(t, envs, 10)
		assert.Equal(t, apiv1.EnvVar{Name: "ARGO_INCLUDE_SCRIPT_OUTPUT", Value: "true"}, envs[2])
	})
}

func TestParamAggregation(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/param-aggregation.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	evenNode := woc.wf.Status.Nodes.FindByDisplayName("print-evenness")
	require.NotNil(t, evenNode)
	require.Len(t, evenNode.Inputs.Parameters, 1)
	assert.Equal(t, `["odd","even"]`, evenNode.Inputs.Parameters[0].Value.String())

	numNode := woc.wf.Status.Nodes.FindByDisplayName("print-nums")
	require.NotNil(t, numNode)
	require.Len(t, numNode.Inputs.Parameters, 1)
	assert.Equal(t, `["1","2"]`, numNode.Inputs.Parameters[0].Value.String())
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
			},
		},
	}
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
			},
		},
	}
	assert.True(t, podHasContainerNeedingTermination(&pod, tmpl))

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
			},
		},
	}
	assert.False(t, podHasContainerNeedingTermination(&pod, tmpl))

	pod = apiv1.Pod{
		Status: apiv1.PodStatus{
			ContainerStatuses: []apiv1.ContainerStatus{
				{
					Name:  common.MainContainerName,
					State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
				},
			},
		},
	}
	assert.False(t, podHasContainerNeedingTermination(&pod, tmpl))

	pod = apiv1.Pod{
		Status: apiv1.PodStatus{
			ContainerStatuses: []apiv1.ContainerStatus{
				{
					Name:  common.MainContainerName,
					State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}},
				},
			},
		},
	}
	assert.True(t, podHasContainerNeedingTermination(&pod, tmpl))

	// containerSet: a failed container must not cause a still-running
	// sibling to be terminated early.
	containerSetTmpl := wfv1.Template{
		ContainerSet: &wfv1.ContainerSetTemplate{
			Containers: []wfv1.ContainerNode{
				{Container: apiv1.Container{Name: "c1"}},
				{Container: apiv1.Container{Name: "c2"}},
			},
		},
	}
	pod = apiv1.Pod{
		Status: apiv1.PodStatus{
			ContainerStatuses: []apiv1.ContainerStatus{
				{
					Name:  "c1",
					State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}},
				},
				{
					Name:  "c2",
					State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}},
				},
			},
		},
	}
	assert.False(t, podHasContainerNeedingTermination(&pod, containerSetTmpl))

	// containerSet: once every sibling has exited (regardless of code) the
	// pod must be terminated so the wait container finishes.
	pod = apiv1.Pod{
		Status: apiv1.PodStatus{
			ContainerStatuses: []apiv1.ContainerStatus{
				{
					Name:  "c1",
					State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}},
				},
				{
					Name:  "c2",
					State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 0}},
				},
			},
		},
	}
	assert.True(t, podHasContainerNeedingTermination(&pod, containerSetTmpl))

	// containerSet with a retryStrategy: a failed container must not tear down the
	// pod while a sibling is still Running (retrying inside the emissary).
	csTmpl := wfv1.Template{
		ContainerSet: &wfv1.ContainerSetTemplate{
			RetryStrategy: &wfv1.ContainerSetRetryStrategy{},
			Containers: []wfv1.ContainerNode{
				{Container: apiv1.Container{Name: "c2"}},
				{Container: apiv1.Container{Name: "c3"}},
			},
		},
	}
	pod = apiv1.Pod{
		Status: apiv1.PodStatus{
			ContainerStatuses: []apiv1.ContainerStatus{
				{Name: "c2", State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 64}}},
				{Name: "c3", State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}},
			},
		},
	}
	assert.False(t, podHasContainerNeedingTermination(&pod, csTmpl))

	// once every container has exited, the containerSet pod is terminated.
	pod = apiv1.Pod{
		Status: apiv1.PodStatus{
			ContainerStatuses: []apiv1.ContainerStatus{
				{Name: "c2", State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 64}}},
				{Name: "c3", State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}}},
			},
		},
	}
	assert.True(t, podHasContainerNeedingTermination(&pod, csTmpl))
}

func TestRetryOnDiffHost(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	// Verify that there are no nodes in the wf status.
	assert.Empty(t, woc.wf.Status.Nodes)

	// Add the parent node for retries.
	nodeName := "test-node"

	nodeID := woc.wf.NodeID(nodeName)
	_, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)

	hostSelector := "kubernetes.io/hostname"
	retries := wfv1.RetryStrategy{}
	retries.Affinity = &wfv1.RetryAffinity{
		NodeAntiAffinity: &wfv1.RetryNodeAntiAffinity{},
	}

	woc.wf.Status.Nodes[nodeID] = *node

	assert.Equal(t, wfv1.NodeRunning, node.Phase)

	// Ensure there are no child nodes yet.
	lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
	assert.Nil(t, lastChild)

	// Add child node.
	childNode := fmt.Sprintf("%s(%d)", nodeName, 0)
	woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
	woc.addChildNode(ctx, nodeName, childNode)

	n, err := woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	lastChild = getChildNodeIndex(n, woc.wf.Status.Nodes, -1)
	assert.NotNil(t, lastChild)

	woc.markNodePhase(ctx, lastChild.Name, wfv1.NodeFailed)
	_, _, err = woc.processNodeRetries(ctx, n, retries, &executeTemplateOpts{})
	require.NoError(t, err)
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, n.Phase)

	// Ensure related fields are not set
	assert.Empty(t, lastChild.HostNodeName)

	// Set host name
	n, err = woc.wf.GetNodeByName(nodeName)
	require.NoError(t, err)
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
	targetNodeSelectorRequirement := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
	sourceNodeSelectorRequirement := apiv1.NodeSelectorRequirement{
		Key:      hostSelector,
		Operator: apiv1.NodeSelectorOpNotIn,
		Values:   []string{lastChild.HostNodeName},
	}
	assert.Equal(t, sourceNodeSelectorRequirement, targetNodeSelectorRequirement)
}

func TestRetryOnNodeAntiAffinity(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/node-anti-affinity-workflow.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)

	// First retry
	pod := pods.Items[0]
	pod.Spec.NodeName = "node0"
	_, err = controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace()).Update(ctx, &pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.FindByDisplayName("retry-fail(0)")
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
	assert.Equal(t, "node0", node.HostNodeName)

	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 2)

	var podRetry1 apiv1.Pod
	for _, p := range pods.Items {
		if p.Name != pod.GetName() {
			podRetry1 = p
		}
	}

	hostSelector := "kubernetes.io/hostname"
	targetNodeSelectorRequirement := podRetry1.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
	sourceNodeSelectorRequirement := apiv1.NodeSelectorRequirement{
		Key:      hostSelector,
		Operator: apiv1.NodeSelectorOpNotIn,
		Values:   []string{node.HostNodeName},
	}
	assert.Equal(t, sourceNodeSelectorRequirement, targetNodeSelectorRequirement)

	// Second retry
	podRetry1.Spec.NodeName = "node1"
	_, err = controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace()).Update(ctx, &podRetry1, metav1.UpdateOptions{})
	require.NoError(t, err)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc.operate(ctx)

	node1 := woc.wf.Status.Nodes.FindByDisplayName("retry-fail(1)")
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodeFailed, node1.Phase)
	assert.Equal(t, "node1", node1.HostNodeName)

	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 3)

	var podRetry2 apiv1.Pod
	for _, p := range pods.Items {
		if p.Name != pod.GetName() && p.Name != podRetry1.GetName() {
			podRetry2 = p
		}
	}

	targetNodeSelectorRequirement = podRetry2.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
	sourceNodeSelectorRequirement = apiv1.NodeSelectorRequirement{
		Key:      hostSelector,
		Operator: apiv1.NodeSelectorOpNotIn,
		Values:   []string{node1.HostNodeName, node.HostNodeName},
	}
	assert.Equal(t, sourceNodeSelectorRequirement, targetNodeSelectorRequirement)
}

func TestRetryOnNodeAntiAffinitySteps(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/node-anti-affinity-steps-workflow.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)

	pod := pods.Items[0]
	pod.Spec.NodeName = "node0"
	_, err = controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace()).Update(ctx, &pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.FindByDisplayName("fail(0)")
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
	assert.Equal(t, "node0", node.HostNodeName)

	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 2)

	var podRetry1 apiv1.Pod
	for _, p := range pods.Items {
		if p.Name != pod.GetName() {
			podRetry1 = p
		}
	}

	hostSelector := "kubernetes.io/hostname"
	targetNodeSelectorRequirement := podRetry1.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0]
	sourceNodeSelectorRequirement := apiv1.NodeSelectorRequirement{
		Key:      hostSelector,
		Operator: apiv1.NodeSelectorOpNotIn,
		Values:   []string{node.HostNodeName},
	}
	assert.Equal(t, sourceNodeSelectorRequirement, targetNodeSelectorRequirement)
}

func TestNoPodsWhenShutdown(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/no-pods-when-shutdown.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.FindByDisplayName("hello-world")
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
	assert.Contains(t, node.Message, "workflow shutdown with strategy: Stop")
}

func TestWorkflowScheduledTimeVariable(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wfsche-variable.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, "2006-01-02T15:04:05-07:00", woc.globalParams()[varkeys.WorkflowScheduledTime.Template()])
}

func TestWorkflowMainEntrypointVariable(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wf-main-entrypoint-variable.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, "whalesay", woc.globalParams()[varkeys.WorkflowMainEntrypoint.Template()])
}

func TestWorkflowInterpolatesNodeNameField(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wf-node-name-field.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	foundPod := false
	for _, element := range woc.wf.Status.Nodes {
		if element.Type == "Pod" {
			foundPod = true
			assert.Equal(t, "hello-world.this-is-part-1", element.Inputs.Parameters[0].Value.String())
		}
	}

	assert.True(t, foundPod)
}

func TestWorkflowShutdownStrategy(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-shutdown-strategy.yaml")

	wf1 := wf.DeepCopy()
	wf1.Name = "whalesay-1"
	cancel, controller := newController(logging.TestContext(t.Context()), wf, wf1)
	defer cancel()
	t.Run("StopStrategy", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		// Simulate the Stop command
		wfstop := woc.wf
		wfstop.Spec.Shutdown = wfv1.ShutdownStrategyStop
		woc1 := newWorkflowOperationCtx(ctx, wfstop, controller)
		woc1.operate(ctx)

		node := woc1.wf.Status.Nodes.FindByDisplayName("whalesay")
		require.NotNil(t, node)
		assert.Contains(t, node.Message, "workflow shutdown with strategy")
		assert.Contains(t, node.Message, "Stop")
	})

	t.Run("TerminateStrategy", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf1, controller)
		woc.operate(ctx)

		for _, node := range woc.wf.Status.Nodes {
			assert.Equal(t, wfv1.NodePending, node.Phase)
		}
		// Updating Pod state
		makePodsPhase(ctx, woc, apiv1.PodPending)
		// Simulate the Terminate command
		wfOut := woc.wf
		wfOut.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
		woc1 := newWorkflowOperationCtx(ctx, wfOut, controller)
		woc1.operate(ctx)
		for _, node := range woc1.wf.Status.Nodes {
			require.NotNil(t, node)
			assert.Contains(t, node.Message, "workflow shutdown with strategy")
			assert.Contains(t, node.Message, "Terminate")
		}
	})
}

func TestHasOutputResultRef(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/result-var-ref-wf.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.True(t, woc.hasOutputResultRef(ctx, "generate-random", &wf.Spec.Templates[0]))
	assert.True(t, woc.hasOutputResultRef(ctx, "generate-random-1", &wf.Spec.Templates[0]))
}

func TestStepsFailFast(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/steps-fail-fast.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	node := woc.wf.Status.Nodes.FindByDisplayName("iteration(0:a)")
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)

	node = woc.wf.Status.Nodes.FindByDisplayName("seq-loop-pz4hh")
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
}

// TestDAGFailFastItems: failFast with an expanded task (withItems)
// and parallelism ends the workflow Failed while one item is still running,
// leaving the TaskGroup node between the DAG task and its items Running
// forever: the failFast branch never assesses it. This fails at base too.
func TestDAGFailFastItems(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/dag-fail-fast-items.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	first := true
	setPodPhases(ctx, woc, func(*wfv1.NodeStatus) apiv1.PodPhase {
		if first {
			first = false
			return apiv1.PodFailed
		}
		return apiv1.PodRunning
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	setPodPhases(ctx, woc, func(n *wfv1.NodeStatus) apiv1.PodPhase {
		if n.Fulfilled() {
			return ""
		}
		return apiv1.PodSucceeded
	})
	for range 3 {
		woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
		woc.operate(ctx)
	}
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	for _, n := range woc.wf.Status.Nodes {
		assert.True(t, n.Fulfilled(), "node %s (%s) left %s", n.Name, n.Type, n.Phase)
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

func TestRootRetryStrategyCompletes(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/root-retry-strategy-completes.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestRootRetryStrategyWithInlineTemplateRetries(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/root-retry-strategy-with-inline-template-completes.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.NodePending, woc.wf.Status.Nodes.FindByDisplayName("hello(0)").Phase)
}

func TestInlineTemplateReferenceOuterTemplateWorkflow(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/inline-template-reference-outer-template-workflow.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes.FindByDisplayName("step1").Phase)
	assert.Equal(t, wfv1.NodePending, woc.wf.Status.Nodes.FindByDisplayName("fail").Phase)
}

func TestSubstituteGlobalVariables(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/global-param-substitute.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	_, err := woc.setExecWorkflow(ctx)
	require.NoError(t, err)
	assert.NotNil(t, woc.execWf)
	assert.Equal(t, "mutex1", woc.execWf.Spec.Synchronization.Mutexes[0].Name)
	tempStr, err := json.Marshal(woc.execWf.Spec.Templates)
	require.NoError(t, err)
	assert.Contains(t, string(tempStr), "{{workflow.parameters.message}}")
}

// test that Labels and Annotations are correctly substituted in the case of
// a Workflow referencing a WorkflowTemplate, where Labels and Annotations can come from:
// - Workflow metadata
// - Workflow spec.workflowMetadata
// - WorkflowTemplate spec.workflowMetadata
func TestSubstituteGlobalVariablesLabelsAnnotations(t *testing.T) {
	tests := []struct {
		name                  string
		workflow              string
		workflowTemplate      string
		expectedMutexName     string
		expectedSchedulerName string
	}{
		{
			// entire template referenced; value not contained in WorkflowTemplate or Workflow
			workflow:              "@testdata/workflow-sub-test-1.yaml",
			workflowTemplate:      "@testdata/workflow-template-sub-test-1.yaml",
			expectedMutexName:     "{{workflow.labels.mutexName}}",
			expectedSchedulerName: "{{workflow.annotations.schedulerName}}",
		},
		{
			// entire template referenced; value is in Workflow.Labels
			workflow:              "@testdata/workflow-sub-test-2.yaml",
			workflowTemplate:      "@testdata/workflow-template-sub-test-1.yaml",
			expectedMutexName:     "myMutex",
			expectedSchedulerName: "myScheduler",
		},
		{
			// entire template referenced; value is in WorkflowTemplate.workflowMetadata
			workflow:              "@testdata/workflow-sub-test-2.yaml",
			workflowTemplate:      "@testdata/workflow-template-sub-test-2.yaml",
			expectedMutexName:     "wfMetadataTemplateMutex",
			expectedSchedulerName: "wfMetadataTemplateScheduler",
		},
		{
			// entire template referenced; value is in Workflow.workflowMetadata
			workflow:              "@testdata/workflow-sub-test-3.yaml",
			workflowTemplate:      "@testdata/workflow-template-sub-test-2.yaml",
			expectedMutexName:     "wfMetadataMutex",
			expectedSchedulerName: "wfMetadataScheduler",
		},
		// test using LabelsFrom
		{
			workflow:              "@testdata/workflow-sub-test-4.yaml",
			workflowTemplate:      "@testdata/workflow-template-sub-test-3.yaml",
			expectedMutexName:     "wfMetadataTemplateMutex",
			expectedSchedulerName: "wfMetadataScheduler",
		},
		{
			// just a single template from the WorkflowTemplate is referenced:
			// shouldn't have access to the global scope of the WorkflowTemplate
			workflow:              "@testdata/workflow-sub-test-5.yaml",
			workflowTemplate:      "@testdata/workflow-template-sub-test-4.yaml",
			expectedMutexName:     "myMutex",
			expectedSchedulerName: "myScheduler",
		},
		{
			// this checks that we can use a sprig expression to set a label (using workflowMetadata.labelsFrom)
			// and the result of that label can also be evaluated in the spec
			workflow:              "@testdata/workflow-sub-test-6.yaml",
			workflowTemplate:      "@testdata/workflow-template-sub-test-2.yaml",
			expectedMutexName:     "wfMetadataScheduler",
			expectedSchedulerName: "wfMetadataScheduler",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(tt.workflow)
			wftmpl := wfv1.MustUnmarshalWorkflowTemplate(tt.workflowTemplate)
			cancel, controller := newController(logging.TestContext(t.Context()), wf, wftmpl)
			defer cancel()
			ctx := logging.TestContext(t.Context())

			woc := newWorkflowOperationCtx(ctx, wf, controller)
			_, err := woc.setExecWorkflow(ctx)

			require.NoError(t, err)
			assert.NotNil(t, woc.execWf)
			assert.Equal(t, tt.expectedMutexName, woc.execWf.Spec.Synchronization.Mutexes[0].Name)
			assert.Equal(t, tt.expectedSchedulerName, woc.execWf.Spec.SchedulerName)
		})
	}
}

func TestWfPendingWithNoPod(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wf-pending.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)
}

func TestMutexWfPendingWithNoPod(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wf-pending-with-sync.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(key string) {
	}, workflowExistenceFunc, false)

	// preempt lock
	_, _, _, _, err := controller.syncManager.TryAcquire(ctx, wf, "test", &wfv1.Synchronization{Mutexes: []*wfv1.Mutex{{Name: "welcome"}}})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Equal(t, wfv1.NodePending, woc.wf.Status.Nodes.FindByDisplayName("hello-world-mpdht").Phase)
	assert.Equal(t, "Waiting for argo/Mutex/welcome lock. Lock status: 0/1", woc.wf.Status.Nodes.FindByDisplayName("hello-world-mpdht").Message)

	woc.controller.syncManager.Release(ctx, wf, "test", &wfv1.Synchronization{Mutexes: []*wfv1.Mutex{{Name: "welcome"}}})
	woc.operate(ctx)
	assert.Empty(t, woc.wf.Status.Nodes.FindByDisplayName("hello-world-mpdht").Message)
}

func TestWFGlobalArtifactNil(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wf-global-artifact-nil.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed, func(pod *apiv1.Pod, _ *wfOperationCtx) {
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

func TestDagTwoChildrenWithNonExpectedNodeType(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/dag-two-children-with-non-expected-node-type.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	sentNode := woc.wf.Status.Nodes.FindByDisplayName("sent")

	// Ensure that both child tasks are labeled as children of the "sent" node
	assert.Len(t, sentNode.Children, 2)
}

// In this test, a pod originating from a container set should not be its own outbound node. "a" should only have one child
// and "main" should be the outbound node.
func TestDagTwoChildrenContainerSet(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/dag-two-children-container-set.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	woc.operate(ctx)

	sentNode := woc.wf.Status.Nodes.FindByDisplayName("a")

	assert.Len(t, sentNode.Children, 1)
}

// TestOperatorRetryExpression tests that retryStrategy.expression works correctly. In this test, the latest child node has
// just failed with exit code 2. The retryStrategy's when condition specifies that retries must only be done when the
// last exit code is NOT 2. We expect the retryStrategy to fail (even though it has 8 tries remainng).
func TestOperatorRetryExpression(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/operator-retry-expression.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	retryNode, err := woc.wf.GetNodeByName("retry-script-9z9pv[1].retry")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, retryNode.Phase)
	assert.Len(t, retryNode.Children, 2)
	assert.Equal(t, "retryStrategy.expression evaluated to false", retryNode.Message)
}

func TestBuildRetryStrategyLocalScope(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/operator-retry-expression.yaml")
	retryNode, err := wf.GetNodeByName("retry-script-9z9pv[1].retry")
	require.NoError(t, err)

	localScope := buildRetryStrategyLocalScope(retryNode, wf.Status.Nodes)

	assert.Len(t, localScope, 5)
	assert.Equal(t, "1", localScope[varkeys.Retries.Template()])
	assert.Equal(t, "1", localScope[varkeys.RetriesLastExitCode.Template()])
	assert.Equal(t, string(wfv1.NodeFailed), localScope[varkeys.RetriesLastStatus.Template()])
	assert.Equal(t, "6", localScope[varkeys.RetriesLastDuration.Template()])
	assert.Equal(t, "Error (exit code 1)", localScope[varkeys.RetriesLastMessage.Template()])
}

// TestOperatorRetryExpressionError tests that retryStrategy.expression works when the last node Errored.
// The expression says retry "true" and as the policy is not specified
// we are proving that the policy applied was Always not OnFailure
func TestOperatorRetryExpressionError(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/operator-retry-expression-error.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	retryNode, err := woc.wf.GetNodeByName("retry-script-9z9pv[1].retry")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeRunning, retryNode.Phase)
	assert.Len(t, retryNode.Children, 3)
}

// TestOperatorRetryExpressionErrorNoExpr tests that the default policy becomes
// OnFailure by having no expression and no explicit retryPolicy
func TestOperatorRetryExpressionErrorNoExpr(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/operator-retry-expression-error-no-expr.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	retryNode, err := woc.wf.GetNodeByName("retry-script-9z9pv[1].retry")
	require.NoError(t, err)

	assert.Equal(t, wfv1.NodeError, retryNode.Phase)
	assert.Len(t, retryNode.Children, 2)
	assert.Equal(t, "Error (exit code 1)", retryNode.Message)
}

func TestExitHandlerWithRetryNodeParam(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/exit-handler-with-retry-node-param.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	retryStepNode, err := woc.wf.GetNodeByName("exit-handler-with-param-xbh52[0].step-1")
	require.NoError(t, err)
	assert.Len(t, retryStepNode.Outputs.Parameters, 1)
	assert.Equal(t, "hello world", retryStepNode.Outputs.Parameters[0].Value.String())
	onExitNode, err := woc.wf.GetNodeByName("exit-handler-with-param-xbh52[0].step-1.onExit")
	require.NoError(t, err)
	assert.Equal(t, "hello world", onExitNode.Inputs.Parameters[0].Value.String())
}

func TestReOperateCompletedWf(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/re-operate-completed-wf.yaml")
	wf.Status.Phase = wfv1.WorkflowError
	wf.Status.FinishedAt = metav1.Now()
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotPanics(t, func() { woc.operate(ctx) })
}

func TestSetWFPodNamesAnnotation(t *testing.T) {
	tests := []struct {
		podNameVersion string
	}{
		{"v1"},
		{"v2"},
	}

	for _, tt := range tests {
		t.Run(tt.podNameVersion, func(t *testing.T) {
			t.Setenv("POD_NAMES", tt.podNameVersion)

			wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/exit-handler-with-retry-node-param.yaml")
			cancel, controller := newController(logging.TestContext(t.Context()), wf)
			defer cancel()

			ctx := logging.TestContext(t.Context())
			woc := newWorkflowOperationCtx(ctx, wf, controller)

			woc.operate(ctx)
			annotations := woc.wf.GetAnnotations()
			assert.Equal(t, annotations[common.AnnotationKeyPodNameVersion], tt.podNameVersion)
		})
	}
}

func TestRetryLoopWithOutputParam(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/retry-loop-with-output-param.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestFailNodesWithoutCreatedPodsAfterDeadlineOrShutdown(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	t.Run("Shutdown", func(t *testing.T) {
		workflow := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-shutting-down-with-nodes-in-pending-after-reconciliation.yaml")
		woc := newWorkflowOperationCtx(ctx, workflow, controller)

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

		woc.failNodesWithoutCreatedPodsAfterDeadlineOrShutdown(ctx)

		assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes[step1NodeName].Phase)
		assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Nodes[step2NodeName].Phase)
	})

	t.Run("RetryNodeMarkedFailedOnTerminate", func(t *testing.T) {
		workflow := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-shutting-down-with-nodes-in-pending-after-reconciliation.yaml")
		woc := newWorkflowOperationCtx(ctx, workflow, controller)

		woc.execWf.Spec.Shutdown = "Terminate"
		woc.execWf.Spec.ActiveDeadlineSeconds = nil

		retryNodeName := "retry-node"
		childNodeName := "retry-node(0)"
		retryNodeID := woc.wf.NodeID(retryNodeName)
		childNodeID := woc.wf.NodeID(childNodeName)

		childNode := wfv1.NodeStatus{
			ID:    childNodeID,
			Name:  childNodeName,
			Type:  wfv1.NodeTypePod,
			Phase: wfv1.NodeFailed,
		}
		retryNode := wfv1.NodeStatus{
			ID:       retryNodeID,
			Name:     retryNodeName,
			Type:     wfv1.NodeTypeRetry,
			Phase:    wfv1.NodeRunning,
			Children: []string{childNodeID},
		}
		woc.wf.Status.Nodes.Set(ctx, childNodeID, childNode)
		woc.wf.Status.Nodes.Set(ctx, retryNodeID, retryNode)

		assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes[retryNodeID].Phase)

		woc.failNodesWithoutCreatedPodsAfterDeadlineOrShutdown(ctx)

		assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Nodes[retryNodeID].Phase)
		assert.Equal(t, "Stopped with strategy 'Terminate'", woc.wf.Status.Nodes[retryNodeID].Message)
	})

	t.Run("RetryNodeStaysRunningWhenChildStillRunning", func(t *testing.T) {
		workflow := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-shutting-down-with-nodes-in-pending-after-reconciliation.yaml")
		woc := newWorkflowOperationCtx(ctx, workflow, controller)

		woc.execWf.Spec.Shutdown = "Terminate"
		woc.execWf.Spec.ActiveDeadlineSeconds = nil

		retryNodeName := "retry-node"
		childNodeName := "retry-node(0)"
		retryNodeID := woc.wf.NodeID(retryNodeName)
		childNodeID := woc.wf.NodeID(childNodeName)

		childNode := wfv1.NodeStatus{
			ID:    childNodeID,
			Name:  childNodeName,
			Type:  wfv1.NodeTypePod,
			Phase: wfv1.NodeRunning,
		}
		retryNode := wfv1.NodeStatus{
			ID:       retryNodeID,
			Name:     retryNodeName,
			Type:     wfv1.NodeTypeRetry,
			Phase:    wfv1.NodeRunning,
			Children: []string{childNodeID},
		}
		woc.wf.Status.Nodes.Set(ctx, childNodeID, childNode)
		woc.wf.Status.Nodes.Set(ctx, retryNodeID, retryNode)

		woc.failNodesWithoutCreatedPodsAfterDeadlineOrShutdown(ctx)

		assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes[retryNodeID].Phase)
	})

	t.Run("RetryNodeMarkedFailedOnStop", func(t *testing.T) {
		workflow := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-shutting-down-with-nodes-in-pending-after-reconciliation.yaml")
		woc := newWorkflowOperationCtx(ctx, workflow, controller)

		woc.execWf.Spec.Shutdown = "Stop"
		woc.execWf.Spec.ActiveDeadlineSeconds = nil

		retryNodeName := "retry-node"
		childNodeName := "retry-node(0)"
		retryNodeID := woc.wf.NodeID(retryNodeName)
		childNodeID := woc.wf.NodeID(childNodeName)

		childNode := wfv1.NodeStatus{
			ID:    childNodeID,
			Name:  childNodeName,
			Type:  wfv1.NodeTypePod,
			Phase: wfv1.NodeFailed,
		}
		retryNode := wfv1.NodeStatus{
			ID:       retryNodeID,
			Name:     retryNodeName,
			Type:     wfv1.NodeTypeRetry,
			Phase:    wfv1.NodeRunning,
			Children: []string{childNodeID},
		}
		woc.wf.Status.Nodes.Set(ctx, childNodeID, childNode)
		woc.wf.Status.Nodes.Set(ctx, retryNodeID, retryNode)

		assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes[retryNodeID].Phase)

		woc.failNodesWithoutCreatedPodsAfterDeadlineOrShutdown(ctx)

		assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Nodes[retryNodeID].Phase)
		assert.Equal(t, "Stopped with strategy 'Stop'", woc.wf.Status.Nodes[retryNodeID].Message)
	})

	t.Run("Deadline", func(t *testing.T) {
		workflow := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-shutting-down-with-nodes-in-pending-after-reconciliation.yaml")
		woc := newWorkflowOperationCtx(ctx, workflow, controller)

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

		woc.failNodesWithoutCreatedPodsAfterDeadlineOrShutdown(ctx)

		assert.Equal(t, wfv1.NodeRunning, woc.wf.Status.Nodes[step1NodeName].Phase)
		assert.Equal(t, wfv1.NodeFailed, woc.wf.Status.Nodes[step2NodeName].Phase)
	})
}

func TestWorkflowTemplateOnExitValidation(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-template-on-exit-validation.yaml")
	wft := wfv1.MustUnmarshalWorkflowTemplate("@testdata/workflow-template-invalid-onexit.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf, wft)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	t.Log(woc.wf)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Contains(t, woc.wf.Status.Message, "invalid spec")
}

func TestWorkflowTemplateEntryPointValidation(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-template-entry-point-validation.yaml")
	wft := wfv1.MustUnmarshalWorkflowTemplate("@testdata/workflow-template-invalid-entrypoint.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf, wft)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	t.Log(woc.wf)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Contains(t, woc.wf.Status.Message, "invalid spec")
}

func nodeWithTemplateName(name string) func(n wfv1.NodeStatus) bool {
	return func(n wfv1.NodeStatus) bool { return n.TemplateName == name }
}

func TestMemoizationTemplateLevelCacheWithStepWithoutCache(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-with-template-level-memoization-and-child-step.yaml")

	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	// Expect both workflowTemplate and the step to be executed
	for _, node := range woc.wf.Status.Nodes {
		t.Logf("%+v", node)
	}
	node := woc.wf.Status.Nodes.Find(nodeWithTemplateName("entrypoint"))
	require.NotNil(t, node, "Entrypoint should exist")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.False(t, node.MemoizationStatus.Hit)
	node = woc.wf.Status.Nodes.Find(nodeWithTemplateName("whalesay"))
	require.NotNil(t, node, "Whalesay step should exist")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
}

func TestMemoizationTemplateLevelCacheWithStepWithCache(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-with-template-level-memoization-and-child-step.yaml")

	// Assume cache is already set
	sampleConfigMapCacheEntry := apiv1.ConfigMap{
		Data: map[string]string{
			"entrypoint-key-1": `{"ExpiresAt":"2020-06-18T17:11:05Z","NodeID":"memoize-abx4124-123129321123","Outputs":{}}`,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "cache-top-entrypoint",
			ResourceVersion: "1630732",
			Labels: map[string]string{
				common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache,
			},
		},
	}

	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())

	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapCacheEntry, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	// Only parent node should exist and it should be a memoization cache hit
	node := woc.wf.Status.Nodes.Find(nodeWithTemplateName("entrypoint"))
	require.NotNil(t, node, "Entrypoint should exist")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.True(t, node.MemoizationStatus.Hit)
	node = woc.wf.Status.Nodes.Find(nodeWithTemplateName("whalesay"))
	require.Nil(t, node, "Whalesay step should not have been executed")
}

// TestMemoizationTemplateLevelCacheStepSavesOutputs ensures that the outputs of a memoized
// steps template are written to the cache, so that a later cache hit can replay them.
func TestMemoizationTemplateLevelCacheStepSavesOutputs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-with-template-level-memoization-and-child-step.yaml")

	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.Find(nodeWithTemplateName("entrypoint"))
	require.NotNil(t, node, "Entrypoint should exist")
	require.Equal(t, wfv1.NodeSucceeded, node.Phase)

	cm, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Get(ctx, "cache-top-entrypoint", metav1.GetOptions{})
	require.NoError(t, err, "Memoization cache ConfigMap should have been created")

	rawEntry, ok := cm.Data["entrypoint-key-1"]
	require.True(t, ok, "Memoization cache should contain an entry for the memoize key")

	var entry cache.Entry
	require.NoError(t, json.Unmarshal([]byte(rawEntry), &entry))
	require.NotNil(t, entry.Outputs, "Cached entry should contain the template outputs, got %s", rawEntry)
	require.Len(t, entry.Outputs.Parameters, 1)
	assert.Equal(t, "url", entry.Outputs.Parameters[0].Name)
	assert.Contains(t, entry.Outputs.Parameters[0].Value.String(), "https://argo-workflows.company.com/workflows/namepace/")
}

func TestMemoizationTemplateLevelCacheWithDagWithoutCache(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-with-template-level-memoization-and-child-dag.yaml")

	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	// Expect both workflowTemplate and the dag to be executed
	node := woc.wf.Status.Nodes.Find(nodeWithTemplateName("entrypoint"))
	require.NotNil(t, node, "Entrypoint should exist")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.False(t, node.MemoizationStatus.Hit)
	node = woc.wf.Status.Nodes.Find(nodeWithTemplateName("whalesay"))
	require.NotNil(t, node, "Whalesay dag should exist")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
}

func TestMemoizationTemplateLevelCacheWithDagWithCache(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-with-template-level-memoization-and-child-dag.yaml")

	// Assume cache is already set
	sampleConfigMapCacheEntry := apiv1.ConfigMap{
		Data: map[string]string{
			"entrypoint-key-1": `{"ExpiresAt":"2020-06-18T17:11:05Z","NodeID":"memoize-abx4124-123129321123","Outputs":{}}`,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            "cache-top-entrypoint",
			ResourceVersion: "1630732",
			Labels: map[string]string{
				common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache,
			},
		},
	}

	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())

	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &sampleConfigMapCacheEntry, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	// Only parent node should exist and it should be a memoization cache hit
	node := woc.wf.Status.Nodes.Find(nodeWithTemplateName("entrypoint"))
	require.NotNil(t, node, "Entrypoint should exist")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.True(t, node.MemoizationStatus.Hit)
	node = woc.wf.Status.Nodes.Find(nodeWithTemplateName("whalesay"))
	require.Nil(t, node, "Whalesay dag should not have been executed")
}

func TestMaxDepth(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/max-depth.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	// Max depth is too small, error expected
	controller.maxStackDepth = 2
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	node := woc.wf.Status.Nodes["hello-world-713168755"]
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodeError, node.Phase)
	assert.Contains(t, node.Message, "Maximum recursion depth exceeded")

	// Max depth is enabled, but not too small, no error expected
	controller.maxStackDepth = 3
	woc = newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	node = woc.wf.Status.Nodes["hello-world-713168755"]
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodePending, node.Phase)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestMaxDepthEnvVariable(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/max-depth.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	// Max depth is disabled, no error expected
	controller.maxStackDepth = 2
	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	t.Setenv("DISABLE_MAX_RECURSION", "true")

	woc.operate(ctx)

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	node := woc.wf.Status.Nodes["hello-world-713168755"]
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodePending, node.Phase)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestGetChildNodeIdsAndLastRetriedNode(t *testing.T) {
	nodeName := "test-node"
	ctx := logging.TestContext(t.Context())
	setup := func() *wfOperationCtx {
		cancel, controller := newController(ctx)
		defer cancel()
		assert.NotNil(t, controller)
		wf := wfv1.MustUnmarshalWorkflow(helloWorldWf)
		assert.NotNil(t, wf)
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		assert.NotNil(t, woc)
		// Verify that there are no nodes in the wf status.
		assert.Empty(t, woc.wf.Status.Nodes)

		// Add the parent node for retries.
		nodeID := woc.wf.NodeID(nodeName)
		_, node := woc.initializeNode(ctx, nodeName, wfv1.NodeTypeRetry, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{}, true)
		woc.wf.Status.Nodes[nodeID] = *node

		// Ensure there are no child nodes yet.
		lastChild := getChildNodeIndex(node, woc.wf.Status.Nodes, -1)
		assert.Nil(t, lastChild)
		return woc
	}
	t.Run("lastChildNode", func(t *testing.T) {
		woc := setup()
		childNodes := []*wfv1.NodeStatus{}
		// Add child nodes.
		for i := range 2 {
			childNode := fmt.Sprintf("%s(%d)", nodeName, i)
			_, n := woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true}, true)
			childNodes = append(childNodes, n)
			woc.addChildNode(ctx, nodeName, childNode)
		}
		node, err := woc.wf.GetNodeByName(nodeName)
		require.NoError(t, err)
		childNodeIds, lastChildNode := getChildNodeIdsAndLastRetriedNode(node, woc.wf.Status.Nodes)

		assert.Len(t, childNodeIds, 2)
		assert.Equal(t, childNodes[1].ID, lastChildNode.ID)
	})

	t.Run("Ignore hooked node", func(t *testing.T) {
		woc := setup()
		childNodes := []*wfv1.NodeStatus{}
		// Add child nodes.
		for i := range 2 {
			childNode := fmt.Sprintf("%s(%d)", nodeName, i)
			_, n := woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true}, true)
			childNodes = append(childNodes, n)
			woc.addChildNode(ctx, nodeName, childNode)
		}

		// Add child hooked nodes
		childNode := fmt.Sprintf("%s.hook.running", nodeName)
		_, n := woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Hooked: true}, true)
		childNodes = append(childNodes, n)
		woc.addChildNode(ctx, nodeName, childNode)

		node, err := woc.wf.GetNodeByName(nodeName)
		require.NoError(t, err)
		childNodeIds, lastChildNode := getChildNodeIdsAndLastRetriedNode(node, woc.wf.Status.Nodes)

		assert.Len(t, childNodeIds, 2)
		// ignore the hooked node
		assert.Equal(t, childNodes[1].ID, lastChildNode.ID)
	})

	t.Run("Retry hooked node", func(t *testing.T) {
		woc := setup()
		childNodes := []*wfv1.NodeStatus{}
		// Add child hooked noes
		for i := range 2 {
			childNode := fmt.Sprintf("%s(%d)", nodeName, i)
			_, n := woc.initializeNode(ctx, childNode, wfv1.NodeTypePod, "", &wfv1.WorkflowStep{}, "", wfv1.NodeRunning, &wfv1.NodeFlag{Retried: true, Hooked: true}, true)
			childNodes = append(childNodes, n)
			woc.addChildNode(ctx, nodeName, childNode)
		}

		node, err := woc.wf.GetNodeByName(nodeName)
		require.NoError(t, err)
		childNodeIds, lastChildNode := getChildNodeIdsAndLastRetriedNode(node, woc.wf.Status.Nodes)

		assert.Len(t, childNodeIds, 2)
		assert.Equal(t, childNodes[1].ID, lastChildNode.ID)
	})
}

func TestRetryWhenEncounterExceededQuota(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/retry-when-encounter-exceeded-quota.yaml")

	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())

	controller.kubeclientset.(*fake.Clientset).CoreV1().(*corefake.FakeCoreV1).PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		createAction, ok := action.(k8stesting.CreateAction)
		assert.True(t, ok)

		pod, ok := createAction.GetObject().(*apiv1.Pod)
		assert.True(t, ok)

		for _, container := range pod.Spec.Containers {
			if container.Name == "main" {
				t.Log("Container args: ", container.Args[0], container.Args[1])
				assert.Equal(t, "1", container.Args[0])
			}
		}

		return true, pod, nil
	})

	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
}

// TestWorkflowNeedReconcile test whether a workflow need reconcile taskresults.
func TestWorkflowNeedReconcile(t *testing.T) {
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/need-reconcile-workflow.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)

	// complete the first pod
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	for _, node := range woc.wf.Status.Nodes {
		woc.wf.Status.MarkTaskResultIncomplete(ctx, node.ID)
	}
	podReconciliationCompleted, err := woc.podReconciliation(ctx)
	require.NoError(t, err)
	assert.False(t, podReconciliationCompleted)

	for idx, node := range woc.wf.Status.Nodes {
		if strings.Contains(node.Name, ".hello1") {
			node.Outputs = &wfv1.Outputs{
				Parameters: []wfv1.Parameter{
					{
						Name:  "workflow_artifact_key",
						Value: wfv1.AnyStringPtr("steps-need-reconcile"),
					},
				},
			}
			woc.wf.Status.Nodes[idx] = node
			woc.wf.Status.MarkTaskResultComplete(ctx, node.ID)
		}
	}
	podReconciliationCompleted, err = woc.podReconciliation(ctx)
	require.NoError(t, err)
	assert.True(t, podReconciliationCompleted)
	woc.operate(ctx)

	// complete the second pod
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	require.Len(t, pods.Items, 2)
	assert.Equal(t, "hello1", pods.Items[0].Spec.Containers[1].Env[0].Value)
	assert.Equal(t, "steps-need-reconcile", pods.Items[1].Spec.Containers[1].Env[0].Value)
}

func TestWorkflowRunningButLabelCompleted(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/workflow-running-but-label-completed.yaml")

	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	reconceilNeeded := reconciliationNeeded(wf)
	assert.False(t, reconceilNeeded)

	delete(wf.Labels, common.LabelKeyCompleted)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.NotEmpty(t, woc.wf.Status.Nodes)
	nodeID := "wf-retry-stopped-pn6mm-1672493720"

	woc.wf.Status.MarkTaskResultIncomplete(ctx, nodeID)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	woc.wf.Status.MarkTaskResultComplete(ctx, nodeID)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)

	delete(wf.Labels, common.LabelKeyCompleted)
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	n := woc.markNodePhase(ctx, wf.Name, wfv1.NodeError)
	assert.Equal(t, wfv1.NodeError, n.Phase)
	woc.wf.Status.MarkTaskResultIncomplete(ctx, nodeID)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	woc.wf.Status.MarkTaskResultComplete(ctx, nodeID)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)

	delete(wf.Labels, common.LabelKeyCompleted)
	woc = newWorkflowOperationCtx(ctx, wf, controller)
	n = woc.markNodePhase(ctx, wf.Name, wfv1.NodeSucceeded)
	assert.Equal(t, wfv1.NodeSucceeded, n.Phase)
	woc.wf.Status.MarkTaskResultIncomplete(ctx, nodeID)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	woc.wf.Status.MarkTaskResultComplete(ctx, nodeID)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestContainerSetWhenPodDeleted tests whether all its children(container) deleted when pod deleted if containerSet is used.
func TestContainerSetWhenPodDeleted(t *testing.T) {
	t.Setenv("RECENTLY_STARTED_POD_DURATION", "0")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wf-has-container-set.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)

	// mark pod Running
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	for _, node := range woc.wf.Status.Nodes {
		if node.Type == wfv1.NodeTypePod {
			assert.Equal(t, wfv1.NodeRunning, node.Phase)
		}
	}

	// delete pod
	deletePods(ctx, woc)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Empty(t, pods.Items)

	// reconcile
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	for _, node := range woc.wf.Status.Nodes {
		assert.Equal(t, wfv1.NodeError, node.Phase)
		if node.Type == wfv1.NodeTypePod {
			assert.Equal(t, "pod deleted", node.Message)
		}
		if node.Type == wfv1.NodeTypeContainer {
			assert.Equal(t, "container deleted", node.Message)
		}
	}
}

// TestContainerSetWithDependenciesWhenPodDeleted tests whether all its children(container) deleted when pod deleted if containerSet with dependencies is used.
func TestContainerSetWithDependenciesWhenPodDeleted(t *testing.T) {
	t.Setenv("RECENTLY_STARTED_POD_DURATION", "0")
	cancel, controller := newController(logging.TestContext(t.Context()))
	defer cancel()
	ctx := logging.TestContext(t.Context())
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wf-has-container-set-with-dependencies.yaml")
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	wf, err = wfcset.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	pods, err := listPods(ctx, woc)
	require.NoError(t, err)
	assert.Len(t, pods.Items, 1)

	// mark pod Running
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	for _, node := range woc.wf.Status.Nodes {
		if node.Type == wfv1.NodeTypePod {
			assert.Equal(t, wfv1.NodeRunning, node.Phase)
		}
	}

	// delete pod
	deletePods(ctx, woc)
	pods, err = listPods(ctx, woc)
	require.NoError(t, err)
	assert.Empty(t, pods.Items)

	// reconcile
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	for _, node := range woc.wf.Status.Nodes {
		assert.Equal(t, wfv1.NodeError, node.Phase)
		if node.Type == wfv1.NodeTypePod {
			assert.Equal(t, "pod deleted", node.Message)
		}
		if node.Type == wfv1.NodeTypeContainer {
			assert.Equal(t, "container deleted", node.Message)
		}
	}
}

func TestGetOutboundNodesFromDAGContainerset(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/dag-containerset-wf.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	found := false
	for _, node := range woc.wf.Status.Nodes {
		if node.Name == "dag-containerset-qlmzl.A.main" {
			assert.Len(t, node.Children, 2)
			found = true
		}
	}
	assert.True(t, found)
}

func TestWithItemsOptionalArtStepsWF(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/with-items-optional-art-steps-wf.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("consume-artifact(0:hello world)")
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodePending, node.Phase)
}

func TestWithItemsOptionalArtDAGWF(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/with-items-optional-art-dag-wf.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("consume-artifact(0:hello world)")
	require.NotNil(t, node)
	assert.Equal(t, wfv1.NodePending, node.Phase)
}

// https://github.com/argoproj/argo-workflows/issues/14426#issuecomment-2870224091
func TestSetGlobalParametersOverriding(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wf-with-value-overriding.yaml")
	t.Run("CheckArgumentFromWF", func(t *testing.T) {
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()
		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)
		woc.operate(ctx)
		assert.Equal(t, "configmap argument overwrite with argument", woc.execWf.Spec.Arguments.Parameters[0].Value.String())
	})
}

func TestMarksWFSucceeded(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/marks-wf-succeeded.yaml")
	cancel, controller := newController(logging.TestContext(t.Context()), wf)
	defer cancel()

	ctx := logging.TestContext(t.Context())
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestWithSequenceExpressionPreservesGlobalScope(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/operator/wf-with-sequence-issue.yaml")
	t.Run("CheckExpressionWithItemAndInputs", func(t *testing.T) {
		cancel, controller := newController(logging.TestContext(t.Context()), wf)
		defer cancel()
		ctx := logging.TestContext(t.Context())
		woc := newWorkflowOperationCtx(ctx, wf, controller)

		// Operate to trigger step expansion
		woc.operate(ctx)

		// The key test is that the workflow should be Running (not Failed)
		// If the expression failed, the workflow would be in Failed state
		assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase,
			"Workflow should be running, indicating expressions were evaluated successfully")

		// Verify no error occurred during expansion
		assert.Empty(t, woc.wf.Status.Message, "No error message should be present")

		// Check that we have the expected number of nodes
		// Should have at least: workflow root + start template + step group + expanded steps
		assert.GreaterOrEqual(t, len(woc.wf.Status.Nodes), 4,
			"Should have created multiple nodes for expanded sequence")
	})
}

// TestPendingTimeoutFailsNode checks that an operation fails a node pending past
// its pendingTimeout but not yet past its timeout (executeProcessedTemplate's
// checkConstraints). The deadline branches themselves are covered by
// TestTemplateTimeoutDeadlines.
func TestPendingTimeoutFailsNode(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("my-ns")
	wf := wfv1.MustUnmarshalWorkflow(pendingTimeoutWf)
	// a timeout far beyond the pendingTimeout, so only the pendingTimeout can fail the node
	wf.Spec.Templates[1].Timeout = "1h"
	wf, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodPending)

	// pendingTimeout is 2s: start the node 3s ago rather than sleeping
	node, err := woc.wf.GetNodeByName("hello-world-dag.dag1")
	require.NoError(t, err)
	node.StartedAt = metav1.NewTime(time.Now().Add(-3 * time.Second))
	woc.wf.Status.Nodes.Set(ctx, node.ID, *node)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	node, err = woc.wf.GetNodeByName("hello-world-dag.dag1")
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
	assert.Equal(t, ErrTimeout.Error(), node.Message)
}

// TestInferFailedReasonArtifactPluginSidecar ensures that a non-zero exit from
// an artifact plugin sidecar does not fail a node whose main and wait
// (or supervisor) containers completed successfully: the aux container tears
// the sidecars down only after all saves succeeded, and the recorded exit code
// can even be a phantom produced by a kill exec racing the sidecar's own exit.
// Failures of main, the aux container, or user sidecars must still fail the
// node as before.
func TestInferFailedReasonArtifactPluginSidecar(t *testing.T) {
	terminated := func(name string, exitCode int32) apiv1.ContainerStatus {
		return apiv1.ContainerStatus{
			Name:  name,
			State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: exitCode, Reason: "Error"}},
		}
	}
	tests := []struct {
		name          string
		ctrs          []apiv1.ContainerStatus
		expectedPhase wfv1.NodePhase
		expectedMsg   string
	}{
		{
			name:          "PluginSidecarExitIgnoredLegacy",
			ctrs:          []apiv1.ContainerStatus{terminated("main", 0), terminated("wait", 0), terminated("artifact-plugin-test", 2)},
			expectedPhase: wfv1.NodeSucceeded,
			expectedMsg:   "",
		},
		{
			name:          "PluginSidecarExitIgnoredInitless",
			ctrs:          []apiv1.ContainerStatus{terminated("main", 0), terminated("supervisor", 0), terminated("artifact-plugin-test", 2)},
			expectedPhase: wfv1.NodeSucceeded,
			expectedMsg:   "",
		},
		{
			name:          "PluginSidecarSigtermExitIgnored",
			ctrs:          []apiv1.ContainerStatus{terminated("main", 0), terminated("wait", 0), terminated("artifact-plugin-test", 143)},
			expectedPhase: wfv1.NodeSucceeded,
			expectedMsg:   "",
		},
		{
			name:          "WaitFailureStillFails",
			ctrs:          []apiv1.ContainerStatus{terminated("main", 0), terminated("wait", 1), terminated("artifact-plugin-test", 2)},
			expectedPhase: wfv1.NodeError,
			expectedMsg:   "wait: Error (exit code 1)",
		},
		{
			name:          "MainFailureStillFails",
			ctrs:          []apiv1.ContainerStatus{terminated("main", 1), terminated("wait", 0), terminated("artifact-plugin-test", 2)},
			expectedPhase: wfv1.NodeFailed,
			expectedMsg:   "main: Error (exit code 1)",
		},
		{
			name:          "UserSidecarFailureStillFails",
			ctrs:          []apiv1.ContainerStatus{terminated("main", 0), terminated("wait", 0), terminated("artifact-plugin-test", 2), terminated("user-sidecar", 1)},
			expectedPhase: wfv1.NodeFailed,
			expectedMsg:   "user-sidecar: Error (exit code 1)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			pod := apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "test-pod"},
				Status:     apiv1.PodStatus{Phase: apiv1.PodFailed, ContainerStatuses: tt.ctrs},
			}
			phase, msg := newWoc(ctx).inferFailedReason(ctx, &pod, nil)
			assert.Equal(t, tt.expectedPhase, phase)
			assert.Equal(t, tt.expectedMsg, msg)
		})
	}
}

func TestSubstitute(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	scope := map[string]string{
		"inputs.parameters.foo": "bar",
		"inputs.parameters.baz": "qux",
	}
	// Case 1: Simple substitution
	res, err := woc.Substitute("{{inputs.parameters.foo}}", scope, nil)
	require.NoError(t, err)
	assert.Equal(t, "bar", res)

	// Case 2: Substitution in expression (non-JSON)
	res, err = woc.Substitute("{{inputs.parameters.foo}} == bar", scope, nil)
	require.NoError(t, err)
	assert.Equal(t, "bar == bar", res)

	// Case 3: Substitution with invalid JSON (should not error with "cannot do template replacements with invalid JSON")
	res, err = woc.Substitute("{{inputs.parameters.foo}} invalid json", scope, nil)
	require.NoError(t, err)
	assert.Equal(t, "bar invalid json", res)
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
