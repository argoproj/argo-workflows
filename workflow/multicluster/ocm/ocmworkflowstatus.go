package ocm

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func (ocm *OCMProcessor) processStatusUpdate(ctx context.Context, wfStatus *wfv1.WorkflowStatusResult) error {

	// status obj should have label of wf's name and ns
	// get original wf using wfStatus name

	workflow, err := ocm.wfInformer.GetIndexer().ByIndex(indexes.UIDIndex, string(wfStatus.Name))
	if err != nil {
		return err
	}
	if len(workflow) == 0 {
		return fmt.Errorf("found no Workflows with UID %q", wfStatus.Name)
	}
	un := workflow[0].(*unstructured.Unstructured)
	wf, err := util.FromUnstructured(un)
	if err != nil {
		fmt.Printf("got error casting to workflow: err=%v\n", err)
		return err
	}

	log.Debugf("successfully located Workflow by UID %q: %q", wfStatus.Name, wf.Name)

	// update wf status from wfStatus object
	wf.Status = *wfStatus.WorkflowStatus.DeepCopy()
	// update wf labels from wfStatus object
	wf.Labels["workflows.argoproj.io/archive-strategy"] = "false"
	wf.Labels[common.LabelKeyCompleted] = "true"
	wf.Labels[common.LabelKeyPhase] = string(wf.Status.Phase)

	wfClient := ocm.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace)
	_, err = wfClient.Update(ctx, wf, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	// delete WorkflowStatusResult
	wfsrClient := ocm.wfclientset.ArgoprojV1alpha1().WorkflowStatusResults(wfStatus.Namespace)
	err = wfsrClient.Delete(ctx, wfStatus.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	return nil
}
