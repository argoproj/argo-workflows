package ocm

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

type OCMProcessor struct {
	wfInformer             cache.SharedIndexInformer // this one gets passed in
	wfStatusInformer       cache.SharedIndexInformer // this one gets constructed locally
	manifestWorkerInformer cache.SharedIndexInformer // this one gets constructed locally
}

func NewOCMProcessor(wfInformer cache.SharedIndexInformer) *OCMProcessor {
	ocm := &OCMProcessor{wfInformer: wfInformer}

	// todo: construct wfStatusInformer and register processStatusUpdate() to be called when there's a Status Update

	return ocm
}

// process Workflow additions and updates
func (ocm *OCMProcessor) ProcessWorkflow(ctx context.Context, wf *wfv1.Workflow) error {

	// locate the label which indicates the cluster name (which is the namespace that our Manifest Work will go)

	// use the Workflow UUID to derive the ManifestWork name

	// see if a ManifestWork already exists with this name/namespace
	// if not, create it
	// else update it (future work)

	return nil
}

func (ocm *OCMProcessor) ProcessWorkflowDeletion(ctx context.Context, wf *wfv1.Workflow) error {
	// locate the label which indicates the cluster name (namespace of ManifestWork)

	// use the Workflow UUID to derive the ManifestWork name

	// delete the ManifestWork

	return nil
}

// find Workflow associated with WorkflowStatusResult and update it
func (ocm *OCMProcessor) processStatusUpdate(ctx context.Context, wfStatus *wfv1.WorkflowStatusResult) error {

	return nil
}
