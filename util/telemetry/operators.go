package telemetry

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (m *Metrics) AddInt(ctx context.Context, name string, val int64, attribs InstAttribs) {
	if instrument := m.GetInstrument(name); instrument != nil {
		instrument.AddInt(ctx, val, attribs)
	} else {
		log.Errorf("Metrics addInt() to non-existent metric %s", name)
	}
}

func (i *Instrument) AddInt(ctx context.Context, val int64, attribs InstAttribs) {
	switch inst := i.otel.(type) {
	case *metric.Int64UpDownCounter:
		(*inst).Add(ctx, val, i.attributes(attribs))
	case *metric.Int64Counter:
		(*inst).Add(ctx, val, i.attributes(attribs))
	default:
		log.Errorf("Metrics addInt() to invalid type %s (%t)", i.name, i.otel)
	}
}

func (m *Metrics) Record(ctx context.Context, name string, val float64, attribs InstAttribs) {
	if instrument := m.GetInstrument(name); instrument != nil {
		instrument.Record(ctx, val, attribs)
	} else {
		log.Errorf("Metrics record() to non-existent metric %s", name)
	}
}

func (i *Instrument) Record(ctx context.Context, val float64, attribs InstAttribs) {
	switch inst := i.otel.(type) {
	case *metric.Float64Histogram:
		(*inst).Record(ctx, val, i.attributes(attribs))
	default:
		log.Errorf("Metrics record() to invalid type %s (%t)", i.name, i.otel)
	}
}

func (i *Instrument) RegisterCallback(m *Metrics, f metric.Callback) error {
	switch inst := i.otel.(type) {
	case *metric.Float64ObservableUpDownCounter:
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

func (i *Instrument) ObserveInt(o metric.Observer, val int64, attribs InstAttribs) {
	switch inst := i.otel.(type) {
	case *metric.Int64ObservableGauge:
		o.ObserveInt64(*inst, val, i.attributes(attribs))
	default:
		log.Errorf("Metrics observeFloat() to invalid type %s (%t)", i.name, i.otel)
	}
}

func (i *Instrument) ObserveFloat(o metric.Observer, val float64, attribs InstAttribs) {
	switch inst := i.otel.(type) {
	case *metric.Float64ObservableGauge:
		o.ObserveFloat64(*inst, val, i.attributes(attribs))
	case *metric.Float64ObservableUpDownCounter:
		o.ObserveFloat64(*inst, val, i.attributes(attribs))
	default:
		log.Errorf("Metrics observeFloat() to invalid type %s (%t)", i.name, i.otel)
	}
}

type InstAttribs []InstAttrib
type InstAttrib struct {
	Name  string
	Value interface{}
}

func (i *Instrument) attributes(labels InstAttribs) metric.MeasurementOption {
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
			log.Errorf("Attempt to use label of unhandled type in metric %s", i.name)
		}
	}
	return metric.WithAttributes(attribs...)
}
