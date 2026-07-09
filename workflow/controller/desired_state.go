package controller

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

// DesiredTask represents a task that the engine believes should be active.
// It contains all the context required to execute (create/reconcile) the task.
type DesiredTask struct {
	// TaskName is the name of the task (e.g., "A", "B", or "steps.step-1")
	TaskName string

	// OriginalTaskName is the name of the task in the DAG/Step definition (used for linking)
	OriginalTaskName string

	// TemplateScope indicates the scope of the template resolution
	TemplateScope string

	// TmplCtx is the resolved template context for this task (captures the template scope chain)
	TmplCtx *templateresolution.TemplateContext

	// Template is the resolved template definition for this task
	Template *wfv1.Template

	// TemplateRef is the original reference (needed for some executeTemplate calls)
	TemplateRef wfv1.TemplateReferenceHolder

	// NodeFlag contains execution flags like Retried, Hooked, etc.
	NodeFlag *wfv1.NodeFlag

	// BoundaryID is the node ID of the boundary (DAG/Steps parent)
	BoundaryID string

	// IsOnExit indicates if this is an exit handler task
	IsOnExit bool

	// Skipped indicates if the task should be marked as skipped (e.g. 'when' clause evaluated to false)
	Skipped bool

	// SkipReason provides the reason for skipping
	SkipReason string

	// ParentNodeNames lists the node names that this task should be added as a child of.
	// This allows the Reconciler to link nodes without knowing the DAG structure.
	ParentNodeNames []string
}

// TaskReconciler defines the interface for actuating the desired state.
type TaskReconciler interface {
	// Reconcile ensures the cluster state matches the desired tasks.
	Reconcile(ctx context.Context, desired []DesiredTask) error
}
