package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// resourceAttributes collects the resource attributes as a map for easy assertion
func resourceAttributes(t *testing.T) map[string]string {
	t.Helper()
	res := workflowsResource(logging.TestContext(t.Context()), "test-service")
	attribs := make(map[string]string)
	for _, kv := range res.Attributes() {
		attribs[string(kv.Key)] = kv.Value.AsString()
	}
	return attribs
}

func TestResourceDefaultServiceName(t *testing.T) {
	assert.Equal(t, "test-service", resourceAttributes(t)[string(semconv.ServiceNameKey)])
}

func TestResourceServiceNameFromEnv(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "from-service-name")
	assert.Equal(t, "from-service-name", resourceAttributes(t)[string(semconv.ServiceNameKey)])
}

func TestResourceServiceNameFromResourceAttributes(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=from-resource-attributes,deployment.environment=production")
	attribs := resourceAttributes(t)
	assert.Equal(t, "from-resource-attributes", attribs[string(semconv.ServiceNameKey)])
	assert.Equal(t, "production", attribs["deployment.environment"])
}

// OTEL_SERVICE_NAME takes precedence over service.name in OTEL_RESOURCE_ATTRIBUTES
func TestResourceServiceNamePrecedence(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "from-service-name")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=from-resource-attributes")
	assert.Equal(t, "from-service-name", resourceAttributes(t)[string(semconv.ServiceNameKey)])
}

func TestResourceSchemaURL(t *testing.T) {
	res := workflowsResource(logging.TestContext(t.Context()), "test-service")
	assert.Equal(t, semconv.SchemaURL, res.SchemaURL())
}
