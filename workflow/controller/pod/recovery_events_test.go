package pod

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestStatusCaptureUpdateRetainsCleanupTriggers(t *testing.T) {
	for _, scenario := range []string{"terminal", "deleting", "replacement", "owner", "node", "policy", "finalizer"} {
		t.Run(scenario, func(t *testing.T) {
			_, old := captureFixture()
			old.Status.Phase = apiv1.PodRunning
			pod := old.DeepCopy()
			pod.ResourceVersion = "next"
			switch scenario {
			case "terminal":
				pod.Status.Phase = apiv1.PodSucceeded
			case "deleting":
				now := metav1.Now()
				pod.DeletionTimestamp = &now
			case "replacement":
				pod.UID = "replacement"
			case "owner":
				pod.OwnerReferences[0].UID = "other-workflow"
			case "node":
				pod.Annotations[common.AnnotationKeyNodeID] = "other-node"
			case "policy":
				pod.Annotations[common.AnnotationKeyPodGCStrategy] = "OnPodCompletion/0s"
			case "finalizer":
				old.Finalizers = nil
			}
			c := identityTestController(t, fake.NewClientset(pod))
			require.True(t, significantPodChange(old, pod), "the informer must deliver cleanup identity/disposition changes")
			c.workqueue = &immediateCleanupRetryQueue{TypedRateLimitingInterface: c.workqueue}
			callbacks := 0
			c.callBack = func(*apiv1.Pod) error { callbacks++; return nil }
			c.updatePodEvent(logging.TestContext(t.Context()), old, pod)
			require.Equal(t, 1, callbacks)
			require.Equal(t, 1, c.workqueue.Len())
			key, quit := c.workqueue.Get()
			require.False(t, quit)
			c.workqueue.Done(key)
			require.Equal(t, newPodCleanupKeyWithUID(pod.Namespace, pod.Name, reconcilePodCleanup, string(pod.UID)), key)
		})
	}
}

func TestCleanupCostLiveUpdatePreservesPendingStopRetry(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("flag=%t", enabled), func(t *testing.T) {
			t.Setenv(common.EnvVarPodStatusCaptureFinalizer, fmt.Sprint(enabled))
			ctx := logging.TestContext(t.Context())
			wf := cleanupCostFixture(32)
			p := cleanupCostPod(wf, 0)
			p.Spec.TerminationGracePeriodSeconds = new(int64(1))
			p.Status.ContainerStatuses = []apiv1.ContainerStatus{{Name: "main", State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}}}
			node := wf.Status.Nodes[p.Annotations[common.AnnotationKeyNodeID]]
			node.Phase = wfv1.NodeFailed
			wf.Status.Nodes[node.ID] = node
			h := newCleanupCostHarness(t, wf, []*apiv1.Pod{p}, "offload", false)
			h.c.addPodEvent(ctx, p)
			h.drain()
			before := h.snapshot()
			require.Equal(t, 1, before.Attempts)
			require.Zero(t, before.Signals, "startup recovery cannot signal without exact capture")

			live := p.DeepCopy()
			live.Status.Message = "still running"
			require.True(t, significantPodChange(p, live))
			h.c.updatePodEvent(ctx, p, live)
			h.drain()
			require.Equal(t, before, h.snapshot(), "ordinary update must not repeat offload hydration")

			node.CapturedPodUID = string(p.UID)
			wf.Status.Nodes[node.ID] = node
			cleanupCostPublish(t, h, wf, "cost-v2", wf.Status.Nodes)
			h.clock.Step(podCleanupRetryDelay)
			h.drain()
			require.Equal(t, 1, h.snapshot().Signals, "existing retry must survive the skipped update and recover without another event")
			h.clock.Step(time.Second)
			h.drain()
			require.Equal(t, 2, h.snapshot().Signals, "recovered stop must retain KILL escalation")
		})
	}
}

func TestCleanupCostWorkflowDeletionAfterLiveUpdate(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	ctx := logging.TestContext(t.Context())
	wf := cleanupCostFixture(32)
	p := cleanupCostPod(wf, 0)
	p.Finalizers = []string{common.FinalizerPodStatus}
	p.Annotations[common.AnnotationKeyPodGCStrategy] = "OnPodCompletion/0s"
	h := newCleanupCostHarness(t, wf, []*apiv1.Pod{p}, "offload", false)
	live := p.DeepCopy()
	live.Status.Message = "still running"
	h.c.updatePodEvent(ctx, p, live)
	h.drain()
	require.Zero(t, h.snapshot().Attempts)

	require.NoError(t, h.wfClient.Tracker().Delete(wfv1.SchemeGroupVersion.WithResource("workflows"), wf.Namespace, wf.Name))
	h.c.QueueWorkflowCleanup(ctx, wf)
	h.drain()
	require.Nil(t, h.pod(p), "workflow deletion must still recover orphan cleanup without another Pod event")
	require.Zero(t, h.snapshot().Hydrate, "an absent owner needs no node hydration")
	require.Equal(t, 1, h.snapshot().Pod["delete"])
}
