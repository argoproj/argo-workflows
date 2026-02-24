package telemetry

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"github.com/argoproj/argo-workflows/v4/util/logging"
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
	otelMeter *metric.Meter
	config    *Config

	// Ensures mutual exclusion in instruments
	mutex       sync.RWMutex
	instruments map[string]*Instrument
}

func (m *Metrics) AddInstrument(name string, inst *Instrument) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.instruments[name] = inst
}

func (m *Metrics) GetInstrument(name string) *Instrument {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	inst, ok := m.instruments[name]
	if !ok {
		return nil
	}
	return inst
}

// IterateROInstruments iterates over every instrument for Read-Only purposes
func (m *Metrics) IterateROInstruments(fn func(i *Instrument)) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	for _, i := range m.instruments {
		fn(i)
	}
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
	logger := logging.RequireLoggerFromContext(ctx)

	if otlpEnabled || otlpMetricsEnabled {
		logger.Info(ctx, "Starting OTLP metrics exporter")

		// NOTE: The OTel SDK default changed from gRPC to http/protobuf. For backwards compatibility,
		// gRPC is preserved as the default in workflows controller, but http/protobuf can be opted-in
		// to by setting the _PROTOCOL env var explicitly.
		// These env vars match the official SDK: https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/#otel_exporter_otlp_metrics_protocol.
		otlpProtocol := os.Getenv(`OTEL_EXPORTER_OTLP_METRICS_PROTOCOL`)
		if otlpProtocol == "" {
			otlpProtocol = os.Getenv(`OTEL_EXPORTER_OTLP_PROTOCOL`)
		}

		switch {
		case otlpProtocol == "" || otlpProtocol == "grpc":
			grpcExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithTemporalitySelector(config.Temporality))
			if err != nil {
				return nil, err
			}
			options = append(options, metricsdk.WithReader(metricsdk.NewPeriodicReader(grpcExporter)))
		case strings.HasPrefix(otlpProtocol, "http/"):
			httpExporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithTemporalitySelector(config.Temporality))
			if err != nil {
				return nil, err
			}
			options = append(options, metricsdk.WithReader(metricsdk.NewPeriodicReader(httpExporter)))
		default:
			logger.WithFatal().WithField("protocol", otlpProtocol).Error(ctx, "OTEL metric protocol invalid")
		}
	}

	if config.Enabled {
		logger.Info(ctx, "Starting Prometheus metrics exporter")
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
		otelMeter:   &meter,
		config:      config,
		instruments: make(map[string]*Instrument),
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
