package persist

import (
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

//go:generate mockery -name OffloadNodeStatusRepo

// A place to save workflow node status.
// Implementations do not need to be fault tolerant. Expect this to be handled higher up the call stack.
// Implementations must be idempotent.
type OffloadNodeStatusRepo interface {
	// Save a node and return its version.
	Save(uid, namespace string, nodes wfv1.Nodes) (string, error)
	Get(uid, version string) (wfv1.Nodes, error)
	List(namespace string) (map[UUIDVersion]wfv1.Nodes, error)
	// List any old offloads.
	ListOldOffloads(namespace string) ([]UUIDVersion, error)
	Delete(uid, version string) error
	IsEnabled() bool
}

const OffloadNodeStatusDisabled = "Workflow has offloaded nodes, but offloading has been disabled"
