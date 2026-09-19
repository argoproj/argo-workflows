package executor

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// planFor selects the flow for a template. Every template prepares and
// runs the same way; what differs is how outputs are collected.
func planFor(tmpl *wfv1.Template, inputPlugins []wfv1.ArtifactPluginName) Plan {
	load := []Stage{loadArtifacts}
	for _, p := range inputPlugins {
		load = append(load, loadArtifactsFromPlugin(p))
	}
	plan := Plan{
		Prepare: []Stage{writeTemplate, stageFiles, Parallel(load...)},
		Run:     []Stage{initializeOutput, waitMain},
		Collect: []Stage{captureScriptResult, saveParameters, saveArtifacts, saveLogs, reportOutputs},
	}
	if tmpl.Resource != nil {
		plan.Collect = []Stage{reportLogs}
	}
	return plan
}

// selectedPlan returns the task's plan, deriving it from the template on
// first use.
func (we *WorkflowExecutor) selectedPlan() *Plan {
	if we.plan == nil {
		plan := planFor(&we.Template, we.inputArtifactPluginNames)
		we.plan = &plan
	}
	return we.plan
}

// Prepare runs the plan's Prepare phase: stages run in order on ctx and
// the first error aborts the phase and is returned, wrapped with the
// stage's name.
func (we *WorkflowExecutor) Prepare(ctx context.Context) error {
	for _, s := range we.selectedPlan().Prepare {
		if err := s.Run(ctx, we); err != nil {
			return fmt.Errorf("%s: %w", s.Name, err)
		}
	}
	return nil
}

// runRecording runs stages in order on ctx, recording each error with
// AddError and continuing to the next stage. Stages that need main's
// outputs are skipped when mainOutputsAvailable is false.
func (we *WorkflowExecutor) runRecording(ctx context.Context, stages []Stage, mainOutputsAvailable bool) {
	for _, s := range stages {
		if s.NeedsMainOutputs && !mainOutputsAvailable {
			continue
		}
		if err := s.Run(ctx, we); err != nil {
			we.AddError(ctx, err)
		}
	}
}
