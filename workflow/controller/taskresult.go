package controller

import (
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
)

func (wfc *WorkflowController) newWorkflowTaskResultInformers() map[string]cache.SharedIndexInformer {
	informers := make(map[string]cache.SharedIndexInformer)
	for cluster, clientset := range wfc.workflowInterfaces {
		labelSelector := labels.NewSelector().
			Add(*workflowReq).
			Add(wfc.clusterReq(cluster)).
			Add(wfc.instanceIDReq()).
			String()
		log.WithField("labelSelector", labelSelector).
			WithField("cluster", cluster).Info("Creating task-result informer")
		informer := wfextvv1alpha1.NewFilteredWorkflowTaskResultInformer(
			clientset,
			wfc.GetManagedNamespace(),
			20*time.Minute,
			cache.Indexers{
				indexes.WorkflowIndex: indexes.MetaWorkflowIndexFunc,
			},
			func(options *metav1.ListOptions) {
				options.LabelSelector = labelSelector
			},
		)
		informer.AddEventHandler(
			cache.ResourceEventHandlerFuncs{
				AddFunc: func(new interface{}) {
					result := new.(*wfv1.WorkflowTaskResult)
					namespace := common.MetaWorkflowNamespace(result)
					workflow := result.Labels[common.LabelKeyWorkflow]
					wfc.wfQueue.AddRateLimited(namespace + "/" + workflow)
				},
				UpdateFunc: func(_, new interface{}) {
					result := new.(*wfv1.WorkflowTaskResult)
					namespace := common.MetaWorkflowNamespace(result)
					workflow := result.Labels[common.LabelKeyWorkflow]
					wfc.wfQueue.AddRateLimited(namespace + "/" + workflow)
				},
			})
		informers[cluster] = informer
	}
	return informers
}

func (woc *wfOperationCtx) taskResultReconciliation() {
	woc.log.Info("task-result reconciliation")
	for cluster, taskResultInformer := range woc.controller.taskResultInformers {
		objs, _ := taskResultInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, woc.wf.Namespace+"/"+woc.wf.Name)
		woc.log.WithField("numObjs", len(objs)).
			WithField("cluster", cluster).
			Info("Task-result reconciliation")
		for _, obj := range objs {
			result := obj.(*wfv1.WorkflowTaskResult)
			nodeID := result.Name
			old := woc.wf.Status.Nodes[nodeID]
			new := old.DeepCopy()
			if result.Outputs.HasOutputs() {
				if new.Outputs == nil {
					new.Outputs = &wfv1.Outputs{}
				}
				result.Outputs.DeepCopyInto(new.Outputs)               // preserve any existing values
				if old.Outputs != nil && new.Outputs.ExitCode == nil { // prevent overwriting of ExitCode
					new.Outputs.ExitCode = old.Outputs.ExitCode
				}
			}
			if result.Progress.IsValid() {
				new.Progress = result.Progress
			}
			if !reflect.DeepEqual(&old, new) {
				woc.log.
					WithField("nodeID", nodeID).
					Info("task-result changed")
				woc.wf.Status.Nodes[nodeID] = *new
				woc.updated = true
			}
		}
	}
}
