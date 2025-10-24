package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

type ErrorCause string

const (
	ErrorCauseOperationPanic              ErrorCause = "OperationPanic"
	ErrorCauseCronWorkflowSubmissionError ErrorCause = "CronWorkflowSubmissionError"
	ErrorCauseCronWorkflowSpecError       ErrorCause = "CronWorkflowSpecError"
)

func addErrorCounter(ctx context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentErrorCount)
	if err != nil {
		return err
	}
	// Initialise all values to zero
	for _, cause := range []ErrorCause{ErrorCauseOperationPanic, ErrorCauseCronWorkflowSubmissionError, ErrorCauseCronWorkflowSpecError} {
		m.AddErrorCount(ctx, 0, string(cause))
	}
	return nil
}

func (m *Metrics) OperationPanic(ctx context.Context) {
	m.AddErrorCount(ctx, 1, string(ErrorCauseOperationPanic))
}

func (m *Metrics) CronWorkflowSubmissionError(ctx context.Context) {
	m.AddErrorCount(ctx, 1, string(ErrorCauseCronWorkflowSubmissionError))
}

func (m *Metrics) CronWorkflowSpecError(ctx context.Context) {
	m.AddErrorCount(ctx, 1, string(ErrorCauseCronWorkflowSpecError))
}
