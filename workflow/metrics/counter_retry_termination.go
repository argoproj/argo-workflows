package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

// RetryTerminationReason is why a retryStrategy duration budget stopped an
// otherwise eligible retry. The set is closed and controller-defined: the
// attribute feeds a counter, so an unbounded value here would be a cardinality
// bug rather than extra detail.
type RetryTerminationReason string

const (
	// RetryTerminationMaxDurationExceeded means backoff.maxDuration had already
	// elapsed when the controller came to schedule the next attempt.
	RetryTerminationMaxDurationExceeded RetryTerminationReason = "MaxDurationExceeded"
	// RetryTerminationBackoffWouldExceedMaxDuration means the attempt was still
	// within the budget, but waiting out the next back-off would cross it.
	RetryTerminationBackoffWouldExceedMaxDuration RetryTerminationReason = "BackoffWouldExceedMaxDuration"
)

func addRetryStrategyTerminationsCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentRetryStrategyTerminationsTotal)
}

// RetryStrategyTerminated records one retry suppressed by a duration budget.
//
// Callers must only invoke this when the retry would otherwise have gone ahead.
// A retry that its policy, the node's own retryability, limit or expression
// would have rejected anyway is not a duration-budget termination, even though
// the duration check happens to run first — counting it would attribute the
// stop to whichever check the controller evaluated earliest rather than to the
// reason the retry did not happen.
func (m *Metrics) RetryStrategyTerminated(ctx context.Context, reason RetryTerminationReason, namespace string) {
	m.AddRetryStrategyTerminationsTotal(ctx, 1, string(reason), namespace)
}
