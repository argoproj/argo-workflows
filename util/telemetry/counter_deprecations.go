package telemetry

import (
	"context"
)

func AddDeprecationCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(InstrumentDeprecatedFeature)
}

func (m *Metrics) DeprecatedFeature(ctx context.Context, deprecation string, namespace string) {
	if namespace != "" {
		m.AddDeprecatedFeature(ctx, 1, deprecation, WithWorkflowNamespace(namespace))
	} else {
		m.AddDeprecatedFeature(ctx, 1, deprecation)
	}
}
