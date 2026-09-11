package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func workflowsResource(ctx context.Context, serviceName string) *resource.Resource {
	res, err := resource.New(
		ctx,
		resource.WithSchemaURL(semconv.SchemaURL),
		// Set the static attributes first, so they can be overridden by the environment.
		resource.WithAttributes(semconv.ServiceName(serviceName)),
		// Discover and provide attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables.
		resource.WithFromEnv(),
	)
	if err != nil {
		logging.RequireLoggerFromContext(ctx).WithError(err).Error(ctx, "Error from opentelemetry resource detection, carrying on anyway")
	}
	return res
}
