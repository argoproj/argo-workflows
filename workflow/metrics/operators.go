package metrics

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (m *Metrics) addInt(ctx context.Context, name string, val int64, labels instAttribs) {
	if instrument, ok := m.allInstruments[name]; ok {
		instrument.addInt(ctx, val, labels)
	} else {
		log.Errorf("Metrics addInt() to non-existent metric %s", name)
	}
}

func (i *instrument) addInt(ctx context.Context, val int64, labels instAttribs) {
	switch inst := i.otel.(type) {
	case *metric.Int64UpDownCounter:
		(*inst).Add(ctx, val, i.attributes(labels))
	case *metric.Int64Counter:
		(*inst).Add(ctx, val, i.attributes(labels))
	default:
		log.Errorf("Metrics addInt() to invalid type %s (%t)", i.name, i.otel)
	}
}

func (m *Metrics) record(ctx context.Context, name string, val float64, labels instAttribs) {
	if instrument, ok := m.allInstruments[name]; ok {
		instrument.record(ctx, val, labels)
	} else {
		log.Errorf("Metrics record() to non-existent metric %s", name)
	}
}

func (i *instrument) record(ctx context.Context, val float64, labels instAttribs) {
	switch inst := i.otel.(type) {
	case *metric.Float64Histogram:
		(*inst).Record(ctx, val, i.attributes(labels))
	default:
		log.Errorf("Metrics record() to invalid type %s (%t)", i.name, i.otel)
	}
}

func (i *instrument) registerCallback(m *Metrics, f metric.Callback) error {
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

func (i *instrument) observeInt(o metric.Observer, val int64, labels instAttribs) {
	switch inst := i.otel.(type) {
	case *metric.Int64ObservableGauge:
		o.ObserveInt64(*inst, val, i.attributes(labels))
	default:
		log.Errorf("Metrics observeFloat() to invalid type %s (%t)", i.name, i.otel)
	}
}

func (i *instrument) observeFloat(o metric.Observer, val float64, labels instAttribs) {
	switch inst := i.otel.(type) {
	case *metric.Float64ObservableGauge:
		o.ObserveFloat64(*inst, val, i.attributes(labels))
	case *metric.Float64ObservableUpDownCounter:
		o.ObserveFloat64(*inst, val, i.attributes(labels))
	default:
		log.Errorf("Metrics observeFloat() to invalid type %s (%t)", i.name, i.otel)
	}
}

type instAttribs []instAttrib
type instAttrib struct {
	name  string
	value interface{}
}

func (i *instrument) attributes(labels instAttribs) metric.MeasurementOption {
	attribs := make([]attribute.KeyValue, 0)
	for _, label := range labels {
		switch value := label.value.(type) {
		case string:
			attribs = append(attribs, attribute.String(label.name, value))
		case bool:
			attribs = append(attribs, attribute.Bool(label.name, value))
		case int:
			attribs = append(attribs, attribute.Int(label.name, value))
		case int64:
			attribs = append(attribs, attribute.Int64(label.name, value))
		case float64:
			attribs = append(attribs, attribute.Float64(label.name, value))
		default:
			log.Errorf("Attempt to use label of unhandled type in metric %s", i.name)
		}
	}
	return metric.WithAttributes(attribs...)
}
