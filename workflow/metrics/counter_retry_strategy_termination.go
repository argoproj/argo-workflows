package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

// RetryStrategyTerminationReason is a bounded duration-budget reason that
// suppressed an otherwise eligible retry attempt.
type RetryStrategyTerminationReason string

const (
	// Keep these values controller-defined and bounded. Future retry budgets can
	// add another reason without introducing a separate metric.
	RetryStrategyTerminationReasonMaxDurationExceeded           RetryStrategyTerminationReason = "MaxDurationExceeded"
	RetryStrategyTerminationReasonBackoffWouldExceedMaxDuration RetryStrategyTerminationReason = "BackoffWouldExceedMaxDuration"
	RetryStrategyTerminationReasonMaxExecutionDurationExceeded  RetryStrategyTerminationReason = "MaxExecutionDurationExceeded"
)

func addRetryStrategyTerminationsCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentRetryStrategyTerminationsTotal)
}

// RecordRetryStrategyTermination records that a duration budget suppressed an
// otherwise eligible retry attempt.
func (m *Metrics) RecordRetryStrategyTermination(ctx context.Context, reason RetryStrategyTerminationReason, namespace string) {
	m.AddRetryStrategyTerminationsTotal(ctx, 1, string(reason), namespace)
}
