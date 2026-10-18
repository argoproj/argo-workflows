package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// resourceAttributes collects the detected resource attributes into a map for easy assertion
func resourceAttributes(t *testing.T) map[attribute.Key]string {
	t.Helper()
	res := workflowsResource(logging.TestContext(t.Context()), "test-service")
	attribs := make(map[attribute.Key]string)
	for _, kv := range res.Attributes() {
		attribs[kv.Key] = kv.Value.String()
	}
	return attribs
}

func TestResourceDefaults(t *testing.T) {
	attribs := resourceAttributes(t)
	assert.Equal(t, "test-service", attribs[semconv.ServiceNameKey])
	assert.NotEmpty(t, attribs[semconv.ServiceVersionKey])
}

func TestResourceServiceNameFromEnv(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "from-service-name")
	assert.Equal(t, "from-service-name", resourceAttributes(t)[semconv.ServiceNameKey])
}

func TestResourceServiceNameFromResourceAttributes(t *testing.T) {
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=from-resource-attributes,deployment.environment=production")
	attribs := resourceAttributes(t)
	assert.Equal(t, "from-resource-attributes", attribs[semconv.ServiceNameKey])
	assert.Equal(t, "production", attribs["deployment.environment"])
}

// OTEL_SERVICE_NAME takes precedence over service.name in OTEL_RESOURCE_ATTRIBUTES
func TestResourceServiceNamePrecedence(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "from-service-name")
	t.Setenv("OTEL_RESOURCE_ATTRIBUTES", "service.name=from-resource-attributes")
	assert.Equal(t, "from-service-name", resourceAttributes(t)[semconv.ServiceNameKey])
}

func TestResourceDetectors(t *testing.T) {
	attribs := resourceAttributes(t)
	assert.Equal(t, "opentelemetry", attribs[semconv.TelemetrySDKNameKey])
	assert.Contains(t, attribs, semconv.HostNameKey)
	assert.Contains(t, attribs, semconv.OSTypeKey)
	assert.Contains(t, attribs, semconv.ProcessPIDKey)
	assert.Equal(t, "go", attribs[semconv.ProcessRuntimeNameKey])
	// WithProcessOwner is deliberately left out: it needs cgo or $USER, which
	// the distroless images have neither of, so it errors on every startup
	assert.NotContains(t, attribs, semconv.ProcessOwnerKey)
}
