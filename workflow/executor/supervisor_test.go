package executor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestStatusMarker_RunningHeartbeat(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	require.NoError(t, writeRunningStatus(statusPath))

	body, err := os.ReadFile(statusPath)
	require.NoError(t, err)
	token, _ := ParseStatusMarker(body)
	assert.Equal(t, StatusRunning, token)
	assert.NoFileExists(t, statusPath+".tmp", "tmp file should be renamed away")
}

func TestStatusMarker_Success(t *testing.T) {
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	require.NoError(t, writeSuccessStatus(statusPath))

	body, err := os.ReadFile(statusPath)
	require.NoError(t, err)
	token, _ := ParseStatusMarker(body)
	assert.Equal(t, StatusReady, token)
	assert.NoFileExists(t, statusPath+".tmp", "tmp file should be renamed away")
}

func TestStatusMarker_FailureCapturesCause(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	writeFailureStatus(ctx, statusPath, errors.New("boom"))

	body, err := os.ReadFile(statusPath)
	require.NoError(t, err)
	token, message := ParseStatusMarker(body)
	assert.Equal(t, StatusFailed, token)
	assert.Equal(t, "boom", message)
	assert.NoFileExists(t, statusPath+".tmp", "tmp file should be renamed away")
}

func TestStatusMarker_FailureWithEmptyCauseStillSignalsFailure(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	// A cause that stringifies to "" must still produce the FAILED token, so
	// the emissary can't misread the failure as success.
	writeFailureStatus(ctx, statusPath, errors.New(""))

	body, err := os.ReadFile(statusPath)
	require.NoError(t, err)
	token, _ := ParseStatusMarker(body)
	assert.Equal(t, StatusFailed, token)
}

func TestStatusMarker_BestEffortOnUnwritable(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	// Path under a directory that doesn't exist — write fails, but the
	// helper must not panic; supervisor's pre-main error still propagates
	// via PostMain even if the marker write itself fails.
	writeFailureStatus(ctx, "/does/not/exist/status", errors.New("boom"))
}

// The controller and emissary docs name /var/run/argo/status; the pod's
// marker must stay there.
func TestStatusMarkerPath_Pod(t *testing.T) {
	assert.Equal(t, filepath.Join(common.VarRunArgoPath, "status"), StatusMarkerPath(PodPaths()))
}
