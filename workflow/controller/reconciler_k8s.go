package controller

import (
	"context"
	"errors"
	"fmt"

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
//   - ErrParallelismReached / ErrResourceRateLimitReached: throttling — caller
//     should stop dispatching further tasks this pass but NOT treat as fatal.
//   - ErrDeadlineExceeded / ErrTimeout: deadline/timeout — propagate without
//     marking the boundary (the node itself was already marked by checkConstraints).
//   - ErrMaxDepthExceeded: recursion guard — propagate without double-marking.
//   - any other error: a real per-task failure. postExecutionHandling already
//     marked the failing task node Errored; the boundary is NOT touched so
//     sibling tasks can still be scheduled.
func (r *K8sTaskReconciler) Reconcile(ctx context.Context, desired []DesiredTask) error {
	for _, dt := range desired {
		// If Skipped
		if dt.Skipped {
			// Check if node exists, if not create as skipped
			if _, err := r.woc.wf.GetNodeByName(dt.TaskName); err != nil {
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
		if err != nil {
			switch {
			case errors.Is(err, ErrParallelismReached),
				errors.Is(err, ErrResourceRateLimitReached):
				// Throttling: surface the sentinel so the caller can tell
				// "deliberate throttle" apart from "reconciler claimed success".
				return err
			case errors.Is(err, ErrDeadlineExceeded),
				errors.Is(err, ErrTimeout):
				// Deadline/timeout: propagate without marking parent as Error
				// (the node itself was already marked by checkConstraints).
				return err
			case errors.Is(err, ErrMaxDepthExceeded):
				// Max recursion depth: propagate without double-marking.
				return err
			}
			// Per-task error: postExecutionHandling already marked the failing
			// task node Errored. Do NOT mark the boundary — sibling tasks must
			// still get a chance to schedule. Log and return so the engine
			// can stop this Reconcile batch but continue overall execution.
			r.woc.log.WithError(err).WithField("task", dt.TaskName).Error(ctx, "task errored")
			return fmt.Errorf("task %s errored: %w", dt.OriginalTaskName, err)
		}

		// Linkage
		r.linkTasks(ctx, dt)
	}
	return nil
}

func (r *K8sTaskReconciler) linkTasks(ctx context.Context, dt DesiredTask) {
	for _, parent := range dt.ParentNodeNames {
		r.woc.addChildNode(ctx, parent, dt.TaskName)
	}
}
