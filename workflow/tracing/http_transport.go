package tracing

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// artifactAttributeTransport adds artifact-specific attributes to spans
type artifactAttributeTransport struct {
	base         http.RoundTripper
	artifactType string
}

func (t *artifactAttributeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	span := trace.SpanFromContext(r.Context())
	span.SetAttributes(
		attribute.String("artifact.type", t.artifactType),
	)
	return t.base.RoundTrip(r)
}

// WrapHTTPArtifactClient wraps an http.Client's transport with otelhttp tracing
// and adds artifact-specific attributes.
// If the client is nil, returns a new traced client.
// If the client's transport is nil, uses http.DefaultTransport.
func WrapHTTPArtifactClient(client *http.Client, artifactType string) *http.Client {
	if client == nil {
		client = &http.Client{}
	}
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	// First wrap with our transport to add artifact attributes,
	// then wrap with otelhttp for standard HTTP tracing
	artifactTransport := &artifactAttributeTransport{base: transport, artifactType: artifactType}
	client.Transport = otelhttp.NewTransport(artifactTransport)
	return client
}
