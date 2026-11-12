package executor

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// Layout is how a task's flow is laid out across processes.
type Layout int

const (
	// LayoutLegacy is the classic pod: an init container prepares, main
	// runs under the emissary, a wait container collects.
	LayoutLegacy Layout = iota
	// LayoutInitless is the init-less pod: a supervisor container prepares
	// and collects while main waits for its ready marker.
	LayoutInitless
	// LayoutWorker runs the whole flow in one process, main included.
	LayoutWorker
)

// linksInputArtifacts reports whether main sees its input artifacts as
// symlinks into the input-artifacts directory (staged by the emissary in
// an init-less pod, by runMain in a worker), rather than through
// per-artifact bind mounts that main's writes land in.
func (l Layout) linksInputArtifacts() bool {
	return l == LayoutInitless || l == LayoutWorker
}

func (l Layout) String() string {
	switch l {
	case LayoutLegacy:
		return "legacy"
	case LayoutInitless:
		return "initless"
	case LayoutWorker:
		return "worker"
	}
	return fmt.Sprintf("Layout(%d)", int(l))
}

// planFor selects the flow for a template in a layout. The layout
// decides how the task is prepared and run: the legacy init container
// installs the binary and loads only non-plugin artifacts (each plugin has
// its own init container), the init-less supervisor writes the template
// and loads everything in parallel, and in both pods the Run phase waits
// for main's container. A worker checks the template, prepares the task's
// own tree and runs main itself. The template decides how outputs are
// collected.
func planFor(tmpl *wfv1.Template, layout Layout, inputPlugins []wfv1.ArtifactPluginName) Plan {
	plan := Plan{
		Prepare: []Stage{installArgoexec, stageFiles, loadArtifacts},
		Run:     []Stage{initializeOutput, waitMain},
		Collect: []Stage{captureScriptResult, saveParameters, saveArtifacts, saveLogs, reportOutputs},
	}
	switch layout {
	case LayoutInitless:
		load := []Stage{loadArtifacts}
		for _, p := range inputPlugins {
			load = append(load, loadArtifactsFromPlugin(p))
		}
		plan.Prepare = []Stage{writeTemplate, stageFiles, Parallel(load...)}
	case LayoutWorker:
		// One process runs everything: the template is in memory, so there
		// is nothing to install or write for another container.
		plan.Prepare = []Stage{checkWorker, createTaskDirs, stageFiles, loadArtifacts}
		plan.Run = []Stage{initializeOutput, runMain}
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
		plan := planFor(&we.Template, we.layout, we.inputArtifactPluginNames)
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
