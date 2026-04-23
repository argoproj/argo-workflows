package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestInputArtifactPluginNames_Empty(t *testing.T) {
	t.Setenv(common.EnvVarInputArtifactPluginNames, "")
	assert.Empty(t, inputArtifactPluginNames())
}

func TestInputArtifactPluginNames_Single(t *testing.T) {
	t.Setenv(common.EnvVarInputArtifactPluginNames, "s3-driver")
	names := inputArtifactPluginNames()
	require.Len(t, names, 1)
	assert.Equal(t, "s3-driver", string(names[0]))
}

func TestInputArtifactPluginNames_Multi(t *testing.T) {
	t.Setenv(common.EnvVarInputArtifactPluginNames, "s3-driver, gcs-driver ,  oci ")
	names := inputArtifactPluginNames()
	require.Len(t, names, 3)
	assert.Equal(t, "s3-driver", string(names[0]))
	assert.Equal(t, "gcs-driver", string(names[1]))
	assert.Equal(t, "oci", string(names[2]))
}

func TestInputArtifactPluginNames_SkipsBlankEntries(t *testing.T) {
	t.Setenv(common.EnvVarInputArtifactPluginNames, "s3,,gcs, ,oci")
	names := inputArtifactPluginNames()
	require.Len(t, names, 3)
}

func TestInputArtifactPluginNames_TrailingComma(t *testing.T) {
	t.Setenv(common.EnvVarInputArtifactPluginNames, "s3-driver,")
	names := inputArtifactPluginNames()
	require.Len(t, names, 1)
	assert.Equal(t, "s3-driver", string(names[0]))
}

func TestReadyMarker_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "ready")

	require.NoError(t, writeReadyMarkerAt(readyPath))

	assert.FileExists(t, readyPath)
	assert.NoFileExists(t, readyPath+".tmp", "tmp file should be renamed away")
}

func TestFailedMarker_CapturesCause(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	failedPath := filepath.Join(dir, "failed")

	writeFailedMarkerAt(ctx, failedPath, errors.New("boom"))

	body, err := os.ReadFile(failedPath)
	require.NoError(t, err)
	assert.Equal(t, "boom", string(body))
	assert.NoFileExists(t, failedPath+".tmp", "tmp file should be renamed away")
}

func TestFailedMarker_BestEffortOnUnwritable(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	// Path under a directory that doesn't exist — write fails, but the
	// helper must not panic; supervisor's pre-main error still propagates
	// via PostMain even if the marker write itself fails.
	writeFailedMarkerAt(ctx, "/does/not/exist/failed", errors.New("boom"))
}

// fakeStages is a controllable preMainStages implementation. Each
// function field, if non-nil, is invoked; if nil, the method returns nil.
// Counters and the loadStarted/loadCancelled channels let tests assert
// orchestration order and errgroup cancellation behavior.
type fakeStages struct {
	writeTemplate             func() error
	stageFiles                func(ctx context.Context) error
	loadWithoutPlugins        func(ctx context.Context) error
	loadFromPlugin            func(ctx context.Context, name wfv1.ArtifactPluginName) error
	writeTemplateCalls        atomic.Int32
	stageFilesCalls           atomic.Int32
	loadWithoutPluginsCalls   atomic.Int32
	loadFromPluginCalls       atomic.Int32
	loadFromPluginNamesMu     sync.Mutex
	loadFromPluginNames       []wfv1.ArtifactPluginName
	loadFromPluginCancelledMu sync.Mutex
	loadFromPluginCancelled   []wfv1.ArtifactPluginName
}

func (f *fakeStages) WriteTemplate() error {
	f.writeTemplateCalls.Add(1)
	if f.writeTemplate != nil {
		return f.writeTemplate()
	}
	return nil
}

func (f *fakeStages) StageFiles(ctx context.Context) error {
	f.stageFilesCalls.Add(1)
	if f.stageFiles != nil {
		return f.stageFiles(ctx)
	}
	return nil
}

func (f *fakeStages) LoadArtifactsWithoutPlugins(ctx context.Context) error {
	f.loadWithoutPluginsCalls.Add(1)
	if f.loadWithoutPlugins != nil {
		return f.loadWithoutPlugins(ctx)
	}
	return nil
}

func (f *fakeStages) LoadArtifactsFromPlugin(ctx context.Context, name wfv1.ArtifactPluginName) error {
	f.loadFromPluginCalls.Add(1)
	f.loadFromPluginNamesMu.Lock()
	f.loadFromPluginNames = append(f.loadFromPluginNames, name)
	f.loadFromPluginNamesMu.Unlock()
	if f.loadFromPlugin != nil {
		err := f.loadFromPlugin(ctx, name)
		if errors.Is(ctx.Err(), context.Canceled) {
			f.loadFromPluginCancelledMu.Lock()
			f.loadFromPluginCancelled = append(f.loadFromPluginCancelled, name)
			f.loadFromPluginCancelledMu.Unlock()
		}
		return err
	}
	return nil
}

func TestRunSupervisorPreMain_HappyPath(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	stages := &fakeStages{}

	err := runSupervisorPreMain(ctx, stages, []wfv1.ArtifactPluginName{"s3", "gcs"})
	require.NoError(t, err)
	assert.EqualValues(t, 1, stages.writeTemplateCalls.Load())
	assert.EqualValues(t, 1, stages.stageFilesCalls.Load())
	assert.EqualValues(t, 1, stages.loadWithoutPluginsCalls.Load())
	assert.EqualValues(t, 2, stages.loadFromPluginCalls.Load())
}

func TestRunSupervisorPreMain_NoPlugins(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	stages := &fakeStages{}

	err := runSupervisorPreMain(ctx, stages, nil)
	require.NoError(t, err)
	assert.EqualValues(t, 1, stages.loadWithoutPluginsCalls.Load())
	assert.EqualValues(t, 0, stages.loadFromPluginCalls.Load())
}

func TestRunSupervisorPreMain_WriteTemplateFailsShortCircuits(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	stages := &fakeStages{writeTemplate: func() error { return errors.New("write fail") }}

	err := runSupervisorPreMain(ctx, stages, []wfv1.ArtifactPluginName{"s3"})
	require.ErrorContains(t, err, "failed to write template")
	// Subsequent stages must not run.
	assert.EqualValues(t, 0, stages.stageFilesCalls.Load())
	assert.EqualValues(t, 0, stages.loadWithoutPluginsCalls.Load())
	assert.EqualValues(t, 0, stages.loadFromPluginCalls.Load())
}

func TestRunSupervisorPreMain_StageFilesFailsShortCircuits(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	stages := &fakeStages{stageFiles: func(ctx context.Context) error { return errors.New("stage fail") }}

	err := runSupervisorPreMain(ctx, stages, []wfv1.ArtifactPluginName{"s3"})
	require.ErrorContains(t, err, "failed to stage files")
	assert.EqualValues(t, 1, stages.writeTemplateCalls.Load())
	assert.EqualValues(t, 0, stages.loadWithoutPluginsCalls.Load())
}

func TestRunSupervisorPreMain_OnePluginFailureCancelsSiblings(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// "fail" plugin returns an error immediately. Other goroutines block
	// on ctx.Done() — they must observe cancellation and return.
	stages := &fakeStages{
		loadWithoutPlugins: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		loadFromPlugin: func(ctx context.Context, name wfv1.ArtifactPluginName) error {
			if name == "fail" {
				return errors.New("plugin boom")
			}
			<-ctx.Done()
			return ctx.Err()
		},
	}

	done := make(chan error, 1)
	go func() { done <- runSupervisorPreMain(ctx, stages, []wfv1.ArtifactPluginName{"fail", "ok1", "ok2"}) }()
	select {
	case err := <-done:
		require.ErrorContains(t, err, "plugin boom",
			"the failing plugin's error should surface; sibling cancellations should not mask it")
	case <-time.After(5 * time.Second):
		t.Fatal("runSupervisorPreMain did not return; siblings did not observe context cancellation")
	}
	// All three plugin goroutines must have been entered.
	assert.EqualValues(t, 3, stages.loadFromPluginCalls.Load())
}

func TestRunSupervisorPreMain_NonPluginLoadFailure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	stages := &fakeStages{
		loadWithoutPlugins: func(ctx context.Context) error { return errors.New("non-plugin boom") },
	}

	err := runSupervisorPreMain(ctx, stages, nil)
	require.ErrorContains(t, err, "non-plugin boom")
}
