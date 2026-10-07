package controller

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

// K8sTaskReconciler implements TaskReconciler for Kubernetes-based execution.
// It wraps wfOperationCtx to perform actual cluster operations.
type K8sTaskReconciler struct {
	woc      *wfOperationCtx
	tmplCtx  *templateresolution.TemplateContext
	nodeName string // The name of the parent/boundary node (for error reporting)
}

// NewK8sTaskReconciler creates a new reconciler.
func NewK8sTaskReconciler(woc *wfOperationCtx, tmplCtx *templateresolution.TemplateContext, nodeName string) *K8sTaskReconciler {
	return &K8sTaskReconciler{
		woc:      woc,
		tmplCtx:  tmplCtx,
		nodeName: nodeName,
	}
}

// Reconcile ensures the cluster state matches the desired tasks.
//
// Return contract:
//   - nil: every desired task either materialized a node or was a no-op
//     (e.g. Skipped re-entry where the node already exists).
//   - ErrParallelismReached / ErrResourceRateLimitReached / ErrDeadlineExceeded:
//     returned at once, nothing recorded; the caller decides whether to keep
//     dispatching.
//   - any other error is the task's own outcome: recordTaskError records it as
//     an Error on the task's node (created and linked if the dispatch left
//     none), the rest of the batch is still reconciled, and the first such
//     error is returned unwrapped, with no task prefix, as entry, onExit and
//     hook errors are, and as the DAG task node's message records it. The
//     boundary is not touched, so sibling tasks can still be scheduled.
func (r *K8sTaskReconciler) Reconcile(ctx context.Context, desired []DesiredTask) error {
	var taskErr error
	for _, dt := range desired {
		_, lookupErr := r.woc.wf.GetNodeByName(dt.TaskName)
		isNew := lookupErr != nil
		// If Skipped
		if dt.Skipped {
			// Check if node exists, if not create as skipped
			if isNew {
				r.woc.initializeNode(ctx, dt.TaskName, wfv1.NodeTypeSkipped, dt.TemplateScope, dt.TemplateRef, dt.BoundaryID, wfv1.NodeSkipped, &wfv1.NodeFlag{}, true, dt.SkipReason)
				r.linkTasks(ctx, dt)
			}
			continue
		}

		// Execute Template — use per-task template context if available (preserves template scope chain).
		// The resolved TmplCtx is passed for child template resolution (e.g. inside a DAG/Steps),
		// while TemplateScope (parent scope) is passed via opts for node creation.
		tmplCtx := r.tmplCtx
		if dt.TmplCtx != nil {
			tmplCtx = dt.TmplCtx
		}
		_, err := r.woc.executeProcessedTemplate(ctx, dt.TaskName, dt.TemplateRef, tmplCtx, dt.Template, &executeTemplateOpts{
			boundaryID:     dt.BoundaryID,
			onExitTemplate: dt.IsOnExit,
			nodeFlag:       dt.NodeFlag,
			templateScope:  dt.TemplateScope,
		})
		// Link a node this call created, once it exists, and only then:
		// creation can be deferred (parallelism) and an
		// edge to a node never created can later be claimed by a colliding
		// name (#16376). A node that already existed is not linked again: its
		// dependencies' outbound nodes may have moved on since (a daemon's next
		// retry attempt) or, for a dependency that had no children, now lead
		// back through the task itself.
		if _, getErr := r.woc.wf.GetNodeByName(dt.TaskName); isNew && getErr == nil {
			r.linkTasks(ctx, dt)
		}
		if err == nil {
			continue
		}
		if isThrottleErr(err) {
			return err
		}
		r.woc.log.WithError(err).WithField("task", dt.TaskName).Error(ctx, "task errored")
		r.recordTaskError(ctx, dt, err)
		if taskErr == nil {
			taskErr = err
		}
	}
	return taskErr
}

// recordTaskError records a task's own dispatch error as an Error on its
// node, creating and linking the node when the
// dispatch failed before creating it. A node that already reached a terminal
// phase (a timed-out node marked Failed, a max-depth Error) keeps it.
func (r *K8sTaskReconciler) recordTaskError(ctx context.Context, dt DesiredTask, err error) {
	node, getErr := r.woc.wf.GetNodeByName(dt.TaskName)
	if getErr != nil {
		r.woc.initializeNode(ctx, dt.TaskName, wfv1.NodeTypeSkipped, dt.TemplateScope, dt.TemplateRef, dt.BoundaryID, wfv1.NodeError, dt.NodeFlag, true, err.Error())
		r.linkTasks(ctx, dt)
		return
	}
	if !node.Fulfilled() {
		r.woc.markNodeError(ctx, dt.TaskName, err)
	}
}

func (r *K8sTaskReconciler) linkTasks(ctx context.Context, dt DesiredTask) {
	for _, parent := range dt.ParentNodeNames {
		r.woc.addChildNode(ctx, parent, dt.TaskName)
	}
}
