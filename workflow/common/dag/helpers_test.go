package dag

import (
	"context"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// NewDAGEvaluator creates a new DAGEvaluator for a workflow and DAG template.
// Test-only convenience: production code builds the task slice itself (the
// engine wraps DAG tasks and Steps adapters) and uses NewDAGEvaluatorFromTasks.
func NewDAGEvaluator(wf *wfv1.Workflow, tmpl *wfv1.Template, boundaryID, boundaryName string) *DAGEvaluator {
	var dagTasks []Task
	if tmpl.DAG != nil {
		dagTasks = make([]Task, len(tmpl.DAG.Tasks))
		for i := range tmpl.DAG.Tasks {
			dagTasks[i] = &DAGTask{DAGTask: &tmpl.DAG.Tasks[i]}
		}
	}
	return NewDAGEvaluatorFromTasks(wf, dagTasks, tmpl, boundaryID, boundaryName)
}

// EvaluateTask evaluates a single task and returns its evaluation result.
// Test-only convenience: production code evaluates the whole DAG via
// EvaluateAll; this gives tests the same per-task view without building
// the full result map.
func (e *DAGEvaluator) EvaluateTask(ctx context.Context, taskName string) EvaluationResult {
	// Run evaluateAllStates to handle cascading omission
	e.evaluateAllStates(ctx)
	return e.evaluateTaskResult(ctx, taskName)
}
