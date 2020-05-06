package hydrator

import (
	"os"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/packer"
)

type Interface interface {
	// whether or not the workflow in hydrated
	IsHydrated(wf *wfv1.Workflow) bool
	// hydrate the workflow - doing nothing if it is already hydrated
	Hydrate(wf *wfv1.Workflow) error
	// dehydrate the workflow - doing nothing if already dehydrated
	Dehydrate(wf *wfv1.Workflow) error
	// hydrate the workflow using the provided nodes
	HydrateWithNodes(wf *wfv1.Workflow, nodes wfv1.Nodes)
}

func New(offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo) Interface {
	return &hydrator{offloadNodeStatusRepo}
}

type hydrator struct {
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
}

func (h hydrator) IsHydrated(wf *wfv1.Workflow) bool {
	return !(wf.Status.CompressedNodes != "" || wf.Status.IsOffloadNodeStatus())
}

func (h hydrator) HydrateWithNodes(wf *wfv1.Workflow, nodes wfv1.Nodes) {
	wf.Status.Nodes = nodes
	wf.Status.CompressedNodes = ""
	wf.Status.OffloadNodeStatusVersion = ""
}

func (h hydrator) Hydrate(wf *wfv1.Workflow) error {
	if h.IsHydrated(wf) {
		return nil
	}
	err := packer.DecompressWorkflow(wf)
	if err != nil {
		return err
	}
	if wf.Status.IsOffloadNodeStatus() {
		offloadedNodes, err := h.offloadNodeStatusRepo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
		if err != nil {
			return err
		}
		h.HydrateWithNodes(wf, offloadedNodes)
	}
	return nil
}

func (h hydrator) Dehydrate(wf *wfv1.Workflow) error {
	if !h.IsHydrated(wf) {
		return nil
	}
	err := packer.CompressWorkflowIfNeeded(wf)
	if err == nil {
		wf.Status.OffloadNodeStatusVersion = ""
		return nil
	}
	if packer.IsTooLargeError(err) || os.Getenv("ALWAYS_OFFLOAD_NODE_STATUS") == "true" {
		offloadVersion, err := h.offloadNodeStatusRepo.Save(string(wf.UID), wf.Namespace, wf.Status.Nodes)
		if err != nil {
			return err
		}
		wf.Status.Nodes = nil
		wf.Status.CompressedNodes = ""
		wf.Status.OffloadNodeStatusVersion = offloadVersion
		return nil
	} else {
		return err
	}
}
