package hydrator

import (
	"os"

	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/packer"
)

type Interface interface {
	Hydrate(wf *wfv1.Workflow) error
	Dehydrate(wf *wfv1.Workflow) error
}

func New(offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo) Interface {
	return &hydrator{offloadNodeStatusRepo}
}

type hydrator struct {
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
}

func (h hydrator) Hydrate(wf *wfv1.Workflow) error {
	err := packer.DecompressWorkflow(wf)
	if err != nil {
		return err
	}
	if wf.Status.IsOffloadNodeStatus() {
		offloadedNodes, err := h.offloadNodeStatusRepo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
		if err != nil {
			return err
		}
		wf.Status.Nodes = offloadedNodes
	}
	return nil
}

func (h hydrator) Dehydrate(wf *wfv1.Workflow) error {
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
