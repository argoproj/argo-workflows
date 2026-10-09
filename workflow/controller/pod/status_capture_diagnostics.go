package pod

import (
	"context"
	"errors"
	"sync"

	"k8s.io/utils/lru"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// StatusCaptureHoldReason identifies an actionable barrier without making a
// diagnostic itself an authorization to remove it.
type StatusCaptureHoldReason string

const (
	CaptureResultPending      StatusCaptureHoldReason = "result_pending"
	CaptureDataUnavailable    StatusCaptureHoldReason = "data_unavailable"
	CaptureLegacyUnsupported  StatusCaptureHoldReason = "legacy_proof_unavailable"
	CaptureIdentityConflict   StatusCaptureHoldReason = "identity_conflict"
	CaptureResultConflict     StatusCaptureHoldReason = "result_conflict"
	CaptureReceiptNotRetained StatusCaptureHoldReason = "receipt_not_retained"
	cleanupDiagnosticCapacity                         = 1024
)

type statusCaptureHoldError struct {
	reason StatusCaptureHoldReason
	detail string
}

func (e *statusCaptureHoldError) Error() string { return e.detail }

// StatusCaptureHold retains a cleanup obligation with a specific diagnostic.
func StatusCaptureHold(reason StatusCaptureHoldReason, detail string) error {
	return &statusCaptureHoldError{reason: reason, detail: detail}
}

// Diagnostics are bounded and disposable. Losing or evicting an entry can only
// repeat a warning; scheduling and cleanup permission never depend on this map.
type cleanupDiagnostics struct {
	mu     sync.Mutex
	recent *lru.Cache
}

func (d *cleanupDiagnostics) changed(key, reason string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.recent == nil {
		d.recent = lru.New(cleanupDiagnosticCapacity)
	}
	previous, ok := d.recent.Get(key)
	d.recent.Add(key, reason)
	return !ok || previous != reason
}

func (d *cleanupDiagnostics) forget(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.recent != nil {
		d.recent.Remove(key)
	}
}

func (c *Controller) reportCleanupHold(ctx context.Context, key string, log logging.Logger, err error) {
	reason := CaptureDataUnavailable
	if hold, ok := errors.AsType[*statusCaptureHoldError](err); ok {
		reason = hold.reason
	}
	log = log.WithField("captureReason", reason).WithError(err)
	if c.cleanupDiagnostics.changed(key, string(reason)+":"+err.Error()) {
		log.Warn(ctx, "pod cleanup is waiting")
	} else {
		log.Debug(ctx, "pod cleanup is still waiting")
	}
}
