package telemetry

import (
	"context"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

type Span struct {
	name       string
	children   []*Span
	attributes []BuiltinAttribute
}

type Spans []*Span

// RuntimeName returns the span name as it appears at runtime (lowerCamelCase).
// This matches what's passed to tracer.Start() in the generated code.
func (s *Span) RuntimeName() string {
	return s.name
}

// Children returns the direct children of this span.
func (s *Span) Children() []*Span {
	return s.children
}

// HasChild returns true if this span has a child with the given runtime name.
func (s *Span) HasChild(runtimeName string) bool {
	for _, child := range s.children {
		if child.RuntimeName() == runtimeName {
			return true
		}
	}
	return false
}

// FindDescendant searches recursively for a descendant span with the given runtime name.
// Returns nil if not found.
func (s *Span) FindDescendant(runtimeName string) *Span {
	for _, child := range s.children {
		if child.RuntimeName() == runtimeName {
			return child
		}
		if found := child.FindDescendant(runtimeName); found != nil {
			return found
		}
	}
	return nil
}

// AllDescendantNames returns all descendant span runtime names (depth-first).
func (s *Span) AllDescendantNames() []string {
	var names []string
	s.collectDescendantNames(&names)
	return names
}

func (s *Span) collectDescendantNames(names *[]string) {
	for _, child := range s.children {
		*names = append(*names, child.RuntimeName())
		child.collectDescendantNames(names)
	}
}

// ExpectedChildren returns the runtime names of expected children.
func (s *Span) ExpectedChildren() []string {
	names := make([]string, len(s.children))
	for i, child := range s.children {
		names[i] = child.RuntimeName()
	}
	return names
}

type Tracing struct {
	//	config   *Config
	provider *tracesdk.TracerProvider
	tracer   trace.Tracer
}

func NewTracing(ctx context.Context, serviceName string, extraOpts ...tracesdk.TracerProviderOption) (*Tracing, error) {
	options := make([]tracesdk.TracerProviderOption, 0)
	options = append(options, tracesdk.WithResource(workflowsResource(ctx, serviceName)))
	options = append(options, extraOpts...)
	_, otlpEnabled := os.LookupEnv(`OTEL_EXPORTER_OTLP_ENDPOINT`)
	_, otlpTracingEnabled := os.LookupEnv(`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`)

	if otlpEnabled || otlpTracingEnabled {
		// NOTE: The OTel SDK default changed from gRPC to http/protobuf. For alignment with metrics
		// gRPC is preserved as the default in workflows controller, but http/protobuf can be opted-in
		// to by setting the _PROTOCOL env var explicitly.
		// These env vars match the official SDK: https://opentelemetry.io/docs/languages/sdk-configuration/otlp-exporter/#otel_exporter_otlp_traces_protocol.
		otlpProtocol := os.Getenv(`OTEL_EXPORTER_OTLP_TRACES_PROTOCOL`)
		if otlpProtocol == "" {
			otlpProtocol = os.Getenv(`OTEL_EXPORTER_OTLP_PROTOCOL`)
		}

		logger := logging.RequireLoggerFromContext(ctx)
		endpoint := os.Getenv(`OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`)
		if endpoint == "" {
			endpoint = os.Getenv(`OTEL_EXPORTER_OTLP_ENDPOINT`)
		}

		switch {
		case otlpProtocol == "" || otlpProtocol == "grpc":
			logger.WithFields(logging.Fields{"protocol": "grpc", "endpoint": endpoint}).Info(ctx, "Starting OTLP tracing exporter")
			grpcExporter, err := otlptracegrpc.New(ctx)
			if err != nil {
				return nil, err
			}
			options = append(options, tracesdk.WithBatcher(grpcExporter))
		case strings.HasPrefix(otlpProtocol, "http/"):
			logger.WithFields(logging.Fields{"protocol": "http", "endpoint": endpoint}).Info(ctx, "Starting OTLP tracing exporter")
			httpExporter, err := otlptracehttp.New(ctx)
			if err != nil {
				return nil, err
			}
			options = append(options, tracesdk.WithBatcher(httpExporter))
		default:
			logger.WithFatal().WithField("protocol", otlpProtocol).Error(ctx, "OTEL tracing protocol invalid")
		}
	}

	provider := tracesdk.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)

	tracer := provider.Tracer(serviceName)
	tracing := &Tracing{
		provider: provider,
		tracer:   tracer,
	}
	return tracing, nil
}

// Shutdown flushes any remaining spans and shuts down the trace provider.
func (t *Tracing) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// BreakTrace completely removes the trace from the context
// The next span created will be a top level trace,
// it does not traverse up the span tree to the parent
func (*Tracing) BreakTrace(ctx context.Context) context.Context {
	return trace.ContextWithSpan(ctx, noop.Span{})
}
