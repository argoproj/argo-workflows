package controller

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions"
	wfextvv1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func (wfc *WorkflowController) newWorkflowTaskResultInformer() wfextvv1alpha1.WorkflowTaskResultInformer {
	informer := externalversions.NewSharedInformerFactoryWithOptions(
		wfc.wfclientset,
		workflowTaskSetResyncPeriod,
		externalversions.WithNamespace(wfc.GetManagedNamespace()),
		externalversions.WithTweakListOptions(func(x *metav1.ListOptions) {
			r := util.InstanceIDRequirement(wfc.Config.InstanceID)
			x.LabelSelector = r.String()
		})).Argoproj().V1alpha1().WorkflowTaskResults()
	informer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(new interface{}) {
				result := new.(*wfv1.WorkflowTaskResult)
				workflow := result.Labels[common.LabelKeyWorkflow]
				wfc.wfQueue.AddRateLimited(result.Namespace + "/" + workflow)
			},
			UpdateFunc: func(_, new interface{}) {
				result := new.(*wfv1.WorkflowTaskResult)
				workflow := result.Labels[common.LabelKeyWorkflow]
				wfc.wfQueue.AddRateLimited(result.Namespace + "/" + workflow)
			},
		})
	return informer
}
