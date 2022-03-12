package controller

import (
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
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
		},
	)
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(new interface{}) {
				result := new.(*wfv1.WorkflowTaskResult)
				namespace := result.Namespace
				workflow := result.Labels[common.LabelKeyWorkflow]
				wfc.wfQueue.AddRateLimited(namespace + "/" + workflow)
			},
			UpdateFunc: func(_, new interface{}) {
				result := new.(*wfv1.WorkflowTaskResult)
				namespace := result.Namespace
				workflow := result.Labels[common.LabelKeyWorkflow]
				wfc.wfQueue.AddRateLimited(namespace + "/" + workflow)
			},
		})

	return informer
}

func (woc *wfOperationCtx) taskResultReconciliation() {
	objs, _ := woc.controller.taskResultInformer.GetIndexer().ByIndex(indexes.WorkflowIndex, woc.wf.Namespace+"/"+woc.wf.Name)
	for _, obj := range objs {
		result := obj.(*wfv1.WorkflowTaskResult)
		old := woc.wf.Status.Nodes[result.Name]
		new := old.DeepCopy()
		new.Outputs = result.Outputs
		new.Progress = result.Progress
		if !reflect.DeepEqual(old, new) {
			woc.log.
				WithField("nodeID", new.ID).
				Info("task-result changed")
			woc.wf.Status.Nodes[result.Name] = *new
			woc.updated = true
		}
	}
}
