package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestExecuteWfLifeCycleHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/execute-wf-life-cycle-hook.yaml")

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("lifecycle-hook-bgsf6.hooks.error")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
	node = woc.wf.Status.Nodes.FindByDisplayName("lifecycle-hook-bgsf6.hooks.running")
	assert.Nil(t, node)
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
}

func TestExecuteTmplLifeCycleHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/execute-tmpl-life-cycle-hook.yaml")

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("step1.hooks.error")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
	node = woc.wf.Status.Nodes.FindByDisplayName("step1.hooks.running")
	assert.Nil(t, node)
}

func TestWorkflowTemplateRefWithHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/workflow-template-ref-with-hook.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate("@testdata/hooks/wft-with-hook.yaml"))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("step-1.hooks.error")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
}

func TestTemplateRefWithHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/template-ref-with-hook.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate("@testdata/hooks/wft-with-hook.yaml"))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	for _, node := range woc.wf.Status.Nodes {
		fmt.Println(node.DisplayName, node.Phase)
	}
	node := woc.wf.Status.Nodes.FindByDisplayName("step-1.hooks.error")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
}

func TestWfTemplateRefWithHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/wf-template-ref-with-hook.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, wfv1.MustUnmarshalWorkflowTemplate("@testdata/hooks/wft-with-hook.yaml"))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("lifecycle-hook-fh7t4.hooks.Failed")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
}

func TestWfHookHasFailures(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/wf-hook-has-failures.yaml")

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("hook-failures.hooks.failure")
	assert.NotNil(t, node)
	assert.Contains(t,
		woc.globalParams()[varkeys.WorkflowFailures.Template()],
		`[{\"displayName\":\"hook-failures\",\"message\":\"Pod failed\",\"templateName\":\"intentional-fail\",\"phase\":\"Failed\",\"podName\":\"hook-failures\"`,
	)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	makePodsPhase(ctx, woc, apiv1.PodFailed)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	_, err := woc.podReconciliation(ctx)
	require.NoError(t, err)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-failures.hooks.failure")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
}

func TestWfHookNoExpression(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/wf-hook-no-expression.yaml")

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Equal(t, "invalid spec: hooks.failure Expression required", woc.wf.Status.Message)
}

func TestStepHookNoExpression(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/step-hook-no-expression.yaml")

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowFailed, woc.wf.Status.Phase)
	assert.Equal(t, "invalid spec: templates.main.steps[0].step-1.foo Expression required", woc.wf.Status.Message)
}

func TestWfHookWfWaitForTriggeredHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/wf-hook-wf-wait-for-triggered-hook.yaml")

	// Setup
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodRunning)

	// Check if running hook is triggered
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("hook-running.hooks.running")
	assert.NotNil(t, node)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)

	// Make all pods running
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-running.hooks.running")
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)

	// Make main pod completed
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pod, _ := podcs.Get(ctx, "hook-running", metav1.GetOptions{})
	pod.Status.Phase = apiv1.PodSucceeded
	updatedPod, _ := podcs.Update(ctx, pod, metav1.UpdateOptions{})
	woc.wf.Status.MarkTaskResultComplete(ctx, woc.nodeID(pod))
	_ = woc.controller.PodController.TestingPodInformer().GetStore().Update(updatedPod)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.Progress("1/2"), woc.wf.Status.Progress)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-running")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.Nil(t, node.NodeFlag)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-running.hooks.running")
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// Make all pod completed
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.Progress("2/2"), woc.wf.Status.Progress)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-running.hooks.running")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)
	node = woc.wf.Status.Nodes.FindByDisplayName("hook-running")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.Nil(t, node.NodeFlag)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

func TestWfTemplHookWfWaitForTriggeredHook(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/wf-templ-hook-wf-wait-for-triggered-hook.yaml")

	// Setup
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodRunning)

	// Check if running hook is triggered
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	node := woc.wf.Status.Nodes.FindByDisplayName("job.hooks.running")
	assert.NotNil(t, node)
	assert.True(t, node.NodeFlag.Hooked)
	assert.Equal(t, wfv1.NodePending, node.Phase)

	// Make all pods running
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	node = woc.wf.Status.Nodes.FindByDisplayName("job.hooks.running")
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)

	// Make main pod completed
	podcs := woc.controller.kubeclientset.CoreV1().Pods(woc.wf.GetNamespace())
	pods, _ := podcs.List(ctx, metav1.ListOptions{})
	pod := pods.Items[0]
	pod.Status.Phase = apiv1.PodSucceeded
	updatedPod, _ := podcs.Update(ctx, &pod, metav1.UpdateOptions{})
	_ = woc.controller.PodController.TestingPodInformer().GetStore().Update(updatedPod)
	woc.wf.Status.MarkTaskResultComplete(ctx, woc.nodeID(&pod))
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.Progress("1/2"), woc.wf.Status.Progress)
	node = woc.wf.Status.Nodes.FindByDisplayName("job")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.Nil(t, node.NodeFlag)
	node = woc.wf.Status.Nodes.FindByDisplayName("job.hooks.running")
	assert.Equal(t, wfv1.NodeRunning, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// Make all pod completed
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.Progress("2/2"), woc.wf.Status.Progress)
	node = woc.wf.Status.Nodes.FindByDisplayName("job.hooks.running")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.True(t, node.NodeFlag.Hooked)
	node = woc.wf.Status.Nodes.FindByDisplayName("job")
	assert.Equal(t, wfv1.NodeSucceeded, node.Phase)
	assert.Nil(t, node.NodeFlag)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestWfHookParallelismBackpressure verifies that a workflow-level running
// hook held back by the workflow's parallelism limit is treated as
// back-pressure, not as a failure: the hook is preserved as pending, runs once
// a slot frees up, and the workflow finishes successfully instead of Error
// with "Max parallelism reached" (#17511).
func TestWfHookParallelismBackpressure(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/wf-hook-parallelism-backpressure.yaml")

	// Setup
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// The business pod occupies the only parallelism slot, so the hook is held
	// back: the entry node stays Running and the hook is recorded as Pending
	// without a pod of its own, so its trigger survives.
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	entry := woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism")
	require.NotNil(t, entry)
	assert.NotEqual(t, wfv1.NodeError, entry.Phase, "entry node must not be marked Error by the throttled hook: %s", entry.Message)
	hookNode := woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism.hooks.running")
	require.NotNil(t, hookNode, "the held-back hook must be preserved as a pending node")
	assert.Equal(t, wfv1.NodePending, hookNode.Phase)
	assert.True(t, hookNode.NodeFlag.Hooked)
	_, hookPodExists, _ := woc.podExists(hookNode.ID)
	assert.False(t, hookPodExists, "no hook pod may be created while the hook is held back")

	// Business pod running: still one active pod, the hook stays held back.
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	entry = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism")
	require.NotNil(t, entry)
	assert.NotEqual(t, wfv1.NodeError, entry.Phase, "entry node must not be marked Error by the throttled hook: %s", entry.Message)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism.hooks.running")
	require.NotNil(t, hookNode)
	assert.Equal(t, wfv1.NodePending, hookNode.Phase)
	_, hookPodExists, _ = woc.podExists(hookNode.ID)
	assert.False(t, hookPodExists, "no hook pod may be created while the hook is held back")

	// Business pod completed: the slot frees up and the hook is dispatched.
	// The hook pod reaches the pod informer asynchronously from the test's
	// point of view, so wait for its delivery instead of racing it.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism.hooks.running")
	require.NotNil(t, hookNode)
	assert.True(t, hookNode.NodeFlag.Hooked)
	require.Eventually(t, func() bool {
		_, exists, err := woc.podExists(hookNode.ID)
		return err == nil && exists
	}, 10*time.Second, time.Millisecond, "the hook must be dispatched once a parallelism slot is free")
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	// Hook pod completed: the workflow finishes successfully.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism.hooks.running")
	require.NotNil(t, hookNode)
	assert.Equal(t, wfv1.NodeSucceeded, hookNode.Phase)
	assert.True(t, hookNode.NodeFlag.Hooked)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestWfHookParallelismBackpressureMemoized verifies that the node recorded
// while a workflow-level running hook is held back does not mark the memoize
// cache as consulted: once a slot frees up the cache is still read, so a hit
// completes the hook without creating a hook pod (#17511).
func TestWfHookParallelismBackpressureMemoized(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/wf-hook-parallelism-backpressure-memoized.yaml")

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	// A cache hit for the hook's memoize key, saved before this workflow ran.
	_, err := controller.kubeclientset.CoreV1().ConfigMaps("default").Create(ctx, &apiv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "notify-cache",
			Namespace: "default",
			Labels:    map[string]string{common.LabelKeyConfigMapType: common.LabelValueTypeConfigMapCache},
		},
		Data: map[string]string{
			"notify-key": `{"nodeID":"old","outputs":{},"creationTimestamp":"2020-09-21T18:12:56Z"}`,
		},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Held back: the hook is preserved, but without a pod of its own and
	// without the memoize cache being marked as consulted.
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	hookNode := woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-memoized.hooks.running")
	require.NotNil(t, hookNode, "the held-back hook must be preserved as a pending node")
	assert.Equal(t, wfv1.NodePending, hookNode.Phase)
	require.Nil(t, hookNode.MemoizationStatus, "the cache must not be marked as consulted while the hook cannot run")
	_, hookPodExists, _ := woc.podExists(hookNode.ID)
	assert.False(t, hookPodExists, "no hook pod may be created while the hook is held back")

	// Business pod running: still one active pod, the hook stays held back.
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-memoized.hooks.running")
	require.NotNil(t, hookNode)
	assert.Equal(t, wfv1.NodePending, hookNode.Phase)
	require.Nil(t, hookNode.MemoizationStatus, "the cache must not be marked as consulted while the hook cannot run")

	// Slot freed: the cache is consulted now, and the hit completes the hook
	// without creating a hook pod.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-memoized.hooks.running")
	require.NotNil(t, hookNode)
	require.NotNil(t, hookNode.MemoizationStatus)
	assert.True(t, hookNode.MemoizationStatus.Hit, "the hook must read the memoize cache once it can run")
	assert.Equal(t, wfv1.NodeSucceeded, hookNode.Phase)
	_, hookPodExists, _ = woc.podExists(hookNode.ID)
	assert.False(t, hookPodExists, "a cache hit must not create a hook pod")
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestWfHookParallelismBackpressureSibling verifies that sibling hooks keep
// being driven while one of them is held back: the first hook in iteration
// order takes the freed parallelism slot, its sibling is throttled a second
// time and re-encounters its already-recorded node, which must be left
// untouched. The second hook references its template through a templateRef, so
// the recorded node must carry the scope its reference was made from — the
// scope the reconciler records for the task — not the resolved one (#17511).
func TestWfHookParallelismBackpressureSibling(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/wf-hook-parallelism-backpressure-sibling.yaml")
	wft := wfv1.MustUnmarshalWorkflowTemplate("@testdata/hooks/wft-hook-parallelism-backpressure.yaml")

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf, wft)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Both hooks are held back by the entry pod's single parallelism slot:
	// both are preserved as nodes of their own, without pods of their own.
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	entry := woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-sibling")
	require.NotNil(t, entry)
	assert.NotEqual(t, wfv1.NodeError, entry.Phase, "entry node must not be marked Error by a throttled hook: %s", entry.Message)
	first := woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-sibling.hooks.first")
	require.NotNil(t, first, "the held-back inline hook must be preserved as a node of its own")
	assert.Equal(t, wfv1.NodePending, first.Phase)
	second := woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-sibling.hooks.second")
	require.NotNil(t, second, "the held-back templateRef hook must be preserved as a node of its own")
	assert.Equal(t, wfv1.NodePending, second.Phase)
	assert.Equal(t, "local/running-hook-parallelism-sibling", second.TemplateScope,
		"the recorded node must carry the scope its reference was made from, as the reconciler records it for the task")
	_, firstPodExists, _ := woc.podExists(first.ID)
	assert.False(t, firstPodExists, "no hook pod may be created while the hook is held back")
	_, secondPodExists, _ := woc.podExists(second.ID)
	assert.False(t, secondPodExists, "no hook pod may be created while the hook is held back")

	// Business pod running: still one active pod, the hooks stay held back.
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	first = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-sibling.hooks.first")
	require.NotNil(t, first)
	assert.Equal(t, wfv1.NodePending, first.Phase)
	second = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-sibling.hooks.second")
	require.NotNil(t, second)
	assert.Equal(t, wfv1.NodePending, second.Phase)

	// Business pod completed: the slot frees up. Whichever hook comes first
	// takes it and its pod occupies the slot again; its sibling is throttled a
	// second time and must meet its already-recorded node untouched.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	first = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-sibling.hooks.first")
	require.NotNil(t, first)
	second = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-sibling.hooks.second")
	require.NotNil(t, second)
	// Exactly one hook pod must surface in the pod informer — delivery is
	// asynchronous from the test's point of view, and both present would mean
	// both dispatched while neither would mean the slot was not taken at all.
	require.Eventually(t, func() bool {
		_, firstExists, firstErr := woc.podExists(first.ID)
		_, secondExists, secondErr := woc.podExists(second.ID)
		return firstErr == nil && secondErr == nil && firstExists != secondExists
	}, 10*time.Second, time.Millisecond, "exactly one hook must take the freed slot; its sibling stays held back")
	_, secondPodExists, _ = woc.podExists(second.ID)
	heldNode := second
	if secondPodExists {
		heldNode = first
	}
	assert.Equal(t, wfv1.NodePending, heldNode.Phase, "the throttled sibling must stay Pending, its recorded node untouched")

	// The dispatched hook's pod completes; the held-back sibling dispatches now.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	require.Eventually(t, func() bool {
		_, exists, err := woc.podExists(heldNode.ID)
		return err == nil && exists
	}, 10*time.Second, time.Millisecond, "the held-back hook must be dispatched once a slot is free")

	// Both hook pods complete: the workflow finishes successfully.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	first = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-sibling.hooks.first")
	require.NotNil(t, first)
	assert.Equal(t, wfv1.NodeSucceeded, first.Phase)
	second = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-sibling.hooks.second")
	require.NotNil(t, second)
	assert.Equal(t, wfv1.NodeSucceeded, second.Phase)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestWfHookParallelismBackpressureSteps verifies that a held-back
// workflow-level hook whose template is composite is recorded with the phase
// its own dispatch would give it: composite nodes initialize straight to
// Running (executeSteps), so the recorded node must too, and it must start
// orchestrating only once a slot is free (#17511).
func TestWfHookParallelismBackpressureSteps(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/wf-hook-parallelism-backpressure-steps.yaml")

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Held back: the composite hook is recorded as a Running steps node — the
	// phase executeSteps gives one — with no children and no pod of its own.
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	entry := woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-steps")
	require.NotNil(t, entry)
	assert.NotEqual(t, wfv1.NodeError, entry.Phase, "entry node must not be marked Error by the throttled hook: %s", entry.Message)
	hookNode := woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-steps.hooks.running")
	require.NotNil(t, hookNode, "the held-back hook must be preserved as a node of its own")
	assert.Equal(t, wfv1.NodeTypeSteps, hookNode.Type)
	assert.Equal(t, wfv1.NodeRunning, hookNode.Phase)
	assert.Empty(t, hookNode.Children, "a held-back hook must not orchestrate any step yet")
	_, hookPodExists, _ := woc.podExists(hookNode.ID)
	assert.False(t, hookPodExists, "no pod may be created while the hook is held back")

	// Business pod running: still one active pod, the hook stays held back.
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-steps.hooks.running")
	require.NotNil(t, hookNode)
	assert.Equal(t, wfv1.NodeRunning, hookNode.Phase)
	assert.Empty(t, hookNode.Children, "a held-back hook must not orchestrate any step yet")

	// Business pod completed: the slot frees up and the hook runs its steps.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-steps.hooks.running")
	require.NotNil(t, hookNode)
	assert.Equal(t, wfv1.NodeRunning, hookNode.Phase)
	require.NotEmpty(t, hookNode.Children, "the dispatched hook must orchestrate its steps")
	stepGroup, err := woc.wf.Status.Nodes.Get(hookNode.Children[0])
	require.NoError(t, err)
	require.NotEmpty(t, stepGroup.Children, "the dispatched hook must run its step")
	announce, err := woc.wf.Status.Nodes.Get(stepGroup.Children[0])
	require.NoError(t, err)
	// The step pod reaches the pod informer asynchronously from the test's
	// point of view, so wait for its delivery instead of racing it.
	require.Eventually(t, func() bool {
		_, exists, err := woc.podExists(announce.ID)
		return err == nil && exists
	}, 10*time.Second, time.Millisecond, "the dispatched hook's step must run in a pod")

	// The hook's step completes: the workflow finishes successfully.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-steps.hooks.running")
	require.NotNil(t, hookNode)
	assert.Equal(t, wfv1.NodeSucceeded, hookNode.Phase)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestWfHookParallelismBackpressureRetry verifies that a held-back hook whose
// template inherits the workflow's spec.retryStrategy is recorded as the Retry
// node an unthrottled dispatch would create (handleRetries injects one before
// dispatching), so its attempts resolve their Retry parent — for retry
// anti-affinity and per-attempt metrics (#17511).
func TestWfHookParallelismBackpressureRetry(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/wf-hook-parallelism-backpressure-retry.yaml")

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// The business pod occupies the only parallelism slot: the hook is held
	// back and recorded as the Retry node its dispatch would have created —
	// the inherited retry strategy applies to a hook like to any template.
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	hookNode := woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-retry.hooks.running")
	require.NotNil(t, hookNode, "the held-back hook must be preserved as a node of its own")
	assert.Equal(t, wfv1.NodeTypeRetry, hookNode.Type, "the recorded node must be the Retry node the inherited strategy implies")
	assert.Equal(t, wfv1.NodeRunning, hookNode.Phase)
	// Deliberately not marked Retried the way handleRetries marks an injected
	// Retry node: getChildNodeIdsRetried would then count this hooked child as
	// an attempt of its parent Retry node (the workflow root under
	// spec.retryStrategy) and re-dispatch the entry template under the hook's
	// name — hooked children are what that filter exists to exclude.
	require.NotNil(t, hookNode.NodeFlag)
	assert.False(t, hookNode.NodeFlag.Retried, "the recorded hook must stay excluded from its parent Retry node's attempts")
	assert.Empty(t, hookNode.Children, "a held-back hook must not have run an attempt yet")
	_, hookPodExists, _ := woc.podExists(hookNode.ID)
	assert.False(t, hookPodExists, "no hook pod may be created while the hook is held back")

	// Business pod running: still one active pod, the hook stays held back.
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-retry.hooks.running")
	require.NotNil(t, hookNode)
	assert.Equal(t, wfv1.NodeTypeRetry, hookNode.Type)
	assert.Empty(t, hookNode.Children, "a held-back hook must not have run an attempt yet")

	// Business pod completed: the slot frees up and the hook dispatches its
	// first attempt under the recorded Retry node.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-retry.hooks.running")
	require.NotNil(t, hookNode)
	require.Len(t, hookNode.Children, 1, "the dispatch must run its attempt under the recorded Retry node")
	attempt, err := woc.wf.Status.Nodes.Get(hookNode.Children[0])
	require.NoError(t, err)
	require.NotNil(t, woc.wf.Status.Nodes.FindRetryNodeByChild(attempt.ID),
		"the attempt must resolve its Retry parent, as it would after an unthrottled dispatch")
	// The attempt pod reaches the pod informer asynchronously from the test's
	// point of view, so wait for its delivery instead of racing it.
	require.Eventually(t, func() bool {
		_, exists, err := woc.podExists(attempt.ID)
		return err == nil && exists
	}, 10*time.Second, time.Millisecond, "the hook must be dispatched once a parallelism slot is free")

	// The attempt completes: the Retry node — and the workflow — succeed.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-retry.hooks.running")
	require.NotNil(t, hookNode)
	assert.Equal(t, wfv1.NodeSucceeded, hookNode.Phase)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}

// TestWfHookParallelismBackpressureSuspend verifies that the suspend input
// preparation executeSuspend performs when it creates a node also runs for a
// node recorded while the hook was held back: such a node is found existing
// on re-entry, so executeSuspend's own preparation never sees it and the
// enum/default the suspend input form shows would otherwise be lost (#17511).
func TestWfHookParallelismBackpressureSuspend(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/wf-hook-parallelism-backpressure-suspend.yaml")

	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// The business pod occupies the only parallelism slot: the suspend hook is
	// held back and recorded with its inputs already prepared — enum and
	// default extracted from the input value, as executeSuspend would.
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	hookNode := woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-suspend.hooks.running")
	require.NotNil(t, hookNode, "the held-back hook must be preserved as a node of its own")
	assert.Equal(t, wfv1.NodeTypeSuspend, hookNode.Type)
	assert.Equal(t, wfv1.NodePending, hookNode.Phase)
	decision := hookNode.Inputs.GetParameterByName("decision")
	require.NotNil(t, decision, "the recorded node must carry its template's inputs")
	require.Len(t, decision.Enum, 2, "the suspend input's enum must be resolved when the node is recorded")
	assert.Equal(t, "approve", decision.Enum[0].String())
	require.NotNil(t, decision.Default, "the suspend input's default must be resolved when the node is recorded")
	assert.Equal(t, "approve", decision.Default.String())
	_, hookPodExists, _ := woc.podExists(hookNode.ID)
	assert.False(t, hookPodExists, "no hook pod may be created while the hook is held back")

	// Business pod running: still one active pod, the hook stays held back
	// with its prepared inputs untouched.
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-suspend.hooks.running")
	require.NotNil(t, hookNode)
	assert.Equal(t, wfv1.NodePending, hookNode.Phase)
	decision = hookNode.Inputs.GetParameterByName("decision")
	require.NotNil(t, decision)
	require.Len(t, decision.Enum, 2, "the resolved inputs must survive the held-back operations")

	// Business pod completed: the slot frees up and the hook suspends. Its
	// node already exists, so executeSuspend skips its own input preparation —
	// what the record-time preparation resolved is what the node keeps.
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	hookNode = woc.wf.Status.Nodes.FindByDisplayName("running-hook-parallelism-suspend.hooks.running")
	require.NotNil(t, hookNode)
	assert.Equal(t, wfv1.NodeRunning, hookNode.Phase, "the hook must suspend once a slot is free")
	decision = hookNode.Inputs.GetParameterByName("decision")
	require.NotNil(t, decision)
	require.Len(t, decision.Enum, 2, "the resolved inputs must survive the dispatch")
	require.NotNil(t, decision.Default)
	assert.Equal(t, "approve", decision.Default.String())
}

// TestExitHookSkippedOutputAbsenceSemantics verifies that hook/exit-handler arguments mirror the
// task/step absence semantics for a skipped node's defaultless output: a pure reference is dropped
// so the hook template's own input default applies, and an inline `??` expression sees the absent
// (nil) optional and falls back, instead of both being clobbered with "".
func TestExitHookSkippedOutputAbsenceSemantics(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/steps-skipped-ref-exit-hook.yaml")
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx) // producer skipped, work pod created

	producer := woc.wf.Status.Nodes.FindByDisplayName("producer")
	require.NotNil(t, producer)
	require.Equal(t, wfv1.NodeSkipped, producer.Phase)

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx) // work succeeded -> exit hook fires

	hook := woc.wf.Status.Nodes.FindByDisplayName("work.onExit")
	require.NotNil(t, hook, "exit hook should run")
	require.NotNil(t, hook.Inputs)

	in := hook.Inputs.GetParameterByName("in")
	require.NotNil(t, in)
	require.NotNil(t, in.Value)
	assert.Equal(t, "FALLBACK", in.Value.String(), "hook template's own input default must apply for a skipped defaultless ref")

	in2 := hook.Inputs.GetParameterByName("in2")
	require.NotNil(t, in2)
	require.NotNil(t, in2.Value)
	assert.Equal(t, "hook-fallback", in2.Value.String(), "?? must fire on the absent optional in hook arguments")
}
