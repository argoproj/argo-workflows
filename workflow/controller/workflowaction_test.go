package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/deprecation"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

func newTestAction(name, wfName string, action wfv1.WorkflowActionType, opts ...func(*wfv1.WorkflowAction)) *wfv1.WorkflowAction {
	a := &wfv1.WorkflowAction{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "argo",
			UID:               types.UID("uid-" + name),
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-time.Minute)},
		},
		Spec: wfv1.WorkflowActionSpec{
			WorkflowRef: wfv1.WorkflowActionRef{Name: wfName},
			Action:      action,
		},
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func getTestAction(t *testing.T, wfc *WorkflowController, name string) *wfv1.WorkflowAction {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	a, err := wfc.wfclientset.ArgoprojV1alpha1().WorkflowActions("argo").Get(ctx, name, metav1.GetOptions{})
	require.NoError(t, err)
	return a
}

func TestWorkflowActionBinderWorkflowNotFound(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, wfc := newController(ctx, newTestAction("a1", "does-not-exist", wfv1.ActionTypeTerminate))
	defer cancel()

	assert.True(t, wfc.processNextActionItem(ctx))

	a := getTestAction(t, wfc, "a1")
	assert.Equal(t, wfv1.WorkflowActionFailed, a.Status.Phase)
	assert.Equal(t, wfv1.WorkflowActionReasonWorkflowNotFound, a.Status.Reason)
	assert.NotNil(t, a.Status.CompletionTime)
}

func TestWorkflowActionBinderWorkflowCompleted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: done-wf
  namespace: argo
  labels:
    workflows.argoproj.io/completed: "true"
status:
  phase: Succeeded
`)
	cancel, wfc := newController(ctx, wf, newTestAction("a2", "done-wf", wfv1.ActionTypeResume))
	defer cancel()

	assert.True(t, wfc.processNextActionItem(ctx))

	a := getTestAction(t, wfc, "a2")
	assert.Equal(t, wfv1.WorkflowActionFailed, a.Status.Phase)
	assert.Equal(t, wfv1.WorkflowActionReasonWorkflowCompleted, a.Status.Reason)
}

func TestWorkflowActionBinderWALRecovery(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// The workflow completed, but this action's UID is recorded as applied: the effect was
	// persisted before a crash, so the action must report success, not WorkflowCompleted.
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: crashed-wf
  namespace: argo
  labels:
    workflows.argoproj.io/completed: "true"
status:
  phase: Failed
  appliedActions:
  - uid-a3
`)
	cancel, wfc := newController(ctx, wf, newTestAction("a3", "crashed-wf", wfv1.ActionTypeTerminate))
	defer cancel()

	assert.True(t, wfc.processNextActionItem(ctx))

	a := getTestAction(t, wfc, "a3")
	assert.Equal(t, wfv1.WorkflowActionSucceeded, a.Status.Phase)
}

func TestPendingActionsForOrdering(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	older := newTestAction("b-older", "my-wf", wfv1.ActionTypeSuspend, func(a *wfv1.WorkflowAction) {
		a.CreationTimestamp = metav1.Time{Time: time.Now().Add(-2 * time.Hour)}
	})
	newer := newTestAction("a-newer", "my-wf", wfv1.ActionTypeTerminate, func(a *wfv1.WorkflowAction) {
		a.CreationTimestamp = metav1.Time{Time: time.Now().Add(-time.Hour)}
	})
	fulfilled := newTestAction("c-done", "my-wf", wfv1.ActionTypeStop, func(a *wfv1.WorkflowAction) {
		a.Status = wfv1.WorkflowActionStatus{Phase: wfv1.WorkflowActionSucceeded}
	})
	cancel, wfc := newController(ctx, newer, older, fulfilled)
	defer cancel()

	actions := wfc.pendingActionsFor("argo", "my-wf")
	require.Len(t, actions, 2)
	assert.Equal(t, "b-older", actions[0].Name)
	assert.Equal(t, "a-newer", actions[1].Name)

	assert.True(t, wfc.hasPendingTerminateAction("argo/my-wf"))
	assert.False(t, wfc.hasPendingTerminateAction("argo/other-wf"))
}

// actionTargetWf is a valid running workflow with one active suspend node ("approve"),
// adapted from workflow/util's suspend fixture. Tests may prepend modifications via
// MustUnmarshalWorkflow and direct field edits.
var actionTargetWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: suspend
  namespace: argo
  labels:
    workflows.argoproj.io/phase: Running
spec:
  entrypoint: suspend
  templates:
  - name: suspend
    steps:
    - - name: approve
        template: approve
  - name: approve
    suspend: {}
status:
  phase: Running
  startedAt: "2020-04-10T15:21:23Z"
  finishedAt: null
  nodes:
    suspend:
      children:
      - suspend-1186457231
      displayName: suspend
      finishedAt: null
      id: suspend
      name: suspend
      phase: Running
      startedAt: "2020-04-10T15:21:23Z"
      templateName: suspend
      templateScope: local/suspend
      type: Steps
    suspend-1186457231:
      boundaryID: suspend
      children:
      - suspend-3422888088
      displayName: '[0]'
      finishedAt: null
      id: suspend-1186457231
      name: suspend[0]
      phase: Running
      startedAt: "2020-04-10T15:21:23Z"
      templateScope: local/suspend
      type: StepGroup
    suspend-3422888088:
      boundaryID: suspend
      displayName: approve
      finishedAt: null
      id: suspend-3422888088
      name: suspend[0].approve
      phase: Running
      startedAt: "2020-04-10T15:21:23Z"
      templateName: approve
      templateScope: local/suspend
      type: Suspend
`

func TestActionReconciliationTerminate(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	action := newTestAction("t1", "suspend", wfv1.ActionTypeTerminate, func(a *wfv1.WorkflowAction) {
		a.Labels = map[string]string{"workflows.argoproj.io/action": "terminate"}
	})
	cancel, controller := newController(ctx, wf, action)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.ShutdownStrategyTerminate, woc.wf.Status.Shutdown)
	assert.Contains(t, woc.wf.Status.AppliedActions, "uid-t1")
	assert.Equal(t, "terminate", woc.wf.Labels["workflows.argoproj.io/action"])
	a := getTestAction(t, controller, "t1")
	assert.Equal(t, wfv1.WorkflowActionSucceeded, a.Status.Phase)
}

func TestActionReconciliationSuspend(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	cancel, controller := newController(ctx, wf, newTestAction("s1", "suspend", wfv1.ActionTypeSuspend))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	require.NotNil(t, woc.wf.Spec.Suspend)
	assert.True(t, *woc.wf.Spec.Suspend)
	a := getTestAction(t, controller, "s1")
	assert.Equal(t, wfv1.WorkflowActionSucceeded, a.Status.Phase)
}

func TestActionReconciliationResume(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	wf.Spec.Suspend = new(true)
	cancel, controller := newController(ctx, wf, newTestAction("r1", "suspend", wfv1.ActionTypeResume))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Nil(t, woc.wf.Spec.Suspend)
	assert.Equal(t, wfv1.NodeSucceeded, woc.wf.Status.Nodes.FindByDisplayName("approve").Phase)
	a := getTestAction(t, controller, "r1")
	assert.Equal(t, wfv1.WorkflowActionSucceeded, a.Status.Phase)
}

func TestActionReconciliationStopSelector(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	action := newTestAction("st1", "suspend", wfv1.ActionTypeStop, func(a *wfv1.WorkflowAction) {
		a.Spec.Stop = &wfv1.StopAction{Message: "stop it", NodeFieldSelector: "displayName=approve"}
	})
	cancel, controller := newController(ctx, wf, action)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	node := woc.wf.Status.Nodes.FindByDisplayName("approve")
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
	assert.Equal(t, "stop it", node.Message)
	assert.Equal(t, wfv1.ShutdownStrategyNone, woc.wf.Status.Shutdown)
	a := getTestAction(t, controller, "st1")
	assert.Equal(t, wfv1.WorkflowActionSucceeded, a.Status.Phase)
}

func TestActionReconciliationResumeSelectorNoMatch(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	action := newTestAction("r2", "suspend", wfv1.ActionTypeResume, func(a *wfv1.WorkflowAction) {
		a.Spec.Resume = &wfv1.ResumeAction{NodeFieldSelector: "displayName=nonexistent"}
	})
	cancel, controller := newController(ctx, wf, action)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Empty(t, woc.wf.Status.AppliedActions)
	a := getTestAction(t, controller, "r2")
	assert.Equal(t, wfv1.WorkflowActionFailed, a.Status.Phase)
	assert.Equal(t, wfv1.WorkflowActionReasonInvalidAction, a.Status.Reason)
	assert.Contains(t, a.Status.Message, "no suspend nodes matching")
}

func TestActionReconciliationOrdering(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	suspend := newTestAction("o-suspend", "suspend", wfv1.ActionTypeSuspend, func(a *wfv1.WorkflowAction) {
		a.CreationTimestamp = metav1.Time{Time: time.Now().Add(-2 * time.Hour)}
	})
	terminate := newTestAction("o-terminate", "suspend", wfv1.ActionTypeTerminate, func(a *wfv1.WorkflowAction) {
		a.CreationTimestamp = metav1.Time{Time: time.Now().Add(-time.Hour)}
	})
	cancel, controller := newController(ctx, wf, suspend, terminate)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Equal(t, wfv1.ShutdownStrategyTerminate, woc.wf.Status.Shutdown)
	require.NotNil(t, woc.wf.Spec.Suspend)
	assert.True(t, *woc.wf.Spec.Suspend)
	for _, name := range []string{"o-suspend", "o-terminate"} {
		a := getTestAction(t, controller, name)
		assert.Equal(t, wfv1.WorkflowActionSucceeded, a.Status.Phase, name)
	}
}

func TestAppliedActionsPrune(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	wf.Status.AppliedActions = []string{"uid-p1"}
	done := newTestAction("p1", "suspend", wfv1.ActionTypeSuspend, func(a *wfv1.WorkflowAction) {
		a.Status = wfv1.WorkflowActionStatus{Phase: wfv1.WorkflowActionSucceeded}
	})
	cancel, controller := newController(ctx, wf, done)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.Empty(t, woc.wf.Status.AppliedActions)
}

func TestGetShutdownStrategyPrefersStatus(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	wf.Status.Shutdown = wfv1.ShutdownStrategyStop
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.Equal(t, wfv1.ShutdownStrategyStop, woc.GetShutdownStrategy())
}

func TestSpecShutdownDeprecationMetric(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	wf.Spec.Shutdown = wfv1.ShutdownStrategyStop
	cancel, controller := newController(ctx, wf)
	defer cancel()
	deprecation.Initialize(controller.metrics.DeprecatedFeature)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("feature", "workflow spec.shutdown"))
	val, err := testExporter.GetInt64CounterValue(ctx, telemetry.InstrumentDeprecatedFeature.Name(), &attribs)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, val, int64(1))
}

func TestActionShutdownNotCountedAsDeprecated(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	cancel, controller := newController(ctx, wf, newTestAction("d1", "suspend", wfv1.ActionTypeTerminate))
	defer cancel()
	deprecation.Initialize(controller.metrics.DeprecatedFeature)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// shutdown arrived via a WorkflowAction (status.shutdown), so the deprecated
	// spec.shutdown path must not be counted
	attribs := attribute.NewSet(attribute.String("feature", "workflow spec.shutdown"))
	_, err := testExporter.GetInt64CounterValue(ctx, telemetry.InstrumentDeprecatedFeature.Name(), &attribs)
	assert.Error(t, err)
}

func TestWorkflowActionGC(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	completed := metav1.Time{Time: time.Now().Add(-time.Minute)}
	done := newTestAction("gc-me", "gone-wf", wfv1.ActionTypeTerminate, func(a *wfv1.WorkflowAction) {
		a.Status = wfv1.WorkflowActionStatus{Phase: wfv1.WorkflowActionSucceeded, CompletionTime: &completed}
	})
	zero := config.TTL(0)
	cancel, wfc := newController(ctx, done, func(wfc *WorkflowController) {
		wfc.Config.WorkflowActionTTL = &zero
	})
	defer cancel()

	// The binder routes terminal actions to the GC queue.
	assert.True(t, wfc.processNextActionItem(ctx))
	// TTL 0 and completion in the past: eligible immediately.
	assert.True(t, wfc.processNextActionGCItem(ctx))

	_, err := wfc.wfclientset.ArgoprojV1alpha1().WorkflowActions("argo").Get(ctx, "gc-me", metav1.GetOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
