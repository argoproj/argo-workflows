package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

type ErrorCause string

const (
	nameErrorCount                                   = `error_count`
	ErrorCauseOperationPanic              ErrorCause = "OperationPanic"
	ErrorCauseCronWorkflowSubmissionError ErrorCause = "CronWorkflowSubmissionError"
	ErrorCauseCronWorkflowSpecError       ErrorCause = "CronWorkflowSpecError"
)

func addErrorCounter(ctx context.Context, m *Metrics) error {
	err := m.CreateInstrument(telemetry.Int64Counter,
		nameErrorCount,
		"Number of errors encountered by the controller by cause",
		"{error}",
		telemetry.WithAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	// Initialise all values to zero
	for _, cause := range []ErrorCause{ErrorCauseOperationPanic, ErrorCauseCronWorkflowSubmissionError, ErrorCauseCronWorkflowSpecError} {
		m.AddInt(ctx, nameErrorCount, 0, telemetry.InstAttribs{{Name: telemetry.AttribErrorCause, Value: string(cause)}})
	}
	return nil
}

func (m *Metrics) OperationPanic(ctx context.Context) {
	m.AddInt(ctx, nameErrorCount, 1, telemetry.InstAttribs{{Name: telemetry.AttribErrorCause, Value: string(ErrorCauseOperationPanic)}})
}

func (m *Metrics) CronWorkflowSubmissionError(ctx context.Context) {
	m.AddInt(ctx, nameErrorCount, 1, telemetry.InstAttribs{{Name: telemetry.AttribErrorCause, Value: string(ErrorCauseCronWorkflowSubmissionError)}})
}

func (m *Metrics) CronWorkflowSpecError(ctx context.Context) {
	m.AddInt(ctx, nameErrorCount, 1, telemetry.InstAttribs{{Name: telemetry.AttribErrorCause, Value: string(ErrorCauseCronWorkflowSpecError)}})
}
