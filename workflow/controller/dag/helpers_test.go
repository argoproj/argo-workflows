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

// EvaluateTask returns the evaluation result for one task.
// Test-only convenience over EvaluateAll, so tests see exactly what the
// engine sees — including the per-child results EvaluateAll emits for
// expanded (withItems/withParam/withSequence) tasks.
func (e *DAGEvaluator) EvaluateTask(ctx context.Context, taskName string) EvaluationResult {
	return e.EvaluateAll(ctx)[taskName]
}
