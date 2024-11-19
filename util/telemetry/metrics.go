package telemetry

import (
	"context"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type Config struct {
	Enabled      bool
	Path         string
	Port         int
	TTL          time.Duration
	IgnoreErrors bool
	Secure       bool
	Modifiers    map[string]Modifier
	Temporality  metricsdk.TemporalitySelector
}

type Metrics struct {
	// Ensures mutual exclusion in workflows map
	Mutex sync.RWMutex

	// Evil context for compatibility with legacy context free interfaces
	Ctx       context.Context
	otelMeter *metric.Meter
	config    *Config

	AllInstruments map[string]*Instrument
}

func NewMetrics(ctx context.Context, serviceName, prometheusName string, config *Config, extraOpts ...metricsdk.Option) (*Metrics, error) {
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)

	options := make([]metricsdk.Option, 0)
	options = append(options, metricsdk.WithResource(res))
	_, otlpEnabled := os.LookupEnv(`OTEL_EXPORTER_OTLP_ENDPOINT`)
	_, otlpMetricsEnabled := os.LookupEnv(`OTEL_EXPORTER_OTLP_METRICS_ENDPOINT`)
	if otlpEnabled || otlpMetricsEnabled {
		log.Info("Starting OTLP metrics exporter")
		otelExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithTemporalitySelector(config.Temporality))
		if err != nil {
			return nil, err
		}
		options = append(options, metricsdk.WithReader(metricsdk.NewPeriodicReader(otelExporter)))
	}

	if config.Enabled {
		log.Info("Starting Prometheus metrics exporter")
		promExporter, err := config.prometheusMetricsExporter(prometheusName)
		if err != nil {
			return nil, err
		}
		options = append(options, metricsdk.WithReader(promExporter))
	}
	options = append(options, extraOpts...)
	options = append(options, view(config))

	provider := metricsdk.NewMeterProvider(options...)
	otel.SetMeterProvider(provider)

	// Add runtime metrics
	err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
	if err != nil {
		return nil, err
	}

	meter := provider.Meter(serviceName)
	metrics := &Metrics{
		Ctx:            ctx,
		otelMeter:      &meter,
		config:         config,
		AllInstruments: make(map[string]*Instrument),
	}

	return metrics, nil
}

type AddMetric func(context.Context, *Metrics) error

func (m *Metrics) Populate(ctx context.Context, adders ...AddMetric) error {
	for _, adder := range adders {
		if err := adder(ctx, m); err != nil {
			return err
		}
	}
	return nil
}
