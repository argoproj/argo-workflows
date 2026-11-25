package controller

import (
	"context"
	"reflect"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	"k8s.io/apimachinery/pkg/selection"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v4/pkg/client/informers/externalversions/workflow/v1alpha1"
	envutil "github.com/argoproj/argo-workflows/v4/util/env"
	informerutil "github.com/argoproj/argo-workflows/v4/util/informer"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
)

var (
	workflowReq, _ = labels.NewRequirement(common.LabelKeyWorkflow, selection.Exists, nil)
)

func (wfc *WorkflowController) newWorkflowTaskResultInformer(ctx context.Context) cache.SharedIndexInformer {
	log := logging.RequireLoggerFromContext(ctx)
	labelSelector := labels.NewSelector().
		Add(*workflowReq).
		Add(wfc.instanceIDReq()).
		String()
	log.WithField("labelSelector", labelSelector).
		Info(ctx, "Watching task results")

	// This is a generated function, so we can't change the context.
	//nolint:contextcheck
	informer := wfextvv1alpha1.NewFilteredWorkflowTaskResultInformer(
		wfc.wfclientset,
		wfc.GetManagedNamespace(),
		20*time.Minute,
		cache.Indexers{
			indexes.WorkflowIndex: indexes.MetaWorkflowIndexFunc,
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
	//nolint:errcheck // the error only happens if the informer was stopped, and it hasn't even started (https://github.com/kubernetes/client-go/blob/46588f2726fa3e25b1704d6418190f424f95a990/tools/cache/shared_informer.go#L580)
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(newObj any) {
				result := newObj.(*wfv1.WorkflowTaskResult)
				workflow := result.Labels[common.LabelKeyWorkflow]
				wfc.wfQueue.AddRateLimited(result.Namespace + "/" + workflow)
			},
			UpdateFunc: func(old, newObj any) {
				wfc.onWorkflowTaskResultUpdate(ctx, old.(*wfv1.WorkflowTaskResult), newObj.(*wfv1.WorkflowTaskResult))
			},
		})
	return informer
}

// onWorkflowTaskResultUpdate enqueues the result's workflow. If this update marks
// the outputs complete and the node's pod has already finished, it uses the
// completion delay; otherwise the normal rate-limited delay.
func (wfc *WorkflowController) onWorkflowTaskResultUpdate(ctx context.Context, old, result *wfv1.WorkflowTaskResult) {
	key := result.Namespace + "/" + result.Labels[common.LabelKeyWorkflow]
	if wfc.taskResultCompletedAfterPod(old, result) {
		wfc.enqueueNodeCompletion(ctx, key, result.OwnerReferences)
		return
	}
	wfc.wfQueue.AddRateLimited(key)
}

// podFinishedAfterTaskResult reports whether this pod update just finished the
// pod while its TaskResult is already complete, or absent for templates without one.
// Adds and deletes, including the relist after a restart, have no oldPod and never count.
func (wfc *WorkflowController) podFinishedAfterTaskResult(oldPod, newPod *apiv1.Pod) bool {
	if wfc.minCompletionRequeueDelay == nil || oldPod == nil {
		return false
	}
	justFinished := !podFinished(oldPod) && podFinished(newPod)
	if !justFinished {
		return false
	}
	// Without a node ID the TaskResult lookup below would find nothing and wrongly count as complete.
	nodeID := newPod.Annotations[common.AnnotationKeyNodeID]
	if nodeID == "" {
		return false
	}
	if wfc.taskResultInformer == nil {
		return true
	}
	// The TaskResult is named after the node ID.
	obj, exists, err := wfc.taskResultInformer.GetIndexer().GetByKey(newPod.Namespace + "/" + nodeID)
	if err != nil {
		return false
	}
	return !exists || outputsReported(obj.(*wfv1.WorkflowTaskResult))
}

// taskResultCompletedAfterPod reports whether this TaskResult update just marked
// the outputs complete while the node's pod has already finished. If the pod is
// gone (evicted, or deleted outside Argo) it returns false.
func (wfc *WorkflowController) taskResultCompletedAfterPod(old, result *wfv1.WorkflowTaskResult) bool {
	if wfc.minCompletionRequeueDelay == nil || wfc.PodController == nil {
		return false
	}
	justReported := !outputsReported(old) && outputsReported(result)
	if !justReported {
		return false
	}
	// The TaskResult is named after the node ID.
	objs, err := wfc.PodController.GetPodsByIndex(indexes.NodeIDIndex, result.Namespace+"/"+result.Name)
	if err != nil {
		return false
	}
	for _, obj := range objs {
		if pod, ok := obj.(*apiv1.Pod); ok && podFinished(pod) {
			return true
		}
	}
	return false
}

// podFinished reports whether a pod reached Succeeded or Failed. A pod deleted
// before finishing does not count.
func podFinished(pod *apiv1.Pod) bool {
	return pod.Status.Phase == apiv1.PodSucceeded || pod.Status.Phase == apiv1.PodFailed
}

// outputsReported reports whether the executor has marked the TaskResult's outputs complete.
func outputsReported(result *wfv1.WorkflowTaskResult) bool {
	return result.Labels[common.LabelKeyReportOutputsCompleted] == "true"
}

func recentlyDeleted(ctx context.Context, node *wfv1.NodeStatus) bool {
	return time.Since(node.FinishedAt.Time) <= envutil.LookupEnvDurationOr(ctx, "RECENTLY_DELETED_POD_DURATION", 2*time.Minute)
}

func recentlyCompleted(ctx context.Context, node *wfv1.NodeStatus) bool {
	return time.Since(node.FinishedAt.Time) <= envutil.LookupEnvDurationOr(ctx, "TASK_RESULT_TIMEOUT_DURATION", 10*time.Minute)
}

func (woc *wfOperationCtx) taskResultReconciliation(ctx context.Context) {
	objs, _ := woc.controller.taskResultInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, woc.wf.Namespace+"/"+woc.wf.Name)
	woc.log.WithField("numObjs", len(objs)).Info(ctx, "Task-result reconciliation")

	ctx, span := woc.controller.tracing.StartReconcileTaskResults(ctx)
	defer span.End()
	for _, obj := range objs {
		result := obj.(*wfv1.WorkflowTaskResult)
		resultName := result.GetName()

		woc.log.WithField("result", result).Debug(ctx, "task result")
		woc.log.WithField("resultName", resultName).Debug(ctx, "task result name")

		label := result.Labels[common.LabelKeyReportOutputsCompleted]
		_, span := woc.controller.tracing.StartReconcileTaskResult(ctx, resultName, label == "true")
		// If the task result is completed, set the state to true.
		switch label {
		case "true":
			woc.log.WithField("resultName", resultName).Debug(ctx, "Marking task result complete")
			woc.wf.Status.MarkTaskResultComplete(ctx, resultName)
		case "false":
			woc.log.WithField("resultName", resultName).Debug(ctx, "Marking task result incomplete")
			woc.wf.Status.MarkTaskResultIncomplete(ctx, resultName)
		}

		nodeID := result.Name
		old, err := woc.wf.Status.Nodes.Get(nodeID)
		if err != nil {
			span.End()
			continue
		}
		// Mark task result as completed if it has no chance to be completed, we use phase here to avoid caring about the sync status.
		if label == "false" && old.Completed() {
			if (!woc.nodePodExist(*old) && recentlyDeleted(ctx, old)) || recentlyCompleted(ctx, old) {
				woc.log.WithField("nodeID", nodeID).Debug(ctx, "Wait for marking task result as completed because pod is recently deleted.")
				// If the pod was deleted, then it is possible that the controller never get another informer message about it.
				// In this case, the workflow will only be requeued after the resync period (20m). This means
				// workflow will not update for 20m. Requeuing here prevents that happening.
				woc.requeue()
				span.End()
				continue
			}
			woc.log.WithField("nodeID", nodeID).Info(ctx, "Marking task result as completed because pod has been deleted for a while.")
			woc.wf.Status.MarkTaskResultComplete(ctx, nodeID)
		}
		newNode := old.DeepCopy()
		if result.Outputs.HasOutputs() {
			if newNode.Outputs == nil {
				newNode.Outputs = &wfv1.Outputs{}
			}
			result.Outputs.DeepCopyInto(newNode.Outputs)               // preserve any existing values
			if old.Outputs != nil && newNode.Outputs.ExitCode == nil { // prevent overwriting of ExitCode
				newNode.Outputs.ExitCode = old.Outputs.ExitCode
			}
		}
		if result.Progress.IsValid() {
			newNode.Progress = result.Progress
		}
		if !reflect.DeepEqual(old, newNode) {
			woc.log.
				WithField("nodeID", nodeID).
				Debug(ctx, "task-result changed")
			woc.wf.Status.Nodes.Set(ctx, nodeID, *newNode)
			woc.updated = true
		}
		span.End()
	}
}
