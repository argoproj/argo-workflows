package executor

import "context"

// Stage is one named step of a task's flow, operating on the task's
// executor. Every Stage is declared in stages.go, the registry, and plans
// in plan.go compose them into a Plan. Adding behaviour to argoexec means
// adding a stage there or a plan here, never a new hand-written sequence
// in a command.
type Stage struct {
	Name string
	Run  func(ctx context.Context, we *WorkflowExecutor) error
	// NeedsMainOutputs marks a Collect stage that reads what the user's
	// main produced. Such stages are skipped when preparation failed and
	// main never ran its command, so they don't fail on missing files.
	NeedsMainOutputs bool
}

// Plan is a task's flow as data: three phases whose runtime rules are
// fixed by the runner rather than by each stage.
//
//   - Prepare runs on the cancellable context; the first error aborts the
//     phase and is returned (see WorkflowExecutor.Prepare).
//   - Run and Collect record each error and continue; Run uses the
//     cancellable context, Collect the background one so termination
//     cannot lose outputs (see WorkflowExecutor.PostMain).
type Plan struct {
	Prepare []Stage
	Run     []Stage
	Collect []Stage
}
