package store

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	sutils "github.com/argoproj/argo-workflows/v4/server/utils"
)

type WorkflowLister interface {
	ListWorkflows(ctx context.Context, namespace, nameFilter, createdAfter, finishedBefore string, listOptions metav1.ListOptions) (*wfv1.WorkflowList, error)
	CountWorkflows(ctx context.Context, namespace, nameFilter, createdAfter, finishedBefore string, listOptions metav1.ListOptions) (int64, error)
}

type kubeLister struct {
	dyn dynamic.Interface
}

var _ WorkflowLister = &kubeLister{}

func NewKubeLister(dyn dynamic.Interface) WorkflowLister {
	return &kubeLister{dyn: dyn}
}

func (k *kubeLister) ListWorkflows(ctx context.Context, namespace, nameFilter, createdAfter, finishedBefore string, listOptions metav1.ListOptions) (*wfv1.WorkflowList, error) {
	gvr := wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowPlural)
	items, meta, err := sutils.TolerantList[wfv1.Workflow](ctx, k.dyn, gvr, namespace, listOptions)
	if err != nil {
		return nil, err
	}
	return &wfv1.WorkflowList{ListMeta: meta, Items: items}, nil
}

func (k *kubeLister) CountWorkflows(ctx context.Context, namespace, nameFilter, createdAfter, finishedBefore string, listOptions metav1.ListOptions) (int64, error) {
	// Count off the raw list: counting needs no typed objects, and going through
	// ListWorkflows/TolerantList would both pay the per-item JSON roundtrip and
	// silently undercount by dropping malformed workflows.
	gvr := wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowPlural)
	return sutils.CountList(ctx, k.dyn, gvr, namespace, listOptions)
}
