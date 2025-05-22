package telemetry

//import ()

type BuiltinAttribute struct {
	name     string
	optional bool
}

type BuiltinInstrument struct {
	name           string
	description    string
	unit           string
	instType       instrumentType
	attributes     []BuiltinAttribute
	defaultBuckets []float64
}

func (bi *BuiltinInstrument) Name() string {
	return bi.name
}

// CreateBuiltinInstrument adds a yaml defined builtin instrument
// opts parameter is for legacy metrics, do not use for new metrics
func (m *Metrics) CreateBuiltinInstrument(instrument BuiltinInstrument, opts ...instrumentOption) error {
	opts = append(opts, WithAsBuiltIn())
	if len(instrument.defaultBuckets) > 0 {
		opts = append(opts,
			WithDefaultBuckets(instrument.defaultBuckets))
	}
	return m.CreateInstrument(instrument.instType,
		instrument.name,
		instrument.description,
		instrument.unit,
		opts...,
	)
}
