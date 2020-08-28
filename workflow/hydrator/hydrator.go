package hydrator

import (
	"os"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

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

var alwaysOffloadNodeStatus = os.Getenv("ALWAYS_OFFLOAD_NODE_STATUS") == "true"

func init() {
	log.WithField("alwaysOffloadNodeStatus", alwaysOffloadNodeStatus).Debug("Hydrator config")
}

type hydrator struct {
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
}

func (h hydrator) IsHydrated(wf *wfv1.Workflow) bool {
	return wf.Status.CompressedNodes == "" && !wf.Status.IsOffloadNodeStatus()
}

func (h hydrator) HydrateWithNodes(wf *wfv1.Workflow, offloadedNodes wfv1.Nodes) {
	wf.Status.Nodes = offloadedNodes
	wf.Status.CompressedNodes = ""
	wf.Status.OffloadNodeStatusVersion = ""
}

func (h hydrator) Hydrate(wf *wfv1.Workflow) error {
	err := packer.DecompressWorkflow(wf)
	if err != nil {
		return err
	}
	if wf.Status.IsOffloadNodeStatus() {
		var offloadedNodes wfv1.Nodes
		err := wait.ExponentialBackoff(retry.DefaultBackoff, func() (bool, error) {
			offloadedNodes, err = h.offloadNodeStatusRepo.Get(string(wf.UID), wf.GetOffloadNodeStatusVersion())
			return err == nil, err
		})
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
	var err error
	if !alwaysOffloadNodeStatus {
		err = packer.CompressWorkflowIfNeeded(wf)
		if err == nil {
			wf.Status.OffloadNodeStatusVersion = ""
			return nil
		}
	}
	if packer.IsTooLargeError(err) || alwaysOffloadNodeStatus {
		var offloadVersion string
		err := wait.ExponentialBackoff(retry.DefaultBackoff, func() (bool, error) {
			offloadVersion, err = h.offloadNodeStatusRepo.Save(string(wf.UID), wf.Namespace, wf.Status.Nodes)
			return err == nil, err
		})
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
