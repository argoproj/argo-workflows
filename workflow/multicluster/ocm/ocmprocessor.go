package ocm

import (
	"context"
	"fmt"

	v1 "open-cluster-management.io/api/work/v1"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

type OCMProcessor struct {
	wfInformer             cache.SharedIndexInformer // this one gets passed in
	wfStatusInformer       cache.SharedIndexInformer // this one gets constructed locally
	manifestWorkerInformer cache.SharedIndexInformer // this one gets constructed locally
	kubeclient             dynamic.Interface
}

func NewOCMProcessor(wfInformer cache.SharedIndexInformer, kubeclient dynamic.Interface) *OCMProcessor {
	ocm := &OCMProcessor{wfInformer: wfInformer}
	mf := v1.ManifestWork{}
	fmt.Println(mf)
	// todo: construct wfStatusInformer and register processStatusUpdate() to be called when there's a Status Update

	// todo: construct manifestWorkerInformer

	return ocm
}

// process Workflow additions and updates
func (ocm *OCMProcessor) ProcessWorkflow(ctx context.Context, wf *wfv1.Workflow) error {

	// locate the label which indicates the cluster name (which is the namespace that our Manifest Work will go)
	mwNamespace, found := wf.Labels[common.LabelKeyCluster]
	if !found {
		return fmt.Errorf("In multicluster mode, the Workflow Controller requires all Workflows to contain label %s", mwNamespace)
	}

	// use the Workflow UUID to derive the ManifestWork name
	mwName := string(wf.UID)

	// see if a ManifestWork already exists with this name/namespace
	_, exists, err := ocm.manifestWorkerInformer.GetStore().GetByKey(mwNamespace + "/" + mwName)
	if err != nil {
		return fmt.Errorf("error attempting to get ManifestWork: err=%v", err)
	}

	// if not, create it
	if !exists {

	} else {
		// update it (future work)

	}

	return nil
}

func (ocm *OCMProcessor) ProcessWorkflowDeletion(ctx context.Context, wf *wfv1.Workflow) error {
	// locate the label which indicates the cluster name (namespace of ManifestWork)

	// use the Workflow UUID to derive the ManifestWork name

	// delete the ManifestWork

	return nil
}

// find Workflow associated with WorkflowStatusResult and update it
/*func (ocm *OCMProcessor) processStatusUpdate(ctx context.Context, wfStatus *wfv1.WorkflowStatusResult) error {

	return nil
}*/

/*
func (ocm *OCMProcessor) newManifestWorkfInformer(resource schema.GroupVersionResource, client dynamic.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	ctx := context.Background()
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return client.Resource(resource).Namespace(namespace).List(ctx, options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return client.Resource(resource).Namespace(namespace).Watch(ctx, options)
			},
		},
		&unstructured.Unstructured{},
		resyncPeriod,
		indexers,
	)
}*/
