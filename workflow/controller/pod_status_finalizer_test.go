package controller

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kwait "k8s.io/apimachinery/pkg/util/wait"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// With ARGO_POD_STATUS_CAPTURE_FINALIZER=true, the pod controller leaves the status finalizer of a deleting pod
// to the pod's running workflow, which removes it once it has persisted the pod's node.

const statusFinalizerWf = `
metadata:
  name: my-wf
  namespace: test
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: c
    - - name: b
        template: c
  - name: c
    container:
      image: my-image
`

// newHeldPodController starts a controller with the flag on until the test ends. Its pod controller rechecks held
// pods at once, as rate limited items, so that the tests see the rechecks queued and run them.
func newHeldPodController(ctx context.Context, t *testing.T, wf *wfv1.Workflow, options ...any) *WorkflowController {
	t.Helper()
	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")
	cancel, controller := newController(ctx, append([]any{wf}, options...)...)
	t.Cleanup(cancel)
	// the fake clientsets drop events written before an informer watches, which it starts after it synced
	watching := func(actions []k8stesting.Action, resource string) bool {
		return slices.ContainsFunc(actions, func(action k8stesting.Action) bool {
			return action.GetVerb() == "watch" && action.GetResource().Resource == resource
		})
	}
	require.Eventually(t, func() bool {
		return watching(controller.kubeclientset.(*fake.Clientset).Actions(), "pods") &&
			watching(controller.dynamicInterface.(*dynamicfake.FakeDynamicClient).Actions(), workflow.WorkflowPlural)
	}, 10*time.Second, time.Millisecond, "the informers are not watching")
	controller.PodController.TestingSetRecheckPeriod(0)
	return controller
}

// startWorkflow starts a controller for statusFinalizerWf, runs its first operation and returns pod a.
func startWorkflow(ctx context.Context, t *testing.T, options ...any) (*WorkflowController, *apiv1.Pod) {
	t.Helper()
	controller := newHeldPodController(ctx, t, wfv1.MustUnmarshalWorkflow(statusFinalizerWf), options...)
	reconcile(ctx, t, controller)
	return controller, podForNode(ctx, t, controller, "my-wf[0].a")
}

// reconcile runs one operation on the persisted workflow, as a workflow worker would, and returns what it persisted.
func reconcile(ctx context.Context, t *testing.T, controller *WorkflowController) *wfv1.Workflow {
	t.Helper()
	workflows := controller.wfclientset.ArgoprojV1alpha1().Workflows("test")
	wf, err := workflows.Get(ctx, "my-wf", metav1.GetOptions{})
	require.NoError(t, err)
	newWorkflowOperationCtx(ctx, wf, controller).operate(ctx)
	wf, err = workflows.Get(ctx, "my-wf", metav1.GetOptions{})
	require.NoError(t, err)
	return wf
}

func podForNode(ctx context.Context, t *testing.T, controller *WorkflowController, nodeName string) *apiv1.Pod {
	t.Helper()
	pods, err := controller.kubeclientset.CoreV1().Pods("test").List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	for _, pod := range pods.Items {
		if pod.Annotations[common.AnnotationKeyNodeName] == nodeName {
			return &pod
		}
	}
	require.FailNow(t, "no pod for node", nodeName)
	return nil
}

// updatePod writes the pod and waits for the pod informer to have it. The fake clientset keeps the
// resourceVersion, and the pod informer ignores updates that keep it, so bump it as the API server would.
func updatePod(ctx context.Context, t *testing.T, controller *WorkflowController, pod *apiv1.Pod) {
	t.Helper()
	rv, _ := strconv.Atoi(pod.ResourceVersion)
	pod.ResourceVersion = strconv.Itoa(rv + 1)
	updated, err := controller.kubeclientset.CoreV1().Pods("test").Update(ctx, pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	waitForInformer(ctx, controller.PodController.TestingPodInformer(), updated, func(obj any) bool {
		return obj.(*apiv1.Pod).ResourceVersion == updated.ResourceVersion
	})
}

// createPod adds a pod that was there before the controller started, and returns it.
func createPod(ctx context.Context, t *testing.T, controller *WorkflowController, pod *apiv1.Pod) *apiv1.Pod {
	t.Helper()
	pod = pod.DeepCopy()
	pod.ResourceVersion = ""
	created, err := controller.kubeclientset.CoreV1().Pods("test").Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)
	waitForInformer(ctx, controller.PodController.TestingPodInformer(), created, func(any) bool { return true })
	return created
}

// succeedPod marks the pod finished, with its last condition transition long ago.
func succeedPod(pod *apiv1.Pod) {
	pod.Status.Phase = apiv1.PodSucceeded
	pod.Status.Conditions = []apiv1.PodCondition{{Type: apiv1.PodReady, LastTransitionTime: metav1.NewTime(time.Now().Add(-10 * time.Minute))}}
}

func workflowUpdates(controller *WorkflowController) int {
	actions := controller.wfclientset.(*fakewfclientset.Clientset).Actions()
	return len(slices.DeleteFunc(actions, func(action k8stesting.Action) bool { return action.GetVerb() != "update" }))
}

func cleanupKey(pod *apiv1.Pod, action string) string {
	return "test/" + pod.Name + "/" + action
}

// assertHeld waits for the pod controller to handle the pod's deletion by queueing a recheck instead of a timed
// removal, runs that recheck, and checks that the pod kept its finalizer for the workflow.
func assertHeld(ctx context.Context, t *testing.T, controller *WorkflowController, pod *apiv1.Pod) {
	t.Helper()
	require.Eventually(t, func() bool {
		return controller.PodController.TestingQueueNumRequeues(cleanupKey(pod, "recheckHeldPod")) > 0
	}, 5*time.Second, time.Millisecond, "%s is not held for its workflow", pod.Name)
	require.True(t, controller.PodController.TestingProcessNextItem(ctx))
	current, err := controller.kubeclientset.CoreV1().Pods("test").Get(ctx, pod.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.Contains(t, current.Finalizers, common.FinalizerPodStatus)
}

// assertReleasedBy waits for the action to be queued for the pod, runs the pod cleanup worker until it has handled
// it, and checks that the pod lost its status finalizer.
func assertReleasedBy(ctx context.Context, t *testing.T, controller *WorkflowController, pod *apiv1.Pod, action string) {
	t.Helper()
	key := cleanupKey(pod, action)
	require.Eventually(t, func() bool {
		return controller.PodController.TestingQueueNumRequeues(key) > 0
	}, 5*time.Second, time.Millisecond, "%s was never queued", key)
	for controller.PodController.TestingQueueNumRequeues(key) > 0 {
		assert.True(t, controller.PodController.TestingProcessNextItem(ctx))
	}
	current, err := controller.kubeclientset.CoreV1().Pods("test").Get(ctx, pod.Name, metav1.GetOptions{})
	require.NoError(t, err)
	assert.NotContains(t, current.Finalizers, common.FinalizerPodStatus)
}

func TestStatusFinalizerKeptUntilNodePersisted(t *testing.T) {
	t.Run("pod succeeded, then deleted while the workflow is not reconciled", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		controller, podA := startWorkflow(ctx, t)
		succeedPod(podA)
		podA.DeletionTimestamp = new(metav1.Now())
		updatePod(ctx, t, controller, podA)
		assertHeld(ctx, t, controller, podA)

		wf := reconcile(ctx, t, controller)
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("my-wf[0].a").Phase)
		assertReleasedBy(ctx, t, controller, podA, "labelPodCompleted")
	})
	t.Run("running pod deleted gracefully, then succeeded within its grace period", func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		controller, podA := startWorkflow(ctx, t)
		podA.Status.Phase = apiv1.PodRunning
		podA.Status.Conditions = []apiv1.PodCondition{{Type: apiv1.PodReady, Status: apiv1.ConditionTrue, LastTransitionTime: metav1.NewTime(time.Now().Add(-30 * time.Minute))}}
		updatePod(ctx, t, controller, podA)
		reconcile(ctx, t, controller)
		podA.DeletionTimestamp = new(metav1.NewTime(time.Now().Add(30 * time.Second)))
		updatePod(ctx, t, controller, podA)
		assertHeld(ctx, t, controller, podA)
		updates := workflowUpdates(controller)
		reconcile(ctx, t, controller)
		assert.Equal(t, updates, workflowUpdates(controller), "the workflow was persisted although nothing changed")

		succeedPod(podA)
		updatePod(ctx, t, controller, podA)
		wf := reconcile(ctx, t, controller)
		assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("my-wf[0].a").Phase)
		assertReleasedBy(ctx, t, controller, podA, "labelPodCompleted")
	})
}

// The controller crashed after it created pod b, before it persisted node b. While it is down pod b succeeds and is
// deleted. After the restart node b is created again and adopts pod b, so its result is still there to take.
func TestStatusFinalizerKeptForUnpersistedNode(t *testing.T) {
	for _, age := range []time.Duration{5 * time.Second, 10 * time.Minute} {
		t.Run("last transition "+age.String()+" ago", func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			controller1, podA := startWorkflow(ctx, t)
			succeedPod(podA)
			updatePod(ctx, t, controller1, podA)
			controller1.wfclientset.(*fakewfclientset.Clientset).PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("connection reset by peer")
			})
			wf := reconcile(ctx, t, controller1)
			require.Nil(t, wf.Status.Nodes.FindByName("my-wf[1].b"))
			podB := podForNode(ctx, t, controller1, "my-wf[1].b")

			succeedPod(podB)
			podB.Status.Conditions[0].LastTransitionTime = metav1.NewTime(time.Now().Add(-age))
			podB.DeletionTimestamp = new(metav1.Now())
			controller2 := newHeldPodController(ctx, t, wf)
			createPod(ctx, t, controller2, podA)
			createPod(ctx, t, controller2, podB)
			assertHeld(ctx, t, controller2, podB)

			reconcile(ctx, t, controller2) // creates node b again, which adopts pod b
			assert.Zero(t, controller2.PodController.TestingQueueNumRequeues(cleanupKey(podB, "removeFinalizer")))
			wf = reconcile(ctx, t, controller2)
			assert.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("my-wf[1].b").Phase)
			assertReleasedBy(ctx, t, controller2, podB, "labelPodCompleted")
		})
	}
}

// Node a was persisted, then the controller restarted before it removed pod a's finalizer. Pod a is deleted before
// or after the restart, and nothing else in the workflow changes after it.
func TestStatusFinalizerReleasedForPersistedNode(t *testing.T) {
	for _, deletedBeforeRestart := range []bool{true, false} {
		t.Run("deleted before restart "+strconv.FormatBool(deletedBeforeRestart), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			controller1, podA := startWorkflow(ctx, t)
			succeedPod(podA)
			updatePod(ctx, t, controller1, podA)
			reconcile(ctx, t, controller1)
			podB := podForNode(ctx, t, controller1, "my-wf[1].b")
			podB.Status.Phase = apiv1.PodRunning
			updatePod(ctx, t, controller1, podB)
			wf := reconcile(ctx, t, controller1)
			require.Equal(t, wfv1.NodeSucceeded, wf.Status.Nodes.FindByName("my-wf[0].a").Phase)
			require.Equal(t, wfv1.NodeRunning, wf.Status.Nodes.FindByName("my-wf[1].b").Phase)

			controller2 := newHeldPodController(ctx, t, wf)
			createPod(ctx, t, controller2, podB)
			if deletedBeforeRestart {
				podA.DeletionTimestamp = new(metav1.Now())
			}
			podA = createPod(ctx, t, controller2, podA)
			if !deletedBeforeRestart {
				podA.DeletionTimestamp = new(metav1.Now())
				updatePod(ctx, t, controller2, podA)
			}
			assertHeld(ctx, t, controller2, podA)

			wf = reconcile(ctx, t, controller2)
			assert.Equal(t, wfv1.WorkflowRunning, wf.Status.Phase)
			assertReleasedBy(ctx, t, controller2, podA, "labelPodCompleted")
		})
	}
}

// A deleting pod held for a running workflow is released when the workflow is deleted.
func TestStatusFinalizerReleasedWhenWorkflowDeleted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	controller, podA := startWorkflow(ctx, t)
	podA.Status.Phase = apiv1.PodRunning
	podA.DeletionTimestamp = new(metav1.Now())
	updatePod(ctx, t, controller, podA)
	assertHeld(ctx, t, controller, podA)

	workflows := controller.dynamicInterface.Resource(wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowPlural))
	require.NoError(t, workflows.Namespace("test").Delete(ctx, "my-wf", metav1.DeleteOptions{}))
	assertReleasedBy(ctx, t, controller, podA, "removeFinalizer")
}

// A running pod on an unreachable node was force deleted 3m ago, so no kubelet will report its result.
func TestStatusFinalizerReleasedForForceDeletedPod(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	controller, podA := startWorkflow(ctx, t)
	podA.Status.Phase = apiv1.PodRunning
	podA.DeletionTimestamp = new(metav1.NewTime(time.Now().Add(-3 * time.Minute)))
	updatePod(ctx, t, controller, podA)
	assertReleasedBy(ctx, t, controller, podA, "recheckHeldPod")
}

// failedPodRestart reset node a to Pending to re-create its failed pod and queued that pod's deletion, which the
// controller lost by restarting. Someone else deleted the pod (e.g. PodGC after a node shutdown).
func TestStatusFinalizerReleasedForRestartingPod(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	withRestart := func(controller *WorkflowController) {
		controller.Config.FailedPodRestart = &config.FailedPodRestartConfig{Enabled: true}
	}
	controller1, podA := startWorkflow(ctx, t, withRestart)
	podA.UID = "pod-a-uid"
	podA.Status.Phase = apiv1.PodFailed
	podA.Status.Reason = "NodeShutdown"
	updatePod(ctx, t, controller1, podA)
	wf := reconcile(ctx, t, controller1)
	require.Equal(t, "pod-a-uid", wf.Status.Nodes.FindByName("my-wf[0].a").RestartingPodUID)

	controller2 := newHeldPodController(ctx, t, wf, withRestart)
	podA.DeletionTimestamp = new(metav1.Now())
	createPod(ctx, t, controller2, podA)
	assertHeld(ctx, t, controller2, podA)

	reconcile(ctx, t, controller2)
	assertReleasedBy(ctx, t, controller2, podA, "removeFinalizer/pod-a-uid")
}

// A Pending pod has no result to wait for: when deleted it is released as before, and created again.
func TestStatusFinalizerNotHeldForPendingPod(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	controller, podA := startWorkflow(ctx, t)
	podA.Status.Phase = apiv1.PodPending
	podA.Status.Conditions = []apiv1.PodCondition{{Type: apiv1.PodScheduled, Status: apiv1.ConditionFalse, Reason: "Unschedulable", LastTransitionTime: metav1.NewTime(time.Now().Add(-10 * time.Minute))}}
	podA.DeletionTimestamp = new(metav1.Now())
	updatePod(ctx, t, controller, podA)
	assertReleasedBy(ctx, t, controller, podA, "removeFinalizer")

	// the API server removes the unscheduled pod once its finalizers are gone
	require.NoError(t, controller.kubeclientset.CoreV1().Pods("test").Delete(ctx, podA.Name, metav1.DeleteOptions{}))
	require.NoError(t, kwait.PollUntilContextTimeout(ctx, time.Millisecond, 5*time.Second, true, func(context.Context) (bool, error) {
		_, exists, err := controller.PodController.TestingPodInformer().GetStore().GetByKey("test/" + podA.Name)
		return !exists, err
	}))
	reconcile(ctx, t, controller)
	assert.Nil(t, podForNode(ctx, t, controller, "my-wf[0].a").DeletionTimestamp, "pod a was not created again")
}
