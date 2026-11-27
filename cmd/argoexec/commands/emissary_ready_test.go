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
	wfexecutor "github.com/argoproj/argo-workflows/v4/workflow/executor"
)

// waitLimit bounds each wait. A passing wait returns as soon as it sees the
// marker, so the limit only matters when something is wrong, and it is long
// because a loaded Windows runner has taken over 8s to see a marker written
// 50ms in. generousTimeout keeps the heartbeat-staleness watchdog well out of
// the way, beyond waitLimit, in tests that exercise the terminal-outcome
// paths; fastPoll keeps them quick.
const (
	waitLimit       = 30 * time.Second
	generousTimeout = time.Minute
	fastPoll        = 10 * time.Millisecond
)

// writeStatusLater writes body to the status marker after 50ms, and sends the
// write's error on the returned channel, so a test can tell a write that
// failed from a wait that didn't see it.
func writeStatusLater(statusPath, body string) <-chan error {
	written := make(chan error, 1)
	go func() {
		time.Sleep(50 * time.Millisecond)
		written <- os.WriteFile(statusPath, []byte(body), 0o644)
	}()
	return written
}

func TestWaitForSupervisorReady_ReadyAppears(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	written := writeStatusLater(statusPath, wfexecutor.StatusReady+"\n")

	waitCtx, cancel := context.WithTimeout(ctx, waitLimit)
	defer cancel()
	err := waitForSupervisorReadyAt(waitCtx, statusPath, generousTimeout, fastPoll)
	require.NoError(t, <-written, "writing the status marker")
	require.NoError(t, err)
}

func TestWaitForSupervisorReady_FailedAppears(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	written := writeStatusLater(statusPath, wfexecutor.StatusFailed+"\nartifact download timeout")

	waitCtx, cancel := context.WithTimeout(ctx, waitLimit)
	defer cancel()
	err := waitForSupervisorReadyAt(waitCtx, statusPath, generousTimeout, fastPoll)
	require.NoError(t, <-written, "writing the status marker")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "artifact download timeout")
}

func TestWaitForSupervisorReady_AlreadyReady(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	require.NoError(t, os.WriteFile(statusPath, []byte(wfexecutor.StatusReady+"\n"), 0o644))

	waitCtx, cancel := context.WithTimeout(ctx, waitLimit)
	defer cancel()
	require.NoError(t, waitForSupervisorReadyAt(waitCtx, statusPath, generousTimeout, fastPoll))
}

func TestWaitForSupervisorReady_AlreadyFailed(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	require.NoError(t, os.WriteFile(statusPath, []byte(wfexecutor.StatusFailed+"\nplugin sidecar timeout"), 0o644))

	waitCtx, cancel := context.WithTimeout(ctx, waitLimit)
	defer cancel()
	err := waitForSupervisorReadyAt(waitCtx, statusPath, generousTimeout, fastPoll)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plugin sidecar timeout")
}

// TestWaitForSupervisorReady_RunningThenReady exercises the heartbeat happy
// path: a RUNNING marker keeps main waiting (a generous timeout means staleness
// never fires), and the later terminal READY write releases it.
func TestWaitForSupervisorReady_RunningThenReady(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	require.NoError(t, os.WriteFile(statusPath, []byte(wfexecutor.StatusRunning+"\n"), 0o644))
	written := writeStatusLater(statusPath, wfexecutor.StatusReady+"\n")

	waitCtx, cancel := context.WithTimeout(ctx, waitLimit)
	defer cancel()
	err := waitForSupervisorReadyAt(waitCtx, statusPath, generousTimeout, fastPoll)
	require.NoError(t, <-written, "writing the status marker")
	require.NoError(t, err)
}

// TestWaitForSupervisorReady_StaleRunningPresumedDead covers a supervisor that
// wrote RUNNING then died without ever writing a terminal status: the marker's
// mtime goes stale past the timeout and main fails fast instead of hanging.
func TestWaitForSupervisorReady_StaleRunningPresumedDead(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	require.NoError(t, os.WriteFile(statusPath, []byte(wfexecutor.StatusRunning+"\n"), 0o644))
	// Backdate the marker so it is already stale relative to the timeout below.
	old := time.Now().Add(-time.Hour)
	require.NoError(t, os.Chtimes(statusPath, old, old))

	waitCtx, cancel := context.WithTimeout(ctx, waitLimit)
	defer cancel()
	err := waitForSupervisorReadyAt(waitCtx, statusPath, 50*time.Millisecond, fastPoll)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "presumed dead")
}

// TestWaitForSupervisorReady_NeverAppearsPresumedDead covers a supervisor that
// died before writing any marker: main bounds the wait by the timeout rather
// than blocking to the pod deadline.
func TestWaitForSupervisorReady_NeverAppearsPresumedDead(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	waitCtx, cancel := context.WithTimeout(ctx, waitLimit)
	defer cancel()
	err := waitForSupervisorReadyAt(waitCtx, statusPath, 50*time.Millisecond, fastPoll)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "never appeared")
}

func TestWaitForSupervisorReady_ContextCancelled(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	// Staleness timeout is generous so the parent-context cancellation, not the
	// dead-supervisor path, is what ends the wait.
	waitCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	err := waitForSupervisorReadyAt(waitCtx, statusPath, generousTimeout, fastPoll)
	require.Error(t, err)
}

// TestWaitForSupervisorReady_ParentCancelledPropagates guards the regression
// where a parent-context cancellation (e.g. pod termination) was swallowed to
// nil and mistaken for a ready supervisor. With no marker present, an explicit
// cancel must surface as an error, not success.
func TestWaitForSupervisorReady_ParentCancelledPropagates(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), logging.NewTestLogger(logging.Info, logging.Text))
	dir := t.TempDir()
	statusPath := filepath.Join(dir, "status")

	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // the marker will never appear
	err := waitForSupervisorReadyAt(cancelCtx, statusPath, generousTimeout, fastPoll)
	require.Error(t, err, "parent cancellation must not be reported as ready")
}
