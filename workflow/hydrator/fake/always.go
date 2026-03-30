package fake

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
)

// this test fake is nearly a Reference Implementation
type always struct{}

func (i always) IsHydrated(wf *wfv1.Workflow) bool {
	return wf.Status.OffloadNodeStatusVersion == ""
}

func (i always) Hydrate(ctx context.Context, wf *wfv1.Workflow) error {
	if !i.IsHydrated(wf) {
		wfv1.MustUnmarshal(wf.Status.OffloadNodeStatusVersion, &wf.Status.Nodes)
		wf.Status.OffloadNodeStatusVersion = ""
	}
	return nil
}

func (i always) Dehydrate(ctx context.Context, wf *wfv1.Workflow) error {
	if i.IsHydrated(wf) {
		wf.Status.OffloadNodeStatusVersion = wfv1.MustMarshallJSON(&wf.Status.Nodes)
		wf.Status.Nodes = nil
	}
	return nil
}

func (i always) HydrateWithNodes(wf *wfv1.Workflow, nodes wfv1.Nodes) {
	wf.Status.Nodes = nodes
	wf.Status.OffloadNodeStatusVersion = ""
}

var Always hydrator.Interface = &always{}
