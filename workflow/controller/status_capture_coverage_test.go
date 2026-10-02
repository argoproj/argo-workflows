package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/workqueue"
	clocktesting "k8s.io/utils/clock/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

func TestStatusCaptureCompletionInformerRequiresPersistedReceipt(t *testing.T) {
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	ctx := logging.TestContext(t.Context())
	completed, pod := legacyCaptureFixture(t)
	pod.Finalizers = []string{common.FinalizerPodStatus, "example.com/keep"}
	running := completed.DeepCopy()
	running.Status.Phase = wfv1.WorkflowRunning
	running.Labels[common.LabelKeyCompleted] = "false"
	cancel, controller := newController(ctx, running)
	defer cancel()
	clock := clocktesting.NewFakeClock(time.Unix(1000, 0))
	queue := workqueue.NewTypedRateLimitingQueueWithConfig(
		workqueue.NewTypedItemExponentialFailureRateLimiter[string](0, 0),
		workqueue.TypedRateLimitingQueueConfig[string]{Clock: clock},
	)
	controller.PodController.TestingSetQueue(queue)
	t.Cleanup(queue.ShutDown)
	kube := controller.kubeclientset.(*fake.Clientset)
	_, err := kube.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return queue.Len() == 1 }, time.Second, time.Millisecond)
	// Remove the initial Pod event so only the Workflow filter exit can start
	// this cleanup. The test injects no further Pod events before release.
	initial, quit := queue.Get()
	require.False(t, quit)
	queue.Done(initial)
	queue.Forget(initial)
	kube.ClearActions()

	completed.ResourceVersion = "2"
	_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(completed.Namespace).Update(ctx, completed, metav1.UpdateOptions{})
	require.NoError(t, err)
	observed, err := util.ToUnstructured(completed)
	require.NoError(t, err)
	_, err = controller.dynamicInterface.Resource(wfv1.SchemeGroupVersion.WithResource("workflows")).Namespace(completed.Namespace).Update(ctx, observed, metav1.UpdateOptions{})
	require.NoError(t, err)
	sweepKey := fmt.Sprintf("%s/%s/reconcileWorkflowCleanup/%s", completed.Namespace, completed.Name, completed.UID)
	require.Eventually(t, func() bool {
		return queue.NumRequeues(sweepKey) == 1 && queue.Len() == 1
	}, time.Second, time.Millisecond, "completion must reach the real filtering informer's DeleteFunc")
	require.True(t, controller.PodController.TestingProcessNextItem(ctx)) // Workflow sweep.
	require.Eventually(t, func() bool { return queue.Len() == 1 }, time.Second, time.Millisecond)
	require.True(t, controller.PodController.TestingProcessNextItem(ctx)) // Missing receipt holds cleanup.
	for _, action := range kube.Actions() {
		assert.NotEqual(t, "patch", action.GetVerb(), "completion alone cannot release the capture barrier")
		assert.NotEqual(t, "delete", action.GetVerb())
	}
	captureBridgePod(ctx, t, kube, pod, false)

	// Only the authoritative API gains the receipt. The informer already
	// filtered this Workflow out, so recovery must use its retained retry.
	node := completed.Status.Nodes[completed.Name]
	node.CapturedPodUID = string(pod.UID)
	completed.Status.Nodes[completed.Name] = node
	_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(completed.Namespace).Update(ctx, completed, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return clock.Waiters() >= 2 }, time.Second, time.Millisecond)
	clock.Step(30 * time.Second)
	require.Eventually(t, func() bool { return queue.Len() == 1 }, time.Second, time.Millisecond)
	require.True(t, controller.PodController.TestingProcessNextItem(ctx)) // Retry observes persisted receipt.
	require.Eventually(t, func() bool { return queue.Len() == 1 }, time.Second, time.Millisecond)
	require.True(t, controller.PodController.TestingProcessNextItem(ctx)) // Mutation checks it again.
	captureBridgePod(ctx, t, kube, pod, true)
}

func TestLegacyCaptureExplicitlyUnsyncedResultDoesNotCommit(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf, pod := legacyCaptureFixture(t)
	node := wf.Status.Nodes[wf.Name]
	node.TaskResultSynced = new(false)
	wf.Status.Nodes[wf.Name] = node
	// The current node's explicit false must win over a stale successful map.
	wf.Status.TaskResultsCompletionStatus = map[string]bool{wf.Name: true}
	cancel, controller := newController(ctx, wf)
	defer cancel()
	_, err := controller.kubeclientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)
	client := controller.wfclientset.(*fakewfclientset.Clientset)
	client.ClearActions()
	_, err = controller.recaptureLegacyPod(ctx, pod)
	require.ErrorContains(t, err, "task-result synchronization is pending")
	for _, action := range client.Actions() {
		assert.NotEqual(t, "update", action.GetVerb(), "an explicitly incomplete result cannot gain a capture receipt")
	}
	persisted, err := client.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, wf.Status, persisted.Status)

	node.TaskResultSynced = new(true)
	wf.Status.Nodes[wf.Name] = node
	_, err = client.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, wf, metav1.UpdateOptions{})
	require.NoError(t, err)
	recaptured, err := controller.recaptureLegacyPod(ctx, pod)
	require.NoError(t, err)
	got := recaptured.Status.Nodes[wf.Name]
	assert.Equal(t, string(pod.UID), got.CapturedPodUID)
	got.CapturedPodUID = ""
	recaptured.Status.Nodes[wf.Name] = got
	assert.Equal(t, wf.Status, recaptured.Status, "recapture must retain the synchronized result")
}

func TestLegacyCaptureStoredWorkflowTemplateScope(t *testing.T) {
	for _, cluster := range []bool{false, true} {
		for _, matches := range []bool{false, true} {
			t.Run(fmt.Sprintf("cluster=%t/matches=%t", cluster, matches), func(t *testing.T) {
				wf, pod := legacyCaptureFixture(t)
				wf.Status.StoredWorkflowSpec = wf.Spec.DeepCopy()
				wf.Spec = wfv1.WorkflowSpec{WorkflowTemplateRef: &wfv1.WorkflowTemplateRef{Name: "stored-template", ClusterScope: cluster}}
				node := wf.Status.Nodes[wf.Name]
				scope := "namespaced/"
				if cluster {
					scope = "cluster/"
				}
				node.TemplateScope = scope + "stored-template"
				if !matches {
					node.TemplateScope = scope + "different-template"
				}
				wf.Status.Nodes[wf.Name] = node
				before := wf.DeepCopy()
				got, err := legacySuccessfulNode(wf, pod)
				if matches {
					require.NoError(t, err)
					assert.Equal(t, node, *got, "stored definition and immutable Pod template prove the same result")
				} else {
					require.ErrorContains(t, err, "execution metadata is inconsistent")
					require.Nil(t, got)
				}
				assert.Equal(t, before, wf, "verification must not rewrite a completed result")
			})
		}
	}
}
