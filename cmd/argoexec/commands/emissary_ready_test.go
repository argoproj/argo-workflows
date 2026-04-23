package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestWaitForSupervisorReady_ReadyAppears(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "ready")
	failedPath := filepath.Join(dir, "failed")

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = os.WriteFile(readyPath, nil, 0o644)
	}()

	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	require.NoError(t, waitForSupervisorReadyAt(waitCtx, readyPath, failedPath))
}

func TestWaitForSupervisorReady_FailedAppears(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "ready")
	failedPath := filepath.Join(dir, "failed")

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = os.WriteFile(failedPath, []byte("artifact download timeout"), 0o644)
	}()

	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := waitForSupervisorReadyAt(waitCtx, readyPath, failedPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "artifact download timeout")
}

func TestWaitForSupervisorReady_AlreadyReady(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "ready")
	failedPath := filepath.Join(dir, "failed")

	require.NoError(t, os.WriteFile(readyPath, nil, 0o644))

	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	require.NoError(t, waitForSupervisorReadyAt(waitCtx, readyPath, failedPath))
}

func TestWaitForSupervisorReady_AlreadyFailed(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "ready")
	failedPath := filepath.Join(dir, "failed")

	require.NoError(t, os.WriteFile(failedPath, []byte("plugin sidecar timeout"), 0o644))

	waitCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	err := waitForSupervisorReadyAt(waitCtx, readyPath, failedPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plugin sidecar timeout")
}

func TestWaitForSupervisorReady_ContextCancelled(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "ready")
	failedPath := filepath.Join(dir, "failed")

	waitCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	err := waitForSupervisorReadyAt(waitCtx, readyPath, failedPath)
	require.Error(t, err)
}

// TestWaitForSupervisorReady_ParentCancelledPropagates guards the regression
// where a parent-context cancellation (e.g. pod termination) was swallowed to
// nil and mistaken for a ready supervisor. With neither marker present, an
// explicit cancel must surface as an error, not success.
func TestWaitForSupervisorReady_ParentCancelledPropagates(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.InitLogger())
	dir := t.TempDir()
	readyPath := filepath.Join(dir, "ready")
	failedPath := filepath.Join(dir, "failed")

	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // neither marker will ever appear
	err := waitForSupervisorReadyAt(cancelCtx, readyPath, failedPath)
	require.Error(t, err, "parent cancellation must not be reported as ready")
}
