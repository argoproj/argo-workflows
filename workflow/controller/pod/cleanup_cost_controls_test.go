package pod

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clienttesting "k8s.io/client-go/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func cleanupCostCaptured(t *testing.T) (*wfv1.Workflow, *apiv1.Pod, *cleanupCostHarness) {
	t.Helper()
	wf := cleanupCostFixture(32)
	p := cleanupCostPod(wf, 0)
	p.Status.Phase = apiv1.PodSucceeded
	p.Finalizers = []string{common.FinalizerPodStatus, "example.com/keep"}
	node := wf.Status.Nodes[p.Annotations[common.AnnotationKeyNodeID]]
	node.Phase = wfv1.NodeSucceeded
	node.TaskResultSynced = new(true)
	node = cleanupCostReceipt(t, node, p.UID)
	wf.Status.Nodes[node.ID] = node
	return wf, p, newCleanupCostHarness(t, wf, []*apiv1.Pod{p}, "offload", false)
}
func cleanupCostProcessHint(t *testing.T, h *cleanupCostHarness, p *apiv1.Pod) {
	t.Helper()
	h.c.addPodEvent(logging.TestContext(t.Context()), p)
	time.Sleep(2 * time.Millisecond)
	h.clock.Step(10 * time.Millisecond)
	require.Eventually(t, func() bool { return h.c.workqueue.Len() > 0 }, time.Second, time.Millisecond)
	h.counts.Attempts++
	require.True(t, h.c.processNextPodCleanupItem(logging.TestContext(t.Context())))
}
func cleanupCostPublish(t *testing.T, h *cleanupCostHarness, wf *wfv1.Workflow, version string, nodes wfv1.Nodes) {
	t.Helper()
	payload, err := json.Marshal(nodes)
	require.NoError(t, err)
	if h.repo.rows == nil {
		h.repo.rows = map[string][]byte{}
	}
	h.repo.rows[version] = payload
	next := wf.DeepCopy()
	next.ResourceVersion = version
	next.Status.Nodes = nil
	next.Status.OffloadNodeStatusVersion = version
	require.NoError(t, h.wfClient.Tracker().Update(wfv1.SchemeGroupVersion.WithResource("workflows"), next, next.Namespace))
}

func TestCleanupCostFreshReferenceBeforeMutation(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, conflict := range []bool{false, true} {
		t.Run(fmt.Sprintf("patch-conflict=%t", conflict), func(t *testing.T) {
			wf, p, h := cleanupCostCaptured(t)
			wrong := wf.Status.Nodes.DeepCopy()
			node := wrong[p.Annotations[common.AnnotationKeyNodeID]]
			node = cleanupCostReceipt(t, node, "old-pod")
			wrong[node.ID] = node
			if conflict {
				first := true
				h.client.PrependReactor("patch", "pods", func(clienttesting.Action) (bool, runtime.Object, error) {
					if !first {
						return false, nil, nil
					}
					first = false
					cleanupCostPublish(t, h, wf, "cost-v2", wrong)
					return true, nil, apierr.NewConflict(schema.GroupResource{Resource: "pods"}, p.Name, fmt.Errorf("changed"))
				})
			}
			cleanupCostProcessHint(t, h, p)
			if !conflict {
				cleanupCostPublish(t, h, wf, "cost-v2", wrong)
			}
			h.drain()
			require.Contains(t, h.pod(p).Finalizers, common.FinalizerPodStatus, "new published result must invalidate an earlier hint")
			gets := h.snapshot().RepoGet
			require.GreaterOrEqual(t, gets, 2)
			cleanupCostPublish(t, h, wf, "cost-v3", wf.Status.Nodes)
			h.clock.Step(30 * time.Second)
			h.drain()
			require.Equal(t, []string{"example.com/keep"}, h.pod(p).Finalizers)
			require.Greater(t, h.snapshot().RepoGet, gets, "eventless retry must read the newly published reference")
		})
	}
}

func TestCleanupCostReadFailureRecovery(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	for _, failure := range []string{"workflow-forbidden", "offload-error", "offload-not-found"} {
		t.Run(failure, func(t *testing.T) {
			_, p, h := cleanupCostCaptured(t)
			broken := true
			switch failure {
			case "workflow-forbidden":
				h.wfClient.PrependReactor("get", "workflows", func(clienttesting.Action) (bool, runtime.Object, error) {
					if broken {
						return true, nil, apierr.NewForbidden(schema.GroupResource{Resource: "workflows"}, "hydration-cost", fmt.Errorf("denied"))
					}
					return false, nil, nil
				})
			case "offload-error":
				h.repo.failure = fmt.Errorf("storage unavailable")
			case "offload-not-found":
				h.repo.failure = apierr.NewNotFound(schema.GroupResource{Resource: "offloads"}, "cost-v1")
			}
			h.c.addPodEvent(logging.TestContext(t.Context()), p)
			h.drain()
			require.Contains(t, h.pod(p).Finalizers, common.FinalizerPodStatus)
			require.Zero(t, h.snapshot().Pod["patch"])
			// Changing only the failed dependency must recover without an event/restart.
			broken = false
			h.repo.failure = nil
			h.clock.Step(30 * time.Second)
			h.drain()
			require.Equal(t, []string{"example.com/keep"}, h.pod(p).Finalizers)
			require.Equal(t, 1, h.snapshot().Pod["patch"])
		})
	}
}

func TestCleanupCostReplacementAndRestart(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	t.Run("Pod UID replacement", func(t *testing.T) {
		_, p, h := cleanupCostCaptured(t)
		cleanupCostProcessHint(t, h, p)
		replacement := p.DeepCopy()
		replacement.UID = "replacement"
		replacement.ResourceVersion = "11"
		require.NoError(t, h.client.Tracker().Update(apiv1.SchemeGroupVersion.WithResource("pods"), replacement, replacement.Namespace))
		h.drain()
		require.Equal(t, replacement.Finalizers, h.pod(p).Finalizers)
		require.Zero(t, h.snapshot().Pod["patch"])
	})
	t.Run("verified old owner replaced", func(t *testing.T) {
		wf, p, h := cleanupCostCaptured(t)
		cleanupCostProcessHint(t, h, p)
		replacement := wf.DeepCopy()
		replacement.UID = "new-owner"
		replacement.Status.Nodes = nil
		replacement.Status.OffloadNodeStatusVersion = "unavailable-new-owner-reference"
		require.NoError(t, h.wfClient.Tracker().Update(wfv1.SchemeGroupVersion.WithResource("workflows"), replacement, replacement.Namespace))
		h.drain()
		require.Equal(t, []string{"example.com/keep"}, h.pod(p).Finalizers)
		require.Equal(t, 1, h.snapshot().RepoGet, "verified orphan cleanup must not read replacement owner's node storage")
		require.Zero(t, h.snapshot().Workflow["update"])
	})
	t.Run("restart retained Pod Add", func(t *testing.T) {
		wf, p, h := cleanupCostCaptured(t)
		h.repo.failure = fmt.Errorf("temporarily unavailable")
		h.c.addPodEvent(logging.TestContext(t.Context()), p)
		h.drain()
		require.Contains(t, h.pod(p).Finalizers, common.FinalizerPodStatus)
		h.c.workqueue.ShutDown()
		restarted := newCleanupCostHarness(t, wf, []*apiv1.Pod{h.pod(p)}, "offload", false)
		restarted.c.addPodEvent(logging.TestContext(t.Context()), p)
		restarted.drain()
		require.Equal(t, []string{"example.com/keep"}, restarted.pod(p).Finalizers)
	})
}
