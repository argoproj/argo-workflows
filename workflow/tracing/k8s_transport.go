package tracing

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v4/util/k8s"
)

// k8sAttributeTransport adds k8s-specific attributes to spans created by otelhttp
type k8sAttributeTransport struct {
	base http.RoundTripper
}

func (t *k8sAttributeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	verb, kind := k8s.ParseRequest(r)

	// Add k8s-specific attributes directly to the span created by otelhttp
	span := trace.SpanFromContext(r.Context())
	span.SetAttributes(
		attribute.String("k8s.verb", verb),
		attribute.String("k8s.kind", kind),
	)

	return t.base.RoundTrip(r)
}

// AddTracingTransportWrapper wraps the k8s client transport with otelhttp
// to trace all Kubernetes API calls with semconv-compliant span names
// and k8s-specific attributes.
func AddTracingTransportWrapper(_ context.Context, config *rest.Config) {
	wrap := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if wrap != nil {
			rt = wrap(rt)
		}
		// First wrap with our transport to add k8s-specific attributes,
		// then wrap with otelhttp. This order ensures otelhttp's span
		// context is available when our RoundTrip runs.
		k8sTransport := &k8sAttributeTransport{base: rt}
		return otelhttp.NewTransport(k8sTransport)
	}
}
