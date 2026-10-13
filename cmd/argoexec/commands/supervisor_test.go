package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestStatusMarker_RunningHeartbeat(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	require.NoError(t, writeRunningStatusAt(statusPath))

	body, err := os.ReadFile(statusPath)
	require.NoError(t, err)
	token, _ := parseSupervisorStatus(body)
	assert.Equal(t, statusRunning, token)
	assert.NoFileExists(t, statusPath+".tmp", "tmp file should be renamed away")
}

func TestStatusMarker_Success(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	require.NoError(t, writeSuccessStatusAt(statusPath))

	body, err := os.ReadFile(statusPath)
	require.NoError(t, err)
	token, _ := parseSupervisorStatus(body)
	assert.Equal(t, statusReady, token)
	assert.NoFileExists(t, statusPath+".tmp", "tmp file should be renamed away")
}

func TestStatusMarker_FailureCapturesCause(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	writeFailureStatusAt(ctx, statusPath, errors.New("boom"))

	body, err := os.ReadFile(statusPath)
	require.NoError(t, err)
	token, message := parseSupervisorStatus(body)
	assert.Equal(t, statusFailed, token)
	assert.Equal(t, "boom", message)
	assert.NoFileExists(t, statusPath+".tmp", "tmp file should be renamed away")
}

func TestStatusMarker_FailureWithEmptyCauseStillSignalsFailure(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	// A cause that stringifies to "" must still produce the FAILED token, so
	// the emissary can't misread the failure as success.
	writeFailureStatusAt(ctx, statusPath, errors.New(""))

	body, err := os.ReadFile(statusPath)
	require.NoError(t, err)
	token, _ := parseSupervisorStatus(body)
	assert.Equal(t, statusFailed, token)
}

func TestStatusMarker_BestEffortOnUnwritable(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	// Path under a directory that doesn't exist — write fails, but the
	// helper must not panic; supervisor's pre-main error still propagates
	// via PostMain even if the marker write itself fails.
	writeFailureStatusAt(ctx, "/does/not/exist/status", errors.New("boom"))
}
