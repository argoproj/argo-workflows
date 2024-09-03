package metrics

import (
	"context"
)

type ErrorCause string

const (
	nameErrorCount                                   = `error_count`
	ErrorCauseOperationPanic              ErrorCause = "OperationPanic"
	ErrorCauseCronWorkflowSubmissionError ErrorCause = "CronWorkflowSubmissionError"
	ErrorCauseCronWorkflowSpecError       ErrorCause = "CronWorkflowSpecError"
)

func addErrorCounter(ctx context.Context, m *Metrics) error {
	err := m.createInstrument(int64Counter,
		nameErrorCount,
		"Number of errors encountered by the controller by cause",
		"{error}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	// Initialise all values to zero
	for _, cause := range []ErrorCause{ErrorCauseOperationPanic, ErrorCauseCronWorkflowSubmissionError, ErrorCauseCronWorkflowSpecError} {
		m.addInt(ctx, nameErrorCount, 0, instAttribs{{name: labelErrorCause, value: string(cause)}})
	}
	return nil
}

func (m *Metrics) OperationPanic(ctx context.Context) {
	m.addInt(ctx, nameErrorCount, 1, instAttribs{{name: labelErrorCause, value: string(ErrorCauseOperationPanic)}})
}

func (m *Metrics) CronWorkflowSubmissionError(ctx context.Context) {
	m.addInt(ctx, nameErrorCount, 1, instAttribs{{name: labelErrorCause, value: string(ErrorCauseCronWorkflowSubmissionError)}})
}

func (m *Metrics) CronWorkflowSpecError(ctx context.Context) {
	m.addInt(ctx, nameErrorCount, 1, instAttribs{{name: labelErrorCause, value: string(ErrorCauseCronWorkflowSpecError)}})
}
