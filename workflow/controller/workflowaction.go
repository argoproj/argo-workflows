package controller

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v4/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/deprecation"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/retry"
	waitutil "github.com/argoproj/argo-workflows/v4/util/wait"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

// newWorkflowActionInformer returns an informer over WorkflowActions, indexed by target
// workflow. Unlike task results, actions carry no mandatory workflow label (kubectl users
// create them bare), so only the instance ID is filtered and indexing reads the spec ref.
func (wfc *WorkflowController) newWorkflowActionInformer(ctx context.Context) cache.SharedIndexInformer {
	labelSelector := labels.NewSelector().
		Add(wfc.instanceIDReq()).
		String()
	informer := wfc.newFilteredInformer(ctx, "workflow actions", labelSelector,
		// This is a generated function, so we can't change the context.
		//nolint:contextcheck
		func(namespace string, resync time.Duration, tweak func(*metav1.ListOptions)) cache.SharedIndexInformer {
			return wfextvv1alpha1.NewFilteredWorkflowActionInformer(wfc.wfclientset, namespace, resync,
				cache.Indexers{indexes.WorkflowActionIndex: indexes.WorkflowActionIndexFunc}, tweak)
		})
	//nolint:errcheck // the error only happens if the informer was stopped, and it hasn't even started
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(obj any) { wfc.enqueueAction(obj) },
			UpdateFunc: func(_, obj any) { wfc.enqueueAction(obj) },
		})
	return informer
}

func (wfc *WorkflowController) enqueueAction(obj any) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err == nil {
		// Add, not AddRateLimited: the queue's requeue count must only count genuine retries
		wfc.wfActionQueue.Add(key)
	}
}

func (wfc *WorkflowController) runActionWorker(ctx context.Context) {
	for wfc.processNextActionItem(ctx) {
	}
}

// actionBindGracePeriod is how long after its creation a WorkflowAction whose target workflow is
// missing from the workflow informer keeps being retried before it is failed as WorkflowNotFound.
// The workflow informer is a separate watch stream from the action informer and can lag it by
// seconds under load, so an action created immediately after its workflow can be processed before
// the workflow appears in the cache.
const actionBindGracePeriod = 10 * time.Second

// processNextActionItem binds a pending action to its target workflow: actions whose target is
// completed fail immediately, and a target missing from the cache is retried until the grace
// period since the action's creation has elapsed, then failed (no reattempt after that);
// otherwise the target workflow is enqueued and the action is drained inside operate(),
// serialized with reconciliation.
func (wfc *WorkflowController) processNextActionItem(ctx context.Context) bool {
	key, quit := wfc.wfActionQueue.Get()
	if quit {
		return false
	}
	defer wfc.wfActionQueue.Done(key)

	obj, exists, err := wfc.wfActionInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		wfc.wfActionQueue.Forget(key)
		return true
	}
	a, ok := obj.(*wfv1.WorkflowAction)
	if !ok {
		wfc.wfActionQueue.Forget(key)
		return true
	}
	if a.Status.Fulfilled() {
		wfc.wfActionQueue.Forget(key)
		wfc.enqueueActionGC(a)
		return true
	}
	wfKey := indexes.WorkflowIndexValue(a.Namespace, a.Spec.WorkflowRef.Name)
	wfObj, wfExists, err := wfc.wfInformer.GetStore().GetByKey(wfKey)
	if err != nil {
		wfc.wfActionQueue.AddRateLimited(key)
		return true
	}
	log := logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"workflowaction": key, "workflow": wfKey})
	if !wfExists {
		// the workflow informer may lag the action informer, so keep retrying (with backoff)
		// for a grace period before declaring the target genuinely missing
		if time.Since(a.CreationTimestamp.Time) < actionBindGracePeriod {
			wfc.wfActionQueue.AddRateLimited(key)
			return true
		}
		wfc.wfActionQueue.Forget(key)
		log.Info(ctx, "Failing workflow action: target workflow not found")
		wfc.failActionOutcome(ctx, a, wfv1.WorkflowActionReasonWorkflowNotFound,
			fmt.Sprintf("workflow %q not found", wfKey))
		return true
	}
	wfc.wfActionQueue.Forget(key)
	un, ok := wfObj.(*unstructured.Unstructured)
	if !ok {
		return true
	}
	// A pinned UID that no longer matches means the named workflow was recreated: the intended
	// target is gone, so fail rather than act on its replacement.
	if a.Spec.WorkflowRef.UID != "" && un.GetUID() != a.Spec.WorkflowRef.UID {
		log.Info(ctx, "Failing workflow action: target workflow UID mismatch")
		wfc.failActionOutcome(ctx, a, wfv1.WorkflowActionReasonWorkflowNotFound,
			fmt.Sprintf("workflow %q with uid %q not found (current uid %q)", wfKey, a.Spec.WorkflowRef.UID, un.GetUID()))
		return true
	}
	// WAL recovery first: an already-applied action must succeed even if the workflow since completed
	if applied, _, _ := unstructured.NestedStringSlice(un.Object, "status", "appliedActions"); slices.Contains(applied, string(a.UID)) {
		wfc.recordActionOutcome(ctx, a, wfv1.WorkflowActionSucceeded, "", "")
		return true
	}
	if un.GetLabels()[common.LabelKeyCompleted] == "true" {
		log.Info(ctx, "Failing workflow action: target workflow is completed")
		wfc.failActionOutcome(ctx, a, wfv1.WorkflowActionReasonWorkflowCompleted, completedWorkflowMessage(a))
		return true
	}
	// Add, not AddRateLimited: the workflow queue's limiter is a fixed requeue interval, which
	// would delay every action by that interval; the target is validated, reconcile it now.
	wfc.wfQueue.Add(wfKey)
	return true
}

// settlePendingActions records a terminal outcome for every pending action targeting a workflow
// that can no longer be reconciled (it completed, or was deleted, after the binder enqueued it),
// so they do not sit Pending until the next informer resync. An action whose UID is in the
// workflow's write-ahead record already took effect and succeeds; the rest fail with reason.
func (wfc *WorkflowController) settlePendingActions(ctx context.Context, un *unstructured.Unstructured, reason string) {
	applied, _, _ := unstructured.NestedStringSlice(un.Object, "status", "appliedActions")
	for _, a := range wfc.pendingActionsFor(un.GetNamespace(), un.GetName()) {
		if slices.Contains(applied, string(a.UID)) {
			wfc.recordActionOutcome(ctx, a, wfv1.WorkflowActionSucceeded, "", "")
			continue
		}
		message := completedWorkflowMessage(a)
		if reason == wfv1.WorkflowActionReasonWorkflowNotFound {
			message = fmt.Sprintf("workflow %q was deleted", un.GetName())
		}
		wfc.failActionOutcome(ctx, a, reason, message)
	}
}

func completedWorkflowMessage(a *wfv1.WorkflowAction) string {
	return fmt.Sprintf("cannot perform %s on completed workflow %q", a.Spec.Action, a.Spec.WorkflowRef.Name)
}

func (wfc *WorkflowController) failActionOutcome(ctx context.Context, a *wfv1.WorkflowAction, reason, message string) {
	wfc.recordActionOutcome(ctx, a, wfv1.WorkflowActionFailed, reason, message)
}

// recordActionOutcome writes a terminal phase to the action's status, retrying conflicts, and
// schedules the action for TTL deletion. It is a no-op for actions that are already terminal.
func (wfc *WorkflowController) recordActionOutcome(ctx context.Context, a *wfv1.WorkflowAction, phase wfv1.WorkflowActionPhase, reason, message string) {
	key, _ := cache.MetaNamespaceKeyFunc(a)
	log := logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"workflowaction": key, "phase": phase, "reason": reason})
	actionIf := wfc.wfclientset.ArgoprojV1alpha1().WorkflowActions(a.Namespace)
	err := waitutil.Backoff(retry.DefaultRetry(ctx), func() (bool, error) {
		latest, getErr := actionIf.Get(ctx, a.Name, metav1.GetOptions{})
		if getErr != nil {
			if apierr.IsNotFound(getErr) {
				return true, nil
			}
			return !errorsutil.IsTransientErr(ctx, getErr), getErr
		}
		if latest.Status.Fulfilled() {
			wfc.enqueueActionGC(latest)
			return true, nil
		}
		latest.Status = wfv1.WorkflowActionStatus{
			Phase:          phase,
			Reason:         reason,
			Message:        message,
			CompletionTime: &metav1.Time{Time: time.Now().UTC()},
		}
		updated, updateErr := actionIf.UpdateStatus(ctx, latest, metav1.UpdateOptions{})
		if updateErr != nil {
			if apierr.IsConflict(updateErr) {
				return false, nil
			}
			return !errorsutil.IsTransientErr(ctx, updateErr), updateErr
		}
		wfc.metrics.WorkflowActionProcessed(ctx, string(a.Spec.Action), string(phase), a.Namespace)
		wfc.recordActionEvent(ctx, updated)
		wfc.enqueueActionGC(updated)
		return true, nil
	})
	if err != nil {
		log.WithError(err).Error(ctx, "Failed to record workflow action outcome")
	}
}

// recordActionEvent emits a Kubernetes Event on the action itself, so `kubectl describe wfa`
// shows why it failed without needing access to the controller logs or the target workflow.
func (wfc *WorkflowController) recordActionEvent(ctx context.Context, a *wfv1.WorkflowAction) {
	recorder := wfc.eventRecorderManager.Get(ctx, a.Namespace)
	if a.Status.Phase == wfv1.WorkflowActionFailed {
		recorder.Event(a, apiv1.EventTypeWarning, "WorkflowActionFailed",
			fmt.Sprintf("%s of workflow %s failed: %s", a.Spec.Action, a.Spec.WorkflowRef.Name, a.Status.Message))
		return
	}
	recorder.Event(a, apiv1.EventTypeNormal, "WorkflowActionSucceeded",
		fmt.Sprintf("%s of workflow %s applied", a.Spec.Action, a.Spec.WorkflowRef.Name))
}

// pendingActionsFor returns the pending actions targeting a workflow, oldest first.
func (wfc *WorkflowController) pendingActionsFor(namespace, name string) []*wfv1.WorkflowAction {
	objs, err := wfc.wfActionInformer.GetIndexer().ByIndex(indexes.WorkflowActionIndex, indexes.WorkflowIndexValue(namespace, name))
	if err != nil {
		return nil
	}
	var actions []*wfv1.WorkflowAction
	for _, obj := range objs {
		if a, ok := obj.(*wfv1.WorkflowAction); ok && !a.Status.Fulfilled() {
			actions = append(actions, a)
		}
	}
	sort.Slice(actions, func(i, j int) bool {
		ti, tj := actions[i].CreationTimestamp, actions[j].CreationTimestamp
		if ti.Equal(&tj) {
			return actions[i].Name < actions[j].Name
		}
		return ti.Before(&tj)
	})
	return actions
}

// pendingActionsForKey is pendingActionsFor keyed by "namespace/name".
func (wfc *WorkflowController) pendingActionsForKey(key string) []*wfv1.WorkflowAction {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return nil
	}
	return wfc.pendingActionsFor(namespace, name)
}

// hasPendingTerminateAction reports whether a pending Terminate action targets the workflow with
// the given "namespace/name" key. It backs the throttler admission check: a Terminate must never
// be postponed by the parallelism limit.
func (wfc *WorkflowController) hasPendingTerminateAction(key string) bool {
	return hasPendingTerminate(wfc.pendingActionsForKey(key))
}

func hasPendingTerminate(actions []*wfv1.WorkflowAction) bool {
	return slices.ContainsFunc(actions, func(a *wfv1.WorkflowAction) bool { return a.Spec.Action == wfv1.ActionTypeTerminate })
}

// actionResult is a drained action outcome awaiting persistence of its workflow effect.
type actionResult struct {
	action  *wfv1.WorkflowAction
	phase   wfv1.WorkflowActionPhase
	reason  string
	message string
}

// actionReconciliation drains the pending WorkflowActions targeting this workflow, applying each
// in creationTimestamp order to the in-memory workflow. Effects are persisted through the normal
// persistUpdates path, with the applied action UIDs recorded in status.appliedActions in the same
// write (a write-ahead record); the actions' own statuses are written afterwards by
// reportActionOutcomes. Validation failures touch nothing and are reported immediately.
func (woc *wfOperationCtx) actionReconciliation(ctx context.Context) {
	actions := woc.controller.pendingActionsFor(woc.wf.Namespace, woc.wf.Name)
	if len(actions) == 0 && len(woc.wf.Status.AppliedActions) == 0 {
		return
	}
	woc.pruneAppliedActions(actions)
	for _, a := range actions {
		if slices.Contains(woc.wf.Status.AppliedActions, string(a.UID)) {
			// crash recovery: the effect is already persisted, only the action status is missing
			woc.pendingActionResults = append(woc.pendingActionResults, actionResult{action: a, phase: wfv1.WorkflowActionSucceeded})
			continue
		}
		if woc.wf.Status.Fulfilled() {
			// the binder normally fails these before we get here; defense for races
			woc.controller.failActionOutcome(ctx, a, wfv1.WorkflowActionReasonWorkflowCompleted, completedWorkflowMessage(a))
			continue
		}
		changed, err := woc.applyAction(ctx, a)
		if err != nil {
			woc.controller.failActionOutcome(ctx, a, wfv1.WorkflowActionReasonInvalidAction, err.Error())
			continue
		}
		if changed {
			woc.wf.Status.AppliedActions = append(woc.wf.Status.AppliedActions, string(a.UID))
			woc.updated = true
		}
		woc.pendingActionResults = append(woc.pendingActionResults, actionResult{action: a, phase: wfv1.WorkflowActionSucceeded})
	}
}

// applyAction applies one action to the in-memory workflow, returning whether it changed anything.
// An error means the action was invalid and nothing was changed.
func (woc *wfOperationCtx) applyAction(ctx context.Context, a *wfv1.WorkflowAction) (bool, error) {
	switch a.Spec.Action {
	case wfv1.ActionTypeTerminate:
		return woc.applyShutdown(a, wfv1.ShutdownStrategyTerminate), nil
	case wfv1.ActionTypeStop:
		if a.Spec.Stop != nil && a.Spec.Stop.NodeFieldSelector != "" {
			return woc.applyNodeSet(ctx, a, a.Spec.Stop.NodeFieldSelector, util.SetOperationValues{Phase: wfv1.NodeFailed, Message: a.Spec.Stop.Message})
		}
		return woc.applyShutdown(a, wfv1.ShutdownStrategyStop), nil
	case wfv1.ActionTypeSuspend:
		if woc.ShouldSuspend() {
			return false, nil
		}
		woc.setSuspended()
		woc.copyActorLabels(a)
		return true, nil
	case wfv1.ActionTypeResume:
		if a.Spec.Resume != nil && a.Spec.Resume.NodeFieldSelector != "" {
			return woc.applyNodeSet(ctx, a, a.Spec.Resume.NodeFieldSelector, util.SetOperationValues{Phase: wfv1.NodeSucceeded, Message: resumeMessage(a), OutputParameters: a.Spec.Resume.OutputParameters})
		}
		// ApplyResume mutates the workflow incrementally and can error partway, so apply it to a
		// copy and commit only on success — a failed action must leave no partial effect.
		wfCopy := woc.wf.DeepCopy()
		changed, err := util.ApplyResume(ctx, wfCopy, resumeMessage(a))
		if err != nil {
			return false, err
		}
		woc.commitWorkflowCopy(wfCopy)
		woc.execWf.Spec.Suspend = nil
		if changed {
			woc.copyActorLabels(a)
		}
		return changed, nil
	default:
		return false, fmt.Errorf("unsupported action %q", a.Spec.Action)
	}
}

func (woc *wfOperationCtx) applyShutdown(a *wfv1.WorkflowAction, strategy wfv1.ShutdownStrategy) bool {
	if woc.wf.EffectiveShutdown() == strategy {
		return false
	}
	woc.wf.Status.Shutdown = strategy
	woc.copyActorLabels(a)
	return true
}

func (woc *wfOperationCtx) applyNodeSet(ctx context.Context, a *wfv1.WorkflowAction, nodeFieldSelector string, values util.SetOperationValues) (bool, error) {
	// ApplySuspendedNodeSetOperation writes node phase, message and global parameters and can then
	// fail on an unknown output parameter, so apply it to a copy and commit only on success.
	wfCopy := woc.wf.DeepCopy()
	changed, err := util.ApplySuspendedNodeSetOperation(ctx, wfCopy, nodeFieldSelector, values)
	if err != nil {
		return false, err
	}
	woc.commitWorkflowCopy(wfCopy)
	if changed {
		woc.copyActorLabels(a)
	}
	return changed, nil
}

// commitWorkflowCopy replaces the in-memory workflow with a successfully mutated copy. Unless a
// stored spec is in use (workflowTemplateRef), execWf aliases wf, so the alias must follow the
// swap or the rest of the reconcile would read a stale spec through execWf.
func (woc *wfOperationCtx) commitWorkflowCopy(wfCopy *wfv1.Workflow) {
	if woc.execWf == woc.wf {
		woc.execWf = wfCopy
	}
	woc.wf = wfCopy
}

// suspendReconciliation keeps status.suspended, the controller-owned suspension state, mirrored
// from the client-owned spec.suspend. The controller always writes the two together (Suspend
// action, startSuspended), so a mismatch means a client changed spec.suspend: a toggle into
// suspension on a started workflow is counted on the deprecated_feature metric (setting
// spec.suspend at creation time, "start suspended", is still supported), and a cleared
// spec.suspend (an old client's resume) unsuspends the workflow, preserving compatibility.
// spec.startSuspended is honoured exactly once, on the reconcile that first moves the workflow
// out of the Unknown phase: every path that leaves Unknown (operate, or the parallelism
// postponement in processNextItem) runs this first. StartedAt cannot serve as the marker
// because a workflow parked Pending on a synchronization lock has none, and a Resume in that
// state must not be undone on the next reconcile.
func (woc *wfOperationCtx) suspendReconciliation(ctx context.Context) {
	firstReconcile := woc.wf.Status.Phase == wfv1.WorkflowUnknown
	if firstReconcile && woc.execWf.Spec.StartSuspended && !woc.ShouldSuspend() {
		woc.setSuspended()
		return
	}
	specSuspended := woc.execWf.Spec.SuspendRequested()
	if specSuspended == woc.wf.Status.Suspended {
		return
	}
	if specSuspended && !firstReconcile {
		// a client suspended a started workflow by setting the deprecated spec.suspend
		deprecation.Record(ctx, deprecation.WorkflowSpecSuspend)
	}
	woc.wf.Status.Suspended = specSuspended
	woc.updated = true
}

// setSuspended marks the workflow suspended. status.suspended is the controller-owned state;
// it is mirrored into spec.suspend so that older clients can still resume the workflow.
func (woc *wfOperationCtx) setSuspended() {
	woc.wf.Spec.Suspend = new(true) //nolint:forbidigo // not-woc-misuse
	woc.execWf.Spec.Suspend = new(true)
	woc.wf.Status.Suspended = true
	woc.updated = true
}

// copyActorLabels carries the requester's identity from the action onto the workflow, preserving
// the audit labels the server used to patch directly (#14102). As with creator.LabelActor, the
// labels describe the latest action: any actor labels from an earlier action are removed first,
// so an action without identity (created with kubectl) does not keep the previous requester's.
func (woc *wfOperationCtx) copyActorLabels(a *wfv1.WorkflowAction) {
	for _, k := range []string{common.LabelKeyAction, common.LabelKeyActor, common.LabelKeyActorEmail, common.LabelKeyActorPreferredUsername} {
		if _, had := woc.wf.Labels[k]; had {
			delete(woc.wf.Labels, k)
			woc.updated = true
		}
		if v, ok := a.Labels[k]; ok {
			if woc.wf.Labels == nil {
				woc.wf.Labels = map[string]string{}
			}
			woc.wf.Labels[k] = v
			woc.updated = true
		}
	}
}

// resumeMessage is recorded on resumed nodes for auditing (#11763): it names the action and, when
// the action carries them, the requester's subject, preferred username and email.
func resumeMessage(a *wfv1.WorkflowAction) string {
	msg := "Resumed by WorkflowAction " + a.Name
	var who []string
	for _, k := range []string{common.LabelKeyActor, common.LabelKeyActorPreferredUsername, common.LabelKeyActorEmail} {
		if v := a.Labels[k]; v != "" {
			who = append(who, v)
		}
	}
	if len(who) > 0 {
		msg += " (" + strings.Join(who, ", ") + ")"
	}
	return msg
}

// pruneAppliedActions drops write-ahead entries whose action is no longer pending (it has a
// recorded terminal status, or no longer exists), keeping status.appliedActions bounded.
func (woc *wfOperationCtx) pruneAppliedActions(pending []*wfv1.WorkflowAction) {
	if len(woc.wf.Status.AppliedActions) == 0 {
		return
	}
	kept := slices.DeleteFunc(slices.Clone(woc.wf.Status.AppliedActions), func(uid string) bool {
		return !slices.ContainsFunc(pending, func(a *wfv1.WorkflowAction) bool { return string(a.UID) == uid })
	})
	if len(kept) != len(woc.wf.Status.AppliedActions) {
		woc.wf.Status.AppliedActions = kept
		woc.updated = true
	}
}

// reportActionOutcomes records the status of actions whose effects have been persisted. It is
// called from persistUpdates after a successful write (and on the no-op path, for actions that
// changed nothing).
func (woc *wfOperationCtx) reportActionOutcomes(ctx context.Context) {
	for _, r := range woc.pendingActionResults {
		woc.controller.recordActionOutcome(ctx, r.action, r.phase, r.reason, r.message)
		woc.eventRecorder.Event(woc.wf, apiv1.EventTypeNormal, "WorkflowActionApplied",
			fmt.Sprintf("%s requested by WorkflowAction %s", r.action.Spec.Action, r.action.Name))
	}
	woc.pendingActionResults = nil
}

// enqueueActionGC schedules a terminal action for deletion once its TTL expires.
func (wfc *WorkflowController) enqueueActionGC(a *wfv1.WorkflowAction) {
	if !a.Status.Fulfilled() {
		return
	}
	key, err := cache.MetaNamespaceKeyFunc(a)
	if err != nil {
		return
	}
	wfc.wfActionGCQueue.AddAfter(key, wfc.actionTTLRemaining(a))
}

// actionTTLRemaining is how long until a terminal action's TTL expires (zero if it already has).
func (wfc *WorkflowController) actionTTLRemaining(a *wfv1.WorkflowAction) time.Duration {
	remaining := wfc.Config.GetWorkflowActionTTL()
	if a.Status.CompletionTime != nil {
		remaining -= time.Since(a.Status.CompletionTime.Time)
	}
	return max(remaining, 0)
}

func (wfc *WorkflowController) runActionGCWorker(ctx context.Context) {
	for wfc.processNextActionGCItem(ctx) {
	}
}

func (wfc *WorkflowController) processNextActionGCItem(ctx context.Context) bool {
	key, quit := wfc.wfActionGCQueue.Get()
	if quit {
		return false
	}
	defer wfc.wfActionGCQueue.Done(key)

	obj, exists, err := wfc.wfActionInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return true
	}
	a, ok := obj.(*wfv1.WorkflowAction)
	if !ok || !a.Status.Fulfilled() {
		return true
	}
	if remaining := wfc.actionTTLRemaining(a); remaining > 0 {
		wfc.wfActionGCQueue.AddAfter(key, remaining)
		return true
	}
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return true
	}
	log := logging.RequireLoggerFromContext(ctx).WithField("workflowaction", key)
	log.Info(ctx, "Deleting workflow action due to TTL")
	err = wfc.wfclientset.ArgoprojV1alpha1().WorkflowActions(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierr.IsNotFound(err) {
		log.WithError(err).Error(ctx, "Failed to delete workflow action")
	}
	return true
}
