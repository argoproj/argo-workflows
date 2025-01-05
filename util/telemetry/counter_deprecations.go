package telemetry

import (
	"context"
)

func AddDeprecationCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(InstrumentDeprecatedFeature)
}

func (m *Metrics) DeprecatedFeature(ctx context.Context, deprecation string, namespace string) {
	attribs := InstAttribs{
		{Name: AttribDeprecatedFeature, Value: deprecation},
	}
	if namespace != "" {
		attribs = append(attribs, InstAttrib{Name: AttribWorkflowNamespace, Value: namespace})
	}
	m.AddInt(ctx, InstrumentDeprecatedFeature.Name(), 1, attribs)
}
