package ocm

import (
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

type OCMProcessor struct {
	WFInformer cache.SharedIndexInformer
}

func (ocm *OCMProcessor) ProcessWorkflow(wf *wfv1.Workflow) error {
	return nil
}

func (ocm *OCMProcessor) ProcessWorkflowDeletion(wf *wfv1.Workflow) error {
	return nil
}

func (ocm *OCMProcessor) ProcessStatusUpdate(wfStatus *wfv1.WorkflowStatusResult) error {
	return nil
}
