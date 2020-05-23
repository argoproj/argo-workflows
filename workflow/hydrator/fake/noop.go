package fake

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/hydrator"
)

// this test fake does nothing
type noop struct{}

func (i noop) IsHydrated(wf *wfv1.Workflow) bool {
	return true
}

func (i noop) Hydrate(wf *wfv1.Workflow) error {
	return nil
}

func (i noop) Dehydrate(wf *wfv1.Workflow) error {
	return nil
}

func (i noop) HydrateWithNodes(wf *wfv1.Workflow, nodes wfv1.Nodes) {}

var Noop hydrator.Interface = &noop{}
