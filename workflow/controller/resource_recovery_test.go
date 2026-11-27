package controller

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func resourceRecoveryWorkflow(action string) *wfv1.Workflow {
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: "resource-recovery", Namespace: "default"},
		Spec: wfv1.WorkflowSpec{
			Entrypoint: "resource",
			Templates: []wfv1.Template{{
				Name: "resource",
				Resource: &wfv1.ResourceTemplate{
					Action: action,
					Manifest: `apiVersion: v1
kind: ConfigMap
metadata:
  name: recover-me
`,
				},
			}},
		},
		Status: wfv1.WorkflowStatus{Nodes: wfv1.Nodes{
			"resource-recovery": {
				ID: "resource-recovery", Name: "resource-recovery", TemplateName: "resource",
				Type: wfv1.NodeTypePod, Phase: wfv1.NodeRunning,
				StartedAt: metav1.NewTime(time.Now().Add(-time.Minute)),
			},
		}},
	}
}

func failedResourcePod(uid string) *apiv1.Pod {
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "resource-recovery", Namespace: "default", UID: types.UID(uid),
			Annotations: map[string]string{common.AnnotationKeyNodeID: "resource-recovery"},
		},
		Status: apiv1.PodStatus{Phase: apiv1.PodFailed, Reason: "Evicted"},
	}
}

func TestResourcePodRecoveryIndependentOfFailedPodRestart(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "enabled"}[enabled], func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := resourceRecoveryWorkflow("create")
			cancel, controller := newController(ctx, wf)
			defer cancel()
			controller.Config.FailedPodRestart = &config.FailedPodRestartConfig{
				Enabled: enabled, MaxRestarts: new(int32(99)),
			}
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			node := wf.Status.Nodes["resource-recovery"]
			for attempt := int32(1); attempt <= maxResourcePodRecoveries; attempt++ {
				pod := failedResourcePod(fmt.Sprintf("failed-%d", attempt))
				got := woc.assessNodeStatus(ctx, pod, &node)
				require.NotNil(t, got)
				require.Equal(t, wfv1.NodePending, got.Phase)
				assert.Equal(t, attempt, got.ResourcePodRecoveries)
				assert.Equal(t, string(pod.UID), got.RecoveringResourcePodUID)
				assert.Zero(t, got.FailedPodRestarts)
				assert.Empty(t, got.RestartingPodUID)
				cleanupKey := "default/resource-recovery/deletePodByUID/" + string(pod.UID)
				assert.Equal(t, 1, controller.PodController.TestingQueueNumRequeues(cleanupKey))

				// Persist/reload the status as a controller restart would do.
				data, err := json.Marshal(got)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(data, &node))
				require.Nil(t, woc.assessNodeStatus(ctx, pod, &node),
					"another observation of the failed pod must not consume an attempt")
				assert.Equal(t, 2, controller.PodController.TestingQueueNumRequeues(cleanupKey),
					"reissue a lost deletion without charging another recovery")
			}
			// Even with FailedPodRestart enabled and a higher limit, recovery
			// must stop at its own cap instead of falling through to restarts.
			got := woc.assessNodeStatus(ctx, failedResourcePod("last"), &node)
			require.NotNil(t, got)
			assert.Equal(t, wfv1.NodeFailed, got.Phase)
			assert.Equal(t, int32(maxResourcePodRecoveries), got.ResourcePodRecoveries)
			assert.Zero(t, got.FailedPodRestarts)
		})
	}
}

func TestResourcePodRecoveryObservesReplacement(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := resourceRecoveryWorkflow("get")
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	node := wf.Status.Nodes["resource-recovery"]
	failed := failedResourcePod("old")
	node = *woc.assessNodeStatus(ctx, failed, &node)
	replacement := failed.DeepCopy()
	replacement.UID = "new"
	replacement.Status = apiv1.PodStatus{Phase: apiv1.PodPending}
	got := woc.assessNodeStatus(ctx, replacement, &node)
	require.NotNil(t, got)
	assert.Empty(t, got.RecoveringResourcePodUID)
	assert.Equal(t, int32(1), got.ResourcePodRecoveries)
	replacement.Status.Phase = apiv1.PodRunning
	got = woc.assessNodeStatus(ctx, replacement, got)
	require.NotNil(t, got)
	assert.Equal(t, wfv1.NodeRunning, got.Phase)
	assert.Equal(t, int32(1), got.ResourcePodRecoveries)
}

func TestResourcePodRecoveryMissingPodCountedOnce(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := resourceRecoveryWorkflow("create")
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	for range 4 {
		_, err := woc.podReconciliation(ctx)
		require.NoError(t, err)
		node := woc.wf.Status.Nodes["resource-recovery"]
		assert.Equal(t, wfv1.NodePending, node.Phase)
		assert.Equal(t, int32(1), node.ResourcePodRecoveries)
		assert.Zero(t, node.FailedPodRestarts)
	}
	node := woc.wf.Status.Nodes["resource-recovery"]
	node.Phase = wfv1.NodeRunning
	node.ResourcePodRecoveries = maxResourcePodRecoveries
	woc.wf.Status.Nodes.Set(ctx, node.ID, node)
	_, err := woc.podReconciliation(ctx)
	require.NoError(t, err)
	assert.Equal(t, wfv1.NodeError, woc.wf.Status.Nodes[node.ID].Phase)
}

func TestResourcePodRecoveryDoesNotReplayOtherMutations(t *testing.T) {
	for _, action := range []string{"apply", "patch", "replace", "delete"} {
		t.Run(action, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := resourceRecoveryWorkflow(action)
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			node := wf.Status.Nodes["resource-recovery"]
			assert.False(t, woc.shouldRecoverResourcePod(ctx, &node))
			got := woc.assessNodeStatus(ctx, failedResourcePod("failed"), &node)
			require.NotNil(t, got)
			assert.Equal(t, wfv1.NodeFailed, got.Phase)
			assert.Zero(t, got.ResourcePodRecoveries)
		})
	}
}

func TestResourcePodRecoveryDoesNotReplayAdditionalSources(t *testing.T) {
	for _, flags := range [][]string{
		{"-f", "other.yaml"}, {"-fother.yaml"}, {"--filename=other.yaml"},
		{"-Rfother.yaml"}, {"-k", "directory"}, {"--kustomize=directory"},
		{"--raw=/api/v1/namespaces/default/configmaps"},
	} {
		t.Run(flags[0], func(t *testing.T) {
			tmpl := resourceRecoveryWorkflow("create").Spec.Templates[0]
			tmpl.Resource.Flags = flags
			assert.False(t, resourceTemplateSupportsRecovery(&tmpl))
		})
	}
	tmpl := resourceRecoveryWorkflow("create").Spec.Templates[0]
	tmpl.Resource.Flags = []string{"--namespace=target", "--validate=false"}
	assert.True(t, resourceTemplateSupportsRecovery(&tmpl))
}
