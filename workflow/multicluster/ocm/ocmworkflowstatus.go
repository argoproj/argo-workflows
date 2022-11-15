package ocm

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (ocm *OCMProcessor) processStatusUpdate(ctx context.Context, wfStatus *wfv1.WorkflowStatusResult) error {

	// status obj should have label of wf's name and ns
	// get original wf using wfStatus name
	workflow, err := ocm.wfStatusInformer.GetIndexer().ByIndex(indexes.UIDIndex, string(wfStatus.UID))
	if err != nil {
		return err
	}
	un := workflow[0].(*unstructured.Unstructured)
	wf, err := util.FromUnstructured(un)
	if err != nil {
		return err
	}

	// update wf status from wfStatus object
	wf.Status = wfStatus.WorkflowStatus

	// update wf labels from wfStatus object
	wf.Labels = wfStatus.Labels

	// delete object

	return nil
}
