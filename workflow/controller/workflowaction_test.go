package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
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
