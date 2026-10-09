package executor

import (
	"context"
	"strings"

	"golang.org/x/sync/errgroup"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// This file is the registry of every stage argoexec can run. It is the one
// place to read to answer "does this already exist": a Stage literal
// anywhere else in the package fails TestStageLiteralsLiveInRegistry.
// Each stage is a thin wrapper over one WorkflowExecutor method; the
// executor methods stay the unit of work, stages are the unit of
// composition.

// Prepare-phase stages.
var (
	// installArgoexec copies the argoexec binary into the shared volume and
	// writes the template: the legacy init container's job. The init-less
	// layout delivers the binary via an image volume and needs only
	// writeTemplate.
	installArgoexec = Stage{Name: "install-argoexec", Run: func(_ context.Context, we *WorkflowExecutor) error {
		return we.Init()
	}}
	writeTemplate = Stage{Name: "write-template", Run: func(_ context.Context, we *WorkflowExecutor) error {
		return we.WriteTemplate()
	}}
	stageFiles = Stage{Name: "stage-files", Run: func(ctx context.Context, we *WorkflowExecutor) error {
		return we.StageFiles(ctx)
	}}
	loadArtifacts = Stage{Name: "load-artifacts", Run: func(ctx context.Context, we *WorkflowExecutor) error {
		return we.LoadArtifactsWithoutPlugins(ctx)
	}}
)

// loadArtifactsFromPlugin loads the input artifacts served by one artifact
// plugin; the plan adds one of these per plugin, run in parallel.
func loadArtifactsFromPlugin(name wfv1.ArtifactPluginName) Stage {
	return Stage{Name: "load-artifacts-from-plugin(" + string(name) + ")", Run: func(ctx context.Context, we *WorkflowExecutor) error {
		return we.LoadArtifactsFromPlugin(ctx, name)
	}}
}

// Run-phase stages.
var (
	initializeOutput = Stage{Name: "initialize-output", Run: func(ctx context.Context, we *WorkflowExecutor) error {
		we.InitializeOutput(ctx)
		return nil
	}}
	waitMain = Stage{Name: "wait", Run: func(ctx context.Context, we *WorkflowExecutor) error {
		return we.Wait(ctx)
	}}
)

// Collect-phase stages. save-artifacts and save-logs hand what they saved
// to report-outputs through the task's savedArtifacts.
var (
	captureScriptResult = Stage{Name: "capture-script-result", NeedsMainOutputs: true, Run: func(ctx context.Context, we *WorkflowExecutor) error {
		return we.CaptureScriptResult(ctx)
	}}
	saveParameters = Stage{Name: "save-parameters", NeedsMainOutputs: true, Run: func(ctx context.Context, we *WorkflowExecutor) error {
		return we.SaveParameters(ctx)
	}}
	saveArtifacts = Stage{Name: "save-artifacts", NeedsMainOutputs: true, Run: func(ctx context.Context, we *WorkflowExecutor) error {
		artifacts, err := we.SaveArtifacts(ctx)
		we.savedArtifacts = artifacts
		return err
	}}
	saveLogs = Stage{Name: "save-logs", Run: func(ctx context.Context, we *WorkflowExecutor) error {
		we.savedArtifacts = append(we.savedArtifacts, we.SaveLogs(ctx)...)
		return nil
	}}
	reportOutputs = Stage{Name: "report-outputs", Run: func(ctx context.Context, we *WorkflowExecutor) error {
		return we.ReportOutputs(ctx, we.savedArtifacts)
	}}
	// reportLogs is the resource template's whole Collect phase: its output
	// parameters were already reported by `argoexec resource`, and a full
	// report-outputs would overwrite them with the template's empty values.
	reportLogs = Stage{Name: "report-logs", Run: func(ctx context.Context, we *WorkflowExecutor) error {
		return we.ReportOutputsLogs(ctx)
	}}
)

// Combinators.

// Parallel returns a stage that runs its children concurrently and returns
// the first error, cancelling the siblings' context.
func Parallel(stages ...Stage) Stage {
	names := make([]string, len(stages))
	for i, s := range stages {
		names[i] = s.Name
	}
	return Stage{Name: "parallel(" + strings.Join(names, ",") + ")", Run: func(ctx context.Context, we *WorkflowExecutor) error {
		g, gctx := errgroup.WithContext(ctx)
		for _, s := range stages {
			g.Go(func() error { return s.Run(gctx, we) })
		}
		return g.Wait()
	}}
}
