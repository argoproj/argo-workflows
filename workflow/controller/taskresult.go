package controller

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"k8s.io/apimachinery/pkg/util/wait"
	retryutil "k8s.io/client-go/util/retry"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	envutil "github.com/argoproj/argo-workflows/v3/util/env"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
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
			options.ResourceVersion = ""
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

func shouldErrorPodTimeout(node *wfv1.NodeStatus) bool {
	return time.Since(node.StartedAt.Time) <= envutil.LookupEnvDurationOr("POD_ABSENT_TIMEOUT", 2*time.Minute)
}

func (woc *wfOperationCtx) taskResultReconciliation() error {

	objs, _ := woc.controller.taskResultInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, woc.wf.Namespace+"/"+woc.wf.Name)
	woc.log.WithField("numObjs", len(objs)).Info("Task-result reconciliation")

	taskresultCli := woc.controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(woc.wf.Namespace)
	// We will be hitting the cache here, should we bypass the cache?
	podList, err := woc.getAllWorkflowPods()
	if err != nil {
		woc.log.Warnf("couldn't retrieve pods")
		return err
	}
	podMap := make(map[string]*apiv1.Pod)

	// can use woc.nodePodExit technically
	for _, pod := range podList {
		pod := pod
		nodeID := woc.nodeID(pod)
		podMap[nodeID] = pod
	}

	for _, obj := range objs {
		result := obj.(*wfv1.WorkflowTaskResult)
		resultName := result.GetName()

		woc.log.Debugf("task result:\n%+v", result)
		woc.log.Debugf("task result name:\n%+v", resultName)

		label, foundLabel := result.Labels[common.LabelKeyReportOutputsCompleted]

		pod, foundPod := podMap[result.Name]
		node, err := woc.wf.Status.Nodes.Get(result.Name)
		if err != nil {
			if foundPod {
				// how does this path make any sense?
				// pod created but informed not yet updated
				woc.log.Errorf("couldn't obtain node for %s, but found pod, this is not expected, doing nothing", result.Name)
			}
			continue
		}

		if foundPod {
			woc.log.Debugf("Got pod %s with phase %s for task result %s and node id %s with label %s", pod.Name, pod.Status.Phase, resultName, result.Name, label)
			woc.log.Debugf("The node phase was %s for node named %s", node.Phase, node.Name)
		} else {
			woc.log.Debugf("couldn't find pod")
			timeout := shouldErrorPodTimeout(node)
			if timeout {
				woc.log.Infof("Determined controller should timeout for %s", result.Name)
				result.Labels[common.LabelKeyReportOutputsCompleted] = "true"

				// patch the label
				err := retryutil.OnError(wait.Backoff{
					Duration: time.Second,
					Factor:   2,
					Jitter:   0.1,
					Steps:    5,
					Cap:      30 * time.Second,
				}, errorsutil.IsTransientErr, func() error {
					data, err := json.Marshal(&wfv1.WorkflowTaskResult{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								common.LabelKeyReportOutputsCompleted: "true",
							},
						},
					})
					if err != nil {
						return err
					}
					_, err = taskresultCli.Patch(context.Background(),
						result.Name,
						types.MergePatchType,
						data,
						metav1.PatchOptions{},
					)
					return err
				})
				if err != nil {
					woc.log.Errorf("couldn't patch taskresult, got error: %s", err)
				}
				woc.markNodePhase(node.Name, wfv1.NodeFailed, "pod was absent")
			} else {
				woc.log.Debugf("Determined controller shouldn't timeout %s", result.Name)
			}
		}
		// If the task result is completed, set the state to true.
		if label == "true" {
			woc.log.Debugf("Marking task result complete %s", resultName)
			woc.wf.Status.MarkTaskResultComplete(resultName)
		} else if label != "true" && foundLabel {
			woc.log.Debugf("Marking task result incomplete %s", resultName)
			woc.wf.Status.MarkTaskResultIncomplete(resultName)
		}

		nodeID := result.Name
		old, err := woc.wf.Status.Nodes.Get(nodeID)
		if err != nil {
			continue
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
		if !reflect.DeepEqual(&old, newNode) {
			woc.log.
				WithField("nodeID", nodeID).
				Info("task-result changed")
			woc.wf.Status.Nodes.Set(nodeID, *newNode)
			woc.updated = true
		}
	}
	return nil
}
