package ocm

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func (ocm *OCMProcessor) processStatusUpdate(ctx context.Context, wfStatus *wfv1.WorkflowStatusResult) error {

	// status obj should have label of wf's name and ns
	// get original wf using wfStatus name
	workflow, err := ocm.wfInformer.GetIndexer().ByIndex(indexes.UIDIndex, string(wfStatus.Name))
	if err != nil {
		return err
	}
	un := workflow[0].(*unstructured.Unstructured)
	wf, err := util.FromUnstructured(un)
	if err != nil {
		return err
	}

	// update wf status from wfStatus object
	wf.Status = *wfStatus.WorkflowStatus.DeepCopy()

	// update wf labels from wfStatus object
	wf.Labels["workflows.argoproj.io/archive-strategy"] = "false"
	wf.Labels[common.LabelKeyCompleted] = "true"
	wf.Labels[common.LabelKeyPhase] = string(wf.Status.Phase)

	// todo: delete object

	return nil
}
