package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func (m *Metrics) AddInt(ctx context.Context, name string, val int64, attribs Attributes) {
	if instrument := m.GetInstrument(name); instrument != nil {
		instrument.AddInt(ctx, val, attribs)
	} else {
		logging.RequireLoggerFromContext(ctx).WithField("name", name).Error(ctx, "Metrics addInt() to non-existent metric")
	}
}

func (i *Instrument) AddInt(ctx context.Context, val int64, attribs Attributes) {
	switch inst := i.otel.(type) {
	case *metric.Int64UpDownCounter:
		(*inst).Add(ctx, val, i.attributes(ctx, attribs))
	case *metric.Int64Counter:
		(*inst).Add(ctx, val, i.attributes(ctx, attribs))
	default:
		logging.RequireLoggerFromContext(ctx).WithField("name", i.name).WithField("type", i.otel).Error(ctx, "Metrics addInt() to invalid type")
	}
}

func (m *Metrics) Record(ctx context.Context, name string, val float64, attribs Attributes) {
	if instrument := m.GetInstrument(name); instrument != nil {
		instrument.Record(ctx, val, attribs)
	} else {
		logging.RequireLoggerFromContext(ctx).WithField("name", name).Error(ctx, "Metrics record() to non-existent metric")
	}
}

func (i *Instrument) Record(ctx context.Context, val float64, attribs Attributes) {
	switch inst := i.otel.(type) {
	case *metric.Float64Histogram:
		(*inst).Record(ctx, val, i.attributes(ctx, attribs))
	default:
		logging.RequireLoggerFromContext(ctx).WithField("name", i.name).WithField("type", i.otel).Error(ctx, "Metrics record() to invalid type")
	}
}

func (i *Instrument) RegisterCallback(m *Metrics, f metric.Callback) error {
	switch inst := i.otel.(type) {
	case *metric.Float64ObservableCounter:
		_, err := (*m.otelMeter).RegisterCallback(f, *inst)
		return err
	case *metric.Float64ObservableGauge:
		_, err := (*m.otelMeter).RegisterCallback(f, *inst)
		return err
	case *metric.Int64ObservableGauge:
		_, err := (*m.otelMeter).RegisterCallback(f, *inst)
		return err
	default:
		return fmt.Errorf("Metrics registerCallback() to invalid type %s (%t)", i.name, i.otel)
	}
}

func (i *Instrument) ObserveInt(ctx context.Context, o metric.Observer, val int64, attribs Attributes) {
	switch inst := i.otel.(type) {
	case *metric.Int64ObservableGauge:
		o.ObserveInt64(*inst, val, i.attributes(ctx, attribs))
	default:
		logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{
			"name": i.name,
			"type": i.otel,
		}).Error(ctx, "Metrics observeInt() to invalid type")
	}
}

func (i *Instrument) ObserveFloat(ctx context.Context, o metric.Observer, val float64, attribs Attributes) {
	switch inst := i.otel.(type) {
	case *metric.Float64ObservableGauge:
		o.ObserveFloat64(*inst, val, i.attributes(ctx, attribs))
	case *metric.Float64ObservableCounter:
		o.ObserveFloat64(*inst, val, i.attributes(ctx, attribs))
	default:
		logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{
			"name": i.name,
			"type": i.otel,
		}).Error(ctx, "Metrics observeFloat() to invalid type")
	}
}

type InstAttribs []InstAttrib
type InstAttrib struct {
	Name  string
	Value any
}

func (i *Instrument) attributes(ctx context.Context, labels Attributes) metric.MeasurementOption {
	attribs := make([]attribute.KeyValue, 0)
	for _, label := range labels {
		switch value := label.Value.(type) {
		case string:
			attribs = append(attribs, attribute.String(label.Name, value))
		case bool:
			attribs = append(attribs, attribute.Bool(label.Name, value))
		case int:
			attribs = append(attribs, attribute.Int(label.Name, value))
		case int64:
			attribs = append(attribs, attribute.Int64(label.Name, value))
		case float64:
			attribs = append(attribs, attribute.Float64(label.Name, value))
		default:
			logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{
				"name": i.name,
				"type": label.Value,
			}).Error(ctx, "Attempt to use label of unhandled type in metric")
		}
	}
	return metric.WithAttributes(attribs...)
}
