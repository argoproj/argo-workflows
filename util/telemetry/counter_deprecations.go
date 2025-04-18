package telemetry

import (
	"context"
)

const (
	nameDeprecated = `deprecated_feature`
)

func AddDeprecationCounter(_ context.Context, m *Metrics) error {
	return m.CreateInstrument(Int64Counter,
		nameDeprecated,
		"Incidents of deprecated feature being used.",
		"{feature}",
		WithAsBuiltIn(),
	)
}

func (m *Metrics) DeprecatedFeature(ctx context.Context, deprecation string, namespace string) {
	attribs := InstAttribs{
		{Name: AttribDeprecatedFeature, Value: deprecation},
	}
	if namespace != "" {
		attribs = append(attribs, InstAttrib{Name: AttribWorkflowNamespace, Value: namespace})
	}
	m.AddInt(ctx, nameDeprecated, 1, attribs)
}
