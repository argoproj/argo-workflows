package controller

import (
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	"k8s.io/apimachinery/pkg/selection"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	envutil "github.com/argoproj/argo-workflows/v3/util/env"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
)

var (
	workflowReq, _ = labels.NewRequirement(common.LabelKeyWorkflow, selection.Exists, nil)
)

func (wfc *WorkflowController) newWorkflowTaskResultInformer() cache.SharedIndexInformer {
	labelSelector := labels.NewSelector().
		Add(*workflowReq).
		Add(wfc.instanceIDReq()).
		String()
	log.WithField("labelSelector", labelSelector).
		Info("Watching task results")
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
	//nolint:errcheck // the error only happens if the informer was stopped, and it hasn't even started (https://github.com/kubernetes/client-go/blob/46588f2726fa3e25b1704d6418190f424f95a990/tools/cache/shared_informer.go#L580)
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(new interface{}) {
				result := new.(*wfv1.WorkflowTaskResult)
				workflow := result.Labels[common.LabelKeyWorkflow]
				wfc.wfQueue.AddRateLimited(result.Namespace + "/" + workflow)
			},
			UpdateFunc: func(old, new interface{}) {
				result := new.(*wfv1.WorkflowTaskResult)
				workflow := result.Labels[common.LabelKeyWorkflow]
				wfc.wfQueue.AddRateLimited(result.Namespace + "/" + workflow)
			},
		})
	return informer
}

func recentlyDeleted(node *wfv1.NodeStatus) bool {
	return time.Since(node.FinishedAt.Time) <= envutil.LookupEnvDurationOr("RECENTLY_DELETED_POD_DURATION", 2*time.Minute)
}

func recentlyCompleted(node *wfv1.NodeStatus) bool {
	return time.Since(node.FinishedAt.Time) <= envutil.LookupEnvDurationOr("TASK_RESULT_TIMEOUT_DURATION", 10*time.Minute)
}

func (woc *wfOperationCtx) taskResultReconciliation() {
	objs, _ := woc.controller.taskResultInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, woc.wf.Namespace+"/"+woc.wf.Name)
	woc.log.WithField("numObjs", len(objs)).Info("Task-result reconciliation")

	for _, obj := range objs {
		result := obj.(*wfv1.WorkflowTaskResult)
		resultName := result.GetName()

		woc.log.Debugf("task result:\n%+v", result)
		woc.log.Debugf("task result name:\n%+v", resultName)

		label := result.Labels[common.LabelKeyReportOutputsCompleted]
		// If the task result is completed, set the state to true.
		switch label {
		case "true":
			woc.log.Debugf("Marking task result complete %s", resultName)
			woc.wf.Status.MarkTaskResultComplete(resultName)
		case "false":
			woc.log.Debugf("Marking task result incomplete %s", resultName)
			woc.wf.Status.MarkTaskResultIncomplete(resultName)
		}

		nodeID := result.Name
		old, err := woc.wf.Status.Nodes.Get(nodeID)
		if err != nil {
			continue
		}
		// Mark task result as completed if it has no chance to be completed, we use phase here to avoid caring about the sync status.
		if label == "false" && old.Phase.Completed() {
			if (!woc.nodePodExist(*old) && recentlyDeleted(old)) || recentlyCompleted(old) {
				woc.log.WithField("nodeID", nodeID).Debug("Wait for marking task result as completed because pod is recently deleted.")
				// If the pod was deleted, then it is possible that the controller never get another informer message about it.
				// In this case, the workflow will only be requeued after the resync period (20m). This means
				// workflow will not update for 20m. Requeuing here prevents that happening.
				woc.requeue()
				continue
			}
			woc.log.WithField("nodeID", nodeID).Info("Marking task result as completed because pod has been deleted for a while.")
			woc.wf.Status.MarkTaskResultComplete(nodeID)
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
				Debug("task-result changed")
			woc.wf.Status.Nodes.Set(nodeID, *newNode)
			woc.updated = true
		}
	}
}
