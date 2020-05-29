package fake

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	testutil "github.com/argoproj/argo/test/util"
	"github.com/argoproj/argo/workflow/hydrator"
)

// this test fake is nearly a Reference Implementation
type always struct{}

func (i always) IsHydrated(wf *wfv1.Workflow) bool {
	return wf.Status.OffloadNodeStatusVersion == ""
}

func (i always) Hydrate(wf *wfv1.Workflow) error {
	if !i.IsHydrated(wf) {
		testutil.MustUnmarshallJSON(wf.Status.OffloadNodeStatusVersion, &wf.Status.Nodes)
		wf.Status.OffloadNodeStatusVersion = ""
	}
	return nil
}

func (i always) Dehydrate(wf *wfv1.Workflow) error {
	if i.IsHydrated(wf) {
		wf.Status.OffloadNodeStatusVersion = testutil.MustMarshallJSON(&wf.Status.Nodes)
		wf.Status.Nodes = nil
	}
	return nil
}

func (i always) HydrateWithNodes(wf *wfv1.Workflow, nodes wfv1.Nodes) {
	wf.Status.Nodes = nodes
	wf.Status.OffloadNodeStatusVersion = ""
}

var Always hydrator.Interface = &always{}
