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
	wfutil "github.com/argoproj/argo-workflows/v4/workflow/util"
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

// The hook's input comes from a ConfigMap that does not exist: validation has no ConfigMap store
// and accepts it, so the hook fails only when it fires.
const runtimeFailingHookTemplate = `
    - name: handle
      inputs:
        parameters:
          - name: region
            valueFrom:
              configMapKeyRef:
                name: does-not-exist
                key: region
      container:
        image: alpine:3.23
        command: [sh, -c]
        args: ["echo {{inputs.parameters.region}}"]
    - name: fail
      container:
        image: alpine:3.23
        command: [sh, -c]
        args: ["exit 1"]
`

func TestLifecycleHookInitErrorIsLinkedToParent(t *testing.T) {
	tests := []struct {
		name         string
		wf           string
		hookNodeName string
		parentName   string
	}{
		{
			name: "WorkflowLevel",
			wf: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hook-init-error
  namespace: argo
spec:
  entrypoint: fail
  hooks:
    Failed:
      expression: workflow.status == "Failed"
      template: handle
  templates:` + runtimeFailingHookTemplate,
			hookNodeName: "hook-init-error.hooks.Failed",
			parentName:   "hook-init-error",
		},
		{
			name: "Step",
			wf: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hook-init-error
  namespace: argo
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: will-fail
            template: fail
            hooks:
              failure:
                expression: steps["will-fail"].status == "Failed"
                template: handle` + runtimeFailingHookTemplate,
			hookNodeName: "hook-init-error[0].will-fail.hooks.failure",
			parentName:   "hook-init-error[0].will-fail",
		},
		{
			name: "DAGTask",
			wf: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hook-init-error
  namespace: argo
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: will-fail
            template: fail
            hooks:
              failure:
                expression: tasks["will-fail"].status == "Failed"
                template: handle` + runtimeFailingHookTemplate,
			hookNodeName: "hook-init-error.will-fail.hooks.failure",
			parentName:   "hook-init-error.will-fail",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(tt.wf)
			ctx := logging.TestContext(t.Context())
			cancel, controller := newController(ctx, wf)
			defer cancel()

			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)
			makePodsPhase(ctx, woc, apiv1.PodFailed)
			woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
			woc.operate(ctx)

			hookNode := woc.wf.Status.Nodes.FindByName(tt.hookNodeName)
			require.NotNil(t, hookNode)
			assert.Equal(t, wfv1.NodeError, hookNode.Phase)
			assert.Contains(t, hookNode.Message, "unable to retrieve inputs.parameters.region from ConfigMap")
			parent := woc.wf.Status.Nodes.FindByName(tt.parentName)
			require.NotNil(t, parent)
			assert.Contains(t, parent.Children, hookNode.ID)

			for i := 0; i < 3 && !woc.wf.Status.Phase.Completed(); i++ {
				woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
				woc.operate(ctx)
			}
			require.True(t, woc.wf.Status.Phase.Completed(), "phase %s", woc.wf.Status.Phase)
			_, _, err := wfutil.FormulateRetryWorkflow(ctx, woc.wf, false, "", nil)
			require.NoError(t, err)
		})
	}
}
