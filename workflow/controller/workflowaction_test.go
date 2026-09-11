package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/deprecation"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
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
	// created before the grace period: the workflow informer has had every chance to catch up
	stale := newTestAction("a1", "does-not-exist", wfv1.ActionTypeTerminate, func(a *wfv1.WorkflowAction) {
		a.CreationTimestamp = metav1.Time{Time: time.Now().Add(-2 * actionBindGracePeriod)}
	})
	cancel, wfc := newController(ctx, stale)
	defer cancel()

	require.True(t, wfc.processNextActionItem(ctx))

	a := getTestAction(t, wfc, "a1")
	assert.Equal(t, wfv1.WorkflowActionFailed, a.Status.Phase)
	assert.Equal(t, wfv1.WorkflowActionReasonWorkflowNotFound, a.Status.Reason)
	assert.NotNil(t, a.Status.CompletionTime)
}

func TestWorkflowActionBinderRetriesWhileWorkflowInformerLags(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// a just-created action whose workflow is not (yet) in the cache is requeued, not failed
	fresh := newTestAction("a1", "not-yet-visible", wfv1.ActionTypeTerminate, func(a *wfv1.WorkflowAction) {
		a.CreationTimestamp = metav1.Now()
	})
	cancel, wfc := newController(ctx, fresh)
	defer cancel()

	require.True(t, wfc.processNextActionItem(ctx))

	a := getTestAction(t, wfc, "a1")
	assert.Empty(t, a.Status.Phase)
	assert.Equal(t, 1, wfc.wfActionQueue.NumRequeues("argo/a1"))
}

func TestWorkflowActionBinderUIDMismatch(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	wf.UID = "current-uid"
	action := newTestAction("u1", "suspend", wfv1.ActionTypeTerminate, func(a *wfv1.WorkflowAction) {
		a.Spec.WorkflowRef.UID = "stale-uid"
	})
	cancel, wfc := newController(ctx, wf, action)
	defer cancel()

	assert.True(t, wfc.processNextActionItem(ctx))

	a := getTestAction(t, wfc, "u1")
	assert.Equal(t, wfv1.WorkflowActionFailed, a.Status.Phase)
	assert.Equal(t, wfv1.WorkflowActionReasonWorkflowNotFound, a.Status.Reason)
	assert.Contains(t, a.Status.Message, "uid")
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

	assert.True(t, woc.wf.Status.Suspended)
	// mirrored into spec.suspend so older clients can resume it
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
	assert.False(t, woc.wf.Status.Suspended)
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
	// Suspend and Resume are order-sensitive: the end state tells which was applied last, so a
	// drain that ignored creationTimestamp would be caught. Names sort against the timestamps.
	for name, tt := range map[string]struct {
		first, second wfv1.WorkflowActionType
		suspended     bool
	}{
		"SuspendThenResume": {wfv1.ActionTypeSuspend, wfv1.ActionTypeResume, false},
		"ResumeThenSuspend": {wfv1.ActionTypeResume, wfv1.ActionTypeSuspend, true},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
			older := newTestAction("z-first", "suspend", tt.first, func(a *wfv1.WorkflowAction) {
				a.CreationTimestamp = metav1.Time{Time: time.Now().Add(-2 * time.Hour)}
			})
			newer := newTestAction("a-second", "suspend", tt.second, func(a *wfv1.WorkflowAction) {
				a.CreationTimestamp = metav1.Time{Time: time.Now().Add(-time.Hour)}
			})
			cancel, controller := newController(ctx, wf, newer, older)
			defer cancel()

			woc := newWorkflowOperationCtx(ctx, wf, controller)
			woc.operate(ctx)

			assert.Equal(t, tt.suspended, woc.wf.Status.Suspended)
			for _, name := range []string{"z-first", "a-second"} {
				a := getTestAction(t, controller, name)
				assert.Equal(t, wfv1.WorkflowActionSucceeded, a.Status.Phase, name)
			}
		})
	}
}

func TestActionReconciliationWALReplay(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// The effect was persisted (status.shutdown and the write-ahead record) but the controller
	// crashed before writing the action's status: the pending action must be reported as
	// Succeeded on replay, without being re-applied, and its record kept until it is terminal.
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	wf.Status.Shutdown = wfv1.ShutdownStrategyTerminate
	wf.Status.AppliedActions = []string{"uid-rp1"}
	cancel, controller := newController(ctx, wf, newTestAction("rp1", "suspend", wfv1.ActionTypeTerminate))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	a := getTestAction(t, controller, "rp1")
	assert.Equal(t, wfv1.WorkflowActionSucceeded, a.Status.Phase)
	assert.Equal(t, []string{"uid-rp1"}, woc.wf.Status.AppliedActions)
}

func TestActionReconciliationReplacesActorLabels(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// the actor labels describe the latest action (#14102): an action without identity (created
	// with kubectl) must not leave the previous requester's labels behind
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	wf.Labels["workflows.argoproj.io/action"] = "suspend"
	wf.Labels["workflows.argoproj.io/actor"] = "alice"
	wf.Labels["workflows.argoproj.io/actor-email"] = "alice.at.example.com"
	cancel, controller := newController(ctx, wf, newTestAction("bare", "suspend", wfv1.ActionTypeTerminate))
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.NotContains(t, woc.wf.Labels, "workflows.argoproj.io/action")
	assert.NotContains(t, woc.wf.Labels, "workflows.argoproj.io/actor")
	assert.NotContains(t, woc.wf.Labels, "workflows.argoproj.io/actor-email")
}

func TestResumeMessageIdentity(t *testing.T) {
	a := newTestAction("r", "wf", wfv1.ActionTypeResume, func(a *wfv1.WorkflowAction) {
		a.Labels = map[string]string{
			"workflows.argoproj.io/actor":                    "system.serviceaccount.argo.argo-server",
			"workflows.argoproj.io/actor-preferred-username": "alice",
		}
	})
	// the subject is present even when there is no email claim (client and server auth modes)
	assert.Equal(t, "Resumed by WorkflowAction r (system.serviceaccount.argo.argo-server, alice)", resumeMessage(a))
	assert.Equal(t, "Resumed by WorkflowAction bare", resumeMessage(newTestAction("bare", "wf", wfv1.ActionTypeResume)))
}

func TestSettlePendingActionsWorkflowCompletedAfterBind(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// The binder saw the workflow incomplete and enqueued it, but it completed before the
	// workflow worker got to it: operate() never runs, so processNextItem itself must settle
	// the actions, succeeding one already recorded as applied and failing the rest.
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: raced-wf
  namespace: argo
  labels:
    workflows.argoproj.io/completed: "true"
status:
  phase: Succeeded
  appliedActions:
  - uid-applied
`)
	cancel, wfc := newController(ctx, wf,
		newTestAction("applied", "raced-wf", wfv1.ActionTypeSuspend),
		newTestAction("late", "raced-wf", wfv1.ActionTypeSuspend))
	defer cancel()

	// completed workflows are filtered from the informer handlers, so enqueue as the binder would
	wfc.wfQueue.Add("argo/raced-wf")
	require.True(t, wfc.processNextItem(ctx))

	assert.Equal(t, wfv1.WorkflowActionSucceeded, getTestAction(t, wfc, "applied").Status.Phase)
	late := getTestAction(t, wfc, "late")
	assert.Equal(t, wfv1.WorkflowActionFailed, late.Status.Phase)
	assert.Equal(t, wfv1.WorkflowActionReasonWorkflowCompleted, late.Status.Reason)
}

func TestSettlePendingActionsWorkflowDeleted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	cancel, wfc := newController(ctx, wf, newTestAction("orphan", "suspend", wfv1.ActionTypeStop))
	defer cancel()

	un, err := util.ToUnstructured(wf)
	require.NoError(t, err)
	wfc.settlePendingActions(ctx, un, wfv1.WorkflowActionReasonWorkflowNotFound)

	a := getTestAction(t, wfc, "orphan")
	assert.Equal(t, wfv1.WorkflowActionFailed, a.Status.Phase)
	assert.Equal(t, wfv1.WorkflowActionReasonWorkflowNotFound, a.Status.Reason)
	assert.Contains(t, a.Status.Message, "deleted")
}

func TestPostponedWorkflowDrainsActions(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// my-wf-1 is postponed by the parallelism limit and never reaches operate(); a Suspend
	// requested against it must still be applied (and reported) while it waits.
	cancel, controller := newController(ctx,
		wfv1.MustUnmarshalWorkflow(freshWf),
		wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: queued
  namespace: argo
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`),
		newTestAction("q-suspend", "queued", wfv1.ActionTypeSuspend),
		func(x *WorkflowController) { x.Config.Parallelism = 1 },
	)
	defer cancel()

	assert.True(t, controller.processNextItem(ctx))
	assert.True(t, controller.processNextItem(ctx))

	expectNamespacedWorkflow(ctx, controller, "argo", "fresh", func(wf *wfv1.Workflow) {
		assert.Equal(t, wfv1.WorkflowRunning, wf.Status.Phase)
	})
	expectNamespacedWorkflow(ctx, controller, "argo", "queued", func(wf *wfv1.Workflow) {
		assert.Equal(t, wfv1.WorkflowPending, wf.Status.Phase)
		assert.True(t, wf.Status.Suspended)
		assert.Empty(t, wf.Status.Nodes, "a postponed workflow must not start")
	})
	assert.Equal(t, wfv1.WorkflowActionSucceeded, getTestAction(t, controller, "q-suspend").Status.Phase)
}

func TestPendingTerminateActionBypassesParallelism(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// like spec.shutdown: Terminate (TestParallelism), a pending Terminate action is never
	// postponed by the parallelism limit
	cancel, controller := newController(ctx,
		wfv1.MustUnmarshalWorkflow(freshWf),
		wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: doomed
  namespace: argo
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`),
		newTestAction("d-terminate", "doomed", wfv1.ActionTypeTerminate),
		func(x *WorkflowController) { x.Config.Parallelism = 1 },
	)
	defer cancel()

	assert.True(t, controller.processNextItem(ctx))
	assert.True(t, controller.processNextItem(ctx))

	expectNamespacedWorkflow(ctx, controller, "argo", "doomed", func(wf *wfv1.Workflow) {
		assert.Equal(t, wfv1.WorkflowFailed, wf.Status.Phase)
		assert.Equal(t, wfv1.ShutdownStrategyTerminate, wf.Status.Shutdown)
	})
	assert.Equal(t, wfv1.WorkflowActionSucceeded, getTestAction(t, controller, "d-terminate").Status.Phase)
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
	wf.Spec.Shutdown = wfv1.ShutdownStrategyTerminate
	wf.Status.Shutdown = wfv1.ShutdownStrategyStop
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	assert.Equal(t, wfv1.ShutdownStrategyStop, woc.GetShutdownStrategy())
}

const freshWf = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: fresh
  namespace: argo
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: argoproj/argosay:v2
`

func deprecationCount(t *testing.T) (int64, error) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	attribs := attribute.NewSet(attribute.String("feature", "workflow spec.suspend"))
	return testExporter.GetInt64CounterValue(ctx, telemetry.InstrumentDeprecatedFeature.Name(), &attribs)
}

func TestStartSuspended(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(freshWf)
	wf.Spec.StartSuspended = true
	cancel, controller := newController(ctx, wf)
	defer cancel()
	deprecation.Initialize(controller.metrics.DeprecatedFeature)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.True(t, woc.wf.Status.Suspended)
	// mirrored into spec.suspend so older clients can resume it
	require.NotNil(t, woc.wf.Spec.Suspend)
	assert.True(t, *woc.wf.Spec.Suspend)
	assert.Empty(t, woc.wf.Status.Nodes)
	// a controller-made suspension is not a deprecated spec.suspend use
	_, err := deprecationCount(t)
	assert.Error(t, err)
}

func TestStartSuspendedNotReappliedWhilePending(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// A startSuspended workflow parked Pending (e.g. on a synchronization lock) has no
	// StartedAt; after a Resume cleared the suspension, the next reconcile must not re-apply
	// startSuspended, which is honoured only on the reconcile that leaves the Unknown phase.
	wf := wfv1.MustUnmarshalWorkflow(freshWf)
	wf.Spec.StartSuspended = true
	wf.Status.Phase = wfv1.WorkflowPending
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.False(t, woc.wf.Status.Suspended)
	assert.Nil(t, woc.wf.Spec.Suspend)
}

func TestCreationTimeSpecSuspendNotCountedAsDeprecated(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// spec.suspend set at creation time ("start suspended") remains supported: it is mirrored
	// into status.suspended without counting on the deprecated_feature metric
	wf := wfv1.MustUnmarshalWorkflow(freshWf)
	wf.Spec.Suspend = new(true)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	deprecation.Initialize(controller.metrics.DeprecatedFeature)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.True(t, woc.wf.Status.Suspended)
	assert.Empty(t, woc.wf.Status.Nodes)
	_, err := deprecationCount(t)
	assert.Error(t, err)
}

func TestStartSuspendedIgnoredAfterStart(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	wf.Spec.StartSuspended = true
	cancel, controller := newController(ctx, wf)
	defer cancel()

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// the workflow already started, so flipping startSuspended does nothing
	assert.False(t, woc.wf.Status.Suspended)
	assert.Nil(t, woc.wf.Spec.Suspend)
}

func TestSpecSuspendDeprecationCounted(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// the controller writes spec.suspend and status.suspended together, so a mismatch is a
	// client toggling the deprecated field; the mirror counts it once per toggle
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	wf.Spec.Suspend = new(true)
	cancel, controller := newController(ctx, wf)
	defer cancel()
	deprecation.Initialize(controller.metrics.DeprecatedFeature)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	val, err := deprecationCount(t)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
	assert.True(t, woc.wf.Status.Suspended)

	// mirrored now, so the same suspension episode is not recounted
	woc2 := newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc2.operate(ctx)
	val, err = deprecationCount(t)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
}

func TestOldClientResumeMirrors(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// an old client resumes by clearing spec.suspend; the mirror follows it down so the
	// workflow actually resumes, without counting deprecated use
	wf := wfv1.MustUnmarshalWorkflow(actionTargetWf)
	wf.Status.Suspended = true
	cancel, controller := newController(ctx, wf)
	defer cancel()
	deprecation.Initialize(controller.metrics.DeprecatedFeature)

	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	assert.False(t, woc.wf.Status.Suspended)
	_, err := deprecationCount(t)
	assert.Error(t, err)
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
	assert.True(t, apierr.IsNotFound(err))
}

func TestWorkflowActionGCWaitsForTTL(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	completed := metav1.Now()
	done := newTestAction("keep-me", "gone-wf", wfv1.ActionTypeTerminate, func(a *wfv1.WorkflowAction) {
		a.Status = wfv1.WorkflowActionStatus{Phase: wfv1.WorkflowActionSucceeded, CompletionTime: &completed}
	})
	hour := config.TTL(time.Hour)
	cancel, wfc := newController(ctx, done, func(wfc *WorkflowController) {
		wfc.Config.WorkflowActionTTL = &hour
	})
	defer cancel()

	remaining := wfc.actionTTLRemaining(done)
	assert.Greater(t, remaining, 59*time.Minute)
	assert.LessOrEqual(t, remaining, time.Hour)

	// the GC worker re-schedules an action whose TTL has not expired instead of deleting it
	wfc.wfActionGCQueue.Add("argo/keep-me")
	assert.True(t, wfc.processNextActionGCItem(ctx))
	_, err := wfc.wfclientset.ArgoprojV1alpha1().WorkflowActions("argo").Get(ctx, "keep-me", metav1.GetOptions{})
	require.NoError(t, err)
}
