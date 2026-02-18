package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func workflowsResource(ctx context.Context, serviceName string) *resource.Resource {
	argoversion := argo.GetVersion()
	attribs := []attribute.KeyValue{
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(argoversion.Version),
	}

	res, err := resource.New(
		ctx,
		resource.WithFromEnv(),      // Discover and provide attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
		resource.WithTelemetrySDK(), // Discover and provide information about the OpenTelemetry SDK used.
		resource.WithProcess(),      // Discover and provide process information.
		resource.WithOS(),           // Discover and provide OS information.
		resource.WithContainer(),    // Discover and provide container information.
		resource.WithHost(),         // Discover and provide host information.
		resource.WithAttributes(attribs...),
	)
	if err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "Error from opentelemetry resource detection, carrying on anyway")
	}
	return res
}
