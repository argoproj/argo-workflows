package controller

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v4/pkg/client/informers/externalversions/workflow/v1alpha1"
	errorsutil "github.com/argoproj/argo-workflows/v4/util/errors"
	informerutil "github.com/argoproj/argo-workflows/v4/util/informer"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/retry"
	waitutil "github.com/argoproj/argo-workflows/v4/util/wait"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
)

// newWorkflowActionInformer returns an informer over WorkflowActions, indexed by target
// workflow. Unlike task results, actions carry no mandatory workflow label (kubectl users
// create them bare), so only the instance ID is filtered and indexing reads the spec ref.
func (wfc *WorkflowController) newWorkflowActionInformer(ctx context.Context) cache.SharedIndexInformer {
	log := logging.RequireLoggerFromContext(ctx)
	labelSelector := labels.NewSelector().
		Add(wfc.instanceIDReq()).
		String()
	log.WithField("labelSelector", labelSelector).
		Info(ctx, "Watching workflow actions")

	// This is a generated function, so we can't change the context.
	//nolint:contextcheck
	informer := wfextvv1alpha1.NewFilteredWorkflowActionInformer(
		wfc.wfclientset,
		wfc.GetManagedNamespace(),
		20*time.Minute,
		cache.Indexers{
			indexes.WorkflowActionIndex: indexes.WorkflowActionIndexFunc,
		},
		func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector
			// `ResourceVersion=0` does not honor the `limit` in API calls, which results in making significant List calls
			// without `limit`. For details, see https://github.com/argoproj/argo-workflows/pull/11343
			// Check if ResourceVersion is "0" and reset it to empty string to avoid missing watch event.
			if options.ResourceVersion == "0" {
				options.ResourceVersion = ""
			}
		},
	)
	//nolint:errcheck // the error only happens if the informer was already started, and it hasn't been
	informer.SetTransform(informerutil.StripManagedFields)
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
		wfc.wfActionQueue.AddRateLimited(key)
	}
}

func (wfc *WorkflowController) runActionWorker(ctx context.Context) {
	for wfc.processNextActionItem(ctx) {
	}
}

// processNextActionItem binds a pending action to its target workflow: actions whose target is
// missing or completed fail immediately (no reattempt); otherwise the target workflow is enqueued
// and the action is drained inside operate(), serialized with reconciliation.
func (wfc *WorkflowController) processNextActionItem(ctx context.Context) bool {
	key, quit := wfc.wfActionQueue.Get()
	if quit {
		return false
	}
	defer wfc.wfActionQueue.Done(key)

	obj, exists, err := wfc.wfActionInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return true
	}
	a, ok := obj.(*wfv1.WorkflowAction)
	if !ok {
		return true
	}
	if a.Status.Fulfilled() {
		wfc.enqueueActionGC(a)
		return true
	}
	wfKey := a.Namespace + "/" + a.Spec.WorkflowRef.Name
	wfObj, wfExists, err := wfc.wfInformer.GetStore().GetByKey(wfKey)
	if err != nil {
		return true
	}
	log := logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"workflowaction": key, "workflow": wfKey})
	if !wfExists {
		log.Info(ctx, "Failing workflow action: target workflow not found")
		wfc.failActionOutcome(ctx, a, wfv1.WorkflowActionReasonWorkflowNotFound,
			fmt.Sprintf("workflow %q not found", wfKey))
		return true
	}
	un, ok := wfObj.(*unstructured.Unstructured)
	if !ok {
		return true
	}
	// WAL recovery first: an already-applied action must succeed even if the workflow since completed
	if applied, _, _ := unstructured.NestedStringSlice(un.Object, "status", "appliedActions"); slices.Contains(applied, string(a.UID)) {
		wfc.recordActionOutcome(ctx, a, wfv1.WorkflowActionSucceeded, "", "")
		return true
	}
	if un.GetLabels()[common.LabelKeyCompleted] == "true" {
		log.Info(ctx, "Failing workflow action: target workflow is completed")
		wfc.failActionOutcome(ctx, a, wfv1.WorkflowActionReasonWorkflowCompleted,
			fmt.Sprintf("cannot perform %s on completed workflow %q", a.Spec.Action, wfKey))
		return true
	}
	wfc.wfQueue.AddRateLimited(wfKey)
	return true
}

func (wfc *WorkflowController) failActionOutcome(ctx context.Context, a *wfv1.WorkflowAction, reason, message string) {
	wfc.recordActionOutcome(ctx, a, wfv1.WorkflowActionFailed, reason, message)
}

// recordActionOutcome writes a terminal phase to the action's status, retrying conflicts, and
// schedules the action for TTL deletion. It is a no-op for actions that are already terminal.
func (wfc *WorkflowController) recordActionOutcome(ctx context.Context, a *wfv1.WorkflowAction, phase wfv1.WorkflowActionPhase, reason, message string) {
	log := logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{"workflowaction": a.Namespace + "/" + a.Name, "phase": phase, "reason": reason})
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
		wfc.enqueueActionGC(updated)
		return true, nil
	})
	if err != nil {
		log.WithError(err).Error(ctx, "Failed to record workflow action outcome")
	}
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

// hasPendingTerminateAction reports whether a pending Terminate action targets the workflow with
// the given "namespace/name" key. It backs the throttler admission check: a Terminate must never
// be postponed by the parallelism limit.
func (wfc *WorkflowController) hasPendingTerminateAction(key string) bool {
	if wfc.wfActionInformer == nil {
		return false
	}
	objs, err := wfc.wfActionInformer.GetIndexer().ByIndex(indexes.WorkflowActionIndex, key)
	if err != nil {
		return false
	}
	for _, obj := range objs {
		if a, ok := obj.(*wfv1.WorkflowAction); ok && !a.Status.Fulfilled() && a.Spec.Action == wfv1.ActionTypeTerminate {
			return true
		}
	}
	return false
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
	remaining := wfc.Config.GetWorkflowActionTTL()
	if a.Status.CompletionTime != nil {
		remaining -= time.Since(a.Status.CompletionTime.Time)
	}
	if remaining < 0 {
		remaining = 0
	}
	wfc.wfActionGCQueue.AddAfter(key, remaining)
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
	if a.Status.CompletionTime != nil {
		if remaining := wfc.Config.GetWorkflowActionTTL() - time.Since(a.Status.CompletionTime.Time); remaining > 0 {
			wfc.wfActionGCQueue.AddAfter(key, remaining)
			return true
		}
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
