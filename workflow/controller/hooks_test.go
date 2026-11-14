package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
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

// A retried hook must not be counted as an attempt of the node it hooks.
func TestRetriedHookIsNotCountedAsAttempt(t *testing.T) {
	for _, tt := range []struct {
		name      string
		workflow  string
		retryNode string
		hookNode  string
		// hookRetried: the hook has a retryStrategy, so its pod is "<hookNode>(0)".
		hookRetried bool
	}{
		{
			// Issue #15163's example. The hook has no retryStrategy, so this guards #16290 (no re-run while the hook runs).
			name:      "workflow level hook, template retryStrategy",
			workflow:  "@testdata/hooks/wf-hook-retry-running-hook.yaml",
			retryNode: "wf-hook-retry-running-hook",
			hookNode:  "wf-hook-retry-running-hook.hooks.running",
		},
		{
			name:        "workflow level hook, spec retryStrategy",
			workflow:    "@testdata/hooks/wf-hook-spec-retry-running-hook.yaml",
			retryNode:   "wf-hook-spec-retry-running-hook",
			hookNode:    "wf-hook-spec-retry-running-hook.hooks.running",
			hookRetried: true,
		},
		{
			name:        "step level hook",
			workflow:    "@testdata/hooks/step-hook-retry-running-hook.yaml",
			retryNode:   "step-hook-retry-running-hook(0)[0].step",
			hookNode:    "step-hook-retry-running-hook(0)[0].step.hooks.running",
			hookRetried: true,
		},
		{
			// Memoization creates the hook node before handleRetries does.
			name:        "step level memoized hook",
			workflow:    "@testdata/hooks/step-hook-retry-memoized-running-hook.yaml",
			retryNode:   "step-hook-retry-memoized-running-hook(0)[0].step",
			hookNode:    "step-hook-retry-memoized-running-hook(0)[0].step.hooks.running",
			hookRetried: true,
		},
		{
			name:        "dag task hook",
			workflow:    "@testdata/hooks/dag-hook-retry-running-hook.yaml",
			retryNode:   "dag-hook-retry-running-hook(0).task",
			hookNode:    "dag-hook-retry-running-hook(0).task.hooks.running",
			hookRetried: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(tt.workflow)
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)

			hookPod := tt.hookNode
			if tt.hookRetried {
				hookPod += "(0)"
			}

			// The first attempt succeeds while the hook keeps running.
			setPodPhases(ctx, woc, func(node *wfv1.NodeStatus) apiv1.PodPhase {
				if node.Name == tt.retryNode+"(0)" {
					return apiv1.PodSucceeded
				}
				return apiv1.PodRunning
			})
			// Reconcile 0 completes the retry node; reconcile 1 must leave it alone.
			for i := range 2 {
				woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
				woc.operate(ctx)

				retryNode := woc.wf.Status.Nodes.FindByName(tt.retryNode)
				require.NotNil(t, retryNode, "reconcile %d", i)
				assert.Equal(t, wfv1.NodeSucceeded, retryNode.Phase, "reconcile %d", i)
				assert.Len(t, getChildNodeIdsRetried(retryNode, woc.wf.Status.Nodes), 1, "reconcile %d: the retry node must have a single attempt", i)
				hookNode := woc.wf.Status.Nodes.FindByName(tt.hookNode)
				require.NotNil(t, hookNode, "reconcile %d", i)
				assert.False(t, hookNode.Fulfilled(), "reconcile %d: the hook must still be running", i)
				assert.Equal(t, &wfv1.NodeFlag{Hooked: true}, hookNode.NodeFlag, "reconcile %d: the hook must not be flagged as an attempt", i)
				if tt.hookRetried {
					hookAttempt := woc.wf.Status.Nodes.FindByName(hookPod)
					require.NotNil(t, hookAttempt, "reconcile %d", i)
					assert.Equal(t, &wfv1.NodeFlag{Hooked: true, Retried: true}, hookAttempt.NodeFlag, "reconcile %d", i)
				}
				assert.ElementsMatch(t, []string{tt.retryNode + "(0)", hookPod}, r4PodNodeNames(ctx, t, woc), "reconcile %d: only the attempt and the hook must have run", i)
				assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "reconcile %d", i)
			}

			makePodsPhase(ctx, woc, apiv1.PodSucceeded)
			woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
			woc.operate(ctx)
			assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
		})
	}
}

// A retried exit hook must not be flagged Retried. The workflow succeeds either
// way, so only the flags and attempt count catch it.
func TestRetriedExitHookIsNotCountedAsAttempt(t *testing.T) {
	for _, tt := range []struct {
		name      string
		workflow  string
		retryNode string
		hookNode  string
	}{
		{
			name:      "step exit hook",
			workflow:  "@testdata/hooks/step-exit-hook-retry.yaml",
			retryNode: "step-exit-hook-retry(0)[0].step",
			hookNode:  "step-exit-hook-retry(0)[0].step.onExit",
		},
		{
			name:      "workflow onExit",
			workflow:  "@testdata/hooks/wf-on-exit-retry.yaml",
			retryNode: "wf-on-exit-retry",
			hookNode:  "wf-on-exit-retry.onExit",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(tt.workflow)
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx, wf)
			defer cancel()
			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)
			for range 3 {
				makePodsPhase(ctx, woc, apiv1.PodSucceeded)
				woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
				woc.operate(ctx)
			}
			assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)

			retryNode := woc.wf.Status.Nodes.FindByName(tt.retryNode)
			require.NotNil(t, retryNode)
			assert.Len(t, getChildNodeIdsRetried(retryNode, woc.wf.Status.Nodes), 1, "the retry node must have a single attempt")
			hookNode := woc.wf.Status.Nodes.FindByName(tt.hookNode)
			require.NotNil(t, hookNode)
			assert.Equal(t, &wfv1.NodeFlag{Hooked: true}, hookNode.NodeFlag, "the hook must not be flagged as an attempt")
			hookAttempt := woc.wf.Status.Nodes.FindByName(tt.hookNode + "(0)")
			require.NotNil(t, hookAttempt)
			assert.Equal(t, &wfv1.NodeFlag{Hooked: true, Retried: true}, hookAttempt.NodeFlag)
			assert.ElementsMatch(t, []string{tt.retryNode + "(0)", tt.hookNode + "(0)"}, r4PodNodeNames(ctx, t, woc))
		})
	}
}

// A hook saved as {Hooked, Retried} by an older controller must not be counted
// as an attempt.
func TestLegacyRetriedHookFlagIsNotCountedAsAttempt(t *testing.T) {
	const (
		retryNode = "step-hook-retry-running-hook(0)[0].step"
		hookNode  = retryNode + ".hooks.running"
	)
	wf := wfv1.MustUnmarshalWorkflow("@testdata/hooks/step-hook-retry-running-hook.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodRunning)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	// Simulate a hook saved by an older controller.
	hook := woc.wf.Status.Nodes.FindByName(hookNode)
	require.NotNil(t, hook)
	hook.NodeFlag = &wfv1.NodeFlag{Hooked: true, Retried: true}
	woc.wf.Status.Nodes.Set(ctx, hook.ID, *hook)

	setPodPhases(ctx, woc, func(node *wfv1.NodeStatus) apiv1.PodPhase {
		if node.Name == retryNode+"(0)" {
			return apiv1.PodSucceeded
		}
		return apiv1.PodRunning
	})
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	step := woc.wf.Status.Nodes.FindByName(retryNode)
	require.NotNil(t, step)
	assert.Equal(t, wfv1.NodeSucceeded, step.Phase)
	assert.Len(t, getChildNodeIdsRetried(step, woc.wf.Status.Nodes), 1, "the saved hook must not count as an attempt")
	assert.ElementsMatch(t, []string{retryNode + "(0)", hookNode + "(0)"}, r4PodNodeNames(ctx, t, woc))

	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
}
