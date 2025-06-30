package fake

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
)

// this test fake does nothing
type noop struct{}

func (i noop) IsHydrated(wf *wfv1.Workflow) bool {
	return true
}

func (i noop) Hydrate(ctx context.Context, wf *wfv1.Workflow) error {
	return nil
}

func (i noop) Dehydrate(ctx context.Context, wf *wfv1.Workflow) error {
	return nil
}

func (i noop) HydrateWithNodes(wf *wfv1.Workflow, nodes wfv1.Nodes) {}

var Noop hydrator.Interface = &noop{}
