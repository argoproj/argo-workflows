package telemetry

import (
	"fmt"
	"sort"

	"go.opentelemetry.io/otel/metric"

	"github.com/argoproj/argo-workflows/v3/util/help"
)

type Instrument struct {
	name        string
	description string
	otel        any
	userdata    any
}

func (m *Metrics) preCreateCheck(name string) error {
	if inst := m.GetInstrument(name); inst != nil {
		return fmt.Errorf("Instrument called %s already exists", name)
	}
	return nil
}

func addHelpLink(name, description string) string {
	return fmt.Sprintf("%s %s", description, help.MetricHelp(name))
}

type instrumentType int

const (
	Float64ObservableGauge instrumentType = iota
	Float64Histogram
	Float64ObservableCounter
	Int64ObservableGauge
	Int64UpDownCounter
	Int64Counter
)

// InstrumentOption applies options to all instruments.
type instrumentOptions struct {
	builtIn        bool
	defaultBuckets []float64
}

// InstrumentOption is a functional option for configuring instruments.
type InstrumentOption func(*instrumentOptions)

func WithAsBuiltIn() InstrumentOption {
	return func(o *instrumentOptions) {
		o.builtIn = true
	}
}

func WithDefaultBuckets(buckets []float64) InstrumentOption {
	return func(o *instrumentOptions) {
		o.defaultBuckets = buckets
	}
}

func collectOptions(options ...InstrumentOption) instrumentOptions {
	var o instrumentOptions
	for _, opt := range options {
		opt(&o)
	}
	return o
}

func (m *Metrics) CreateInstrument(instType instrumentType, name, desc, unit string, options ...InstrumentOption) error {
	opts := collectOptions(options...)
	err := m.preCreateCheck(name)
	if err != nil {
		return err
	}

	if opts.builtIn {
		desc = addHelpLink(name, desc)
	}
	var instPtr any
	switch instType {
	case Float64ObservableGauge:
		inst, insterr := (*m.otelMeter).Float64ObservableGauge(name,
			metric.WithDescription(desc),
			metric.WithUnit(unit),
		)
		instPtr = &inst
		err = insterr
	case Float64Histogram:
		inst, insterr := (*m.otelMeter).Float64Histogram(name,
			metric.WithDescription(desc),
			metric.WithUnit(unit),
			metric.WithExplicitBucketBoundaries(m.buckets(name, opts.defaultBuckets)...),
		)
		instPtr = &inst
		err = insterr
	case Float64ObservableCounter:
		inst, insterr := (*m.otelMeter).Float64ObservableCounter(name,
			metric.WithDescription(desc),
			metric.WithUnit(unit),
		)
		instPtr = &inst
		err = insterr
	case Int64ObservableGauge:
		inst, insterr := (*m.otelMeter).Int64ObservableGauge(name,
			metric.WithDescription(desc),
			metric.WithUnit(unit),
		)
		instPtr = &inst
		err = insterr
	case Int64UpDownCounter:
		inst, insterr := (*m.otelMeter).Int64UpDownCounter(name,
			metric.WithDescription(desc),
			metric.WithUnit(unit),
		)
		instPtr = &inst
		err = insterr
	case Int64Counter:
		inst, insterr := (*m.otelMeter).Int64Counter(name,
			metric.WithDescription(desc),
			metric.WithUnit(unit),
		)
		instPtr = &inst
		err = insterr
	default:
		return fmt.Errorf("internal error creating metric instrument of unknown type %v", instType)
	}
	if err != nil {
		return err
	}
	m.AddInstrument(name, &Instrument{
		name:        name,
		description: desc,
		otel:        instPtr,
	})
	return nil
}

func (m *Metrics) buckets(name string, defaultBuckets []float64) []float64 {
	if opts, ok := m.config.Modifiers[name]; ok {
		if len(opts.HistogramBuckets) > 0 {
			buckets := opts.HistogramBuckets
			sort.Float64s(buckets)
			return buckets
		}
	}
	return defaultBuckets
}

func (i *Instrument) GetName() string {
	return i.name
}

func (i *Instrument) GetDescription() string {
	return i.description
}

func (i *Instrument) GetOtel() any {
	return i.otel
}

func (i *Instrument) SetUserdata(data any) {
	i.userdata = data
}

func (i *Instrument) GetUserdata() any {
	return i.userdata
}
