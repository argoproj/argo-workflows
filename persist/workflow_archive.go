package persist

import (
	"time"

	"k8s.io/apimachinery/pkg/labels"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

//go:generate mockery -name WorkflowArchive

// A place to save workflow node status.
// Implementations do not need to be fault tolerant. Expect this to be handled higher up the call stack.
// Implementations must be idempotent.
type WorkflowArchive interface {
	// Archive the workflow.
	ArchiveWorkflow(wf *wfv1.Workflow) error
	// List workflows. The most recently started workflows at the beginning (i.e. index 0 is the most recent).
	ListWorkflows(namespace string, minStartAt, maxStartAt time.Time, labelRequirements labels.Requirements, limit, offset int) (wfv1.Workflows, error)
	// Will return nil if not found rather than an error.
	// Should return only  "meta.name", "meta.namespace", "meta.uid", "status.phase", "status.startedAt", "status.finishedAt"
	GetWorkflow(uid string) (*wfv1.Workflow, error)
	DeleteWorkflow(uid string) error
	// Perform any periodic clean-up. E.g. Delete any archived workflows that are older than the TTL.
	Run(stopCh <-chan struct{})
	// Whether or not archiving is possible. Non-null implementations should always return true.
	IsEnabled() bool
}
