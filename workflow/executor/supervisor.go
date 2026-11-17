package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Status marker protocol between the supervisor (writer) and the emissary in
// main (reader), exchanged via the task's status marker (StatusMarkerPath) in
// the shared directory. The first line of the marker is one of these tokens;
// for StatusFailed the remaining lines carry the failure message.
const (
	StatusRunning = "RUNNING" // pre-main in progress; rewritten as a heartbeat
	StatusReady   = "READY"   // pre-main succeeded; main may proceed
	StatusFailed  = "FAILED"  // pre-main failed; message follows on later lines

	// SupervisorHeartbeatInterval is how often the supervisor rewrites the
	// status marker while pre-main runs. Each rewrite advances the marker's
	// mtime, which is what proves liveness to main.
	SupervisorHeartbeatInterval = 5 * time.Second
)

// StatusMarkerPath is where the supervisor writes the task's status marker.
func StatusMarkerPath(paths TaskPaths) string {
	return filepath.Join(paths.VarRunArgo, "status")
}

// ParseStatusMarker splits the marker into its first-line token and the
// remaining message (used by the FAILED token to carry the cause).
func ParseStatusMarker(body []byte) (token, message string) {
	first, rest, _ := strings.Cut(string(body), "\n")
	return strings.TrimSpace(first), strings.TrimSpace(rest)
}

// startStatusHeartbeat writes an initial RUNNING status, then rewrites it every
// SupervisorHeartbeatInterval on a background goroutine until the returned stop
// function is called. Each rewrite advances the marker's mtime, which main's
// emissary uses to distinguish a live (but slow) supervisor from a dead one.
//
// The initial write is synchronous so a broken shared mount fails fast (its
// error is returned). stop() cancels the goroutine and blocks until it has
// exited, guaranteeing no heartbeat write can race the terminal status write
// that follows it.
func startStatusHeartbeat(ctx context.Context, path string) (stop func(), err error) {
	if err := writeRunningStatus(path); err != nil {
		return nil, fmt.Errorf("failed to write initial status marker: %w", err)
	}
	hbCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(SupervisorHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				if err := writeRunningStatus(path); err != nil {
					logging.RequireLoggerFromContext(ctx).WithError(err).Warn(ctx, "failed to write status heartbeat")
				}
			}
		}
	}()
	return func() { cancel(); <-done }, nil
}

// writeRunningStatus rewrites the status marker at path as RUNNING (the
// heartbeat).
func writeRunningStatus(path string) error {
	return writeStatusMarker(path, []byte(StatusRunning+"\n"))
}

// writeSuccessStatus signals to the emissary in main that pre-main setup is
// complete by writing the terminal READY token.
func writeSuccessStatus(path string) error {
	return writeStatusMarker(path, []byte(StatusReady+"\n"))
}

// writeFailureStatus records a pre-main failure: the terminal FAILED token
// followed by the cause. Best-effort — if the write fails, main keeps waiting
// and eventually hits the heartbeat-staleness timeout.
func writeFailureStatus(ctx context.Context, path string, cause error) {
	body := StatusFailed
	if msg := cause.Error(); msg != "" {
		body += "\n" + msg
	}
	if err := writeStatusMarker(path, []byte(body)); err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "failed to write failure status marker")
	}
}

// writeStatusMarker writes body to path via write-then-rename so a reader
// watching path (via inotify) only ever observes the fully-written file — it
// never sees a torn terminal message. We deliberately do not fsync — the
// marker lives in an emptyDir and the pod is gone if the node crashes before
// the write hits disk.
func writeStatusMarker(path string, body []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return fmt.Errorf("failed to write marker tmp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename marker %s: %w", path, err)
	}
	return nil
}
