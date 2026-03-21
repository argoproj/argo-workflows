package tracing

import (
	"net/http"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// s3AttributeTransport adds S3-specific attributes to spans created by otelhttp
type s3AttributeTransport struct {
	base http.RoundTripper
}

func (t *s3AttributeTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	operation, bucket, key := parseS3Request(r)

	// Add S3-specific attributes directly to the span created by otelhttp
	span := trace.SpanFromContext(r.Context())
	span.SetAttributes(
		attribute.String("s3.operation", operation),
		attribute.String("s3.bucket", bucket),
	)
	if key != "" {
		span.SetAttributes(attribute.String("s3.key", key))
	}

	return t.base.RoundTrip(r)
}

// parseS3Request extracts S3 operation, bucket, and key from the request
func parseS3Request(r *http.Request) (operation, bucket, key string) {
	// Determine operation from HTTP method and query params
	switch r.Method {
	case http.MethodGet:
		if r.URL.RawQuery == "" || strings.Contains(r.URL.RawQuery, "prefix=") {
			operation = "GetObject"
			if strings.Contains(r.URL.RawQuery, "prefix=") {
				operation = "ListObjects"
			}
		} else {
			operation = "GetObject"
		}
	case http.MethodPut:
		operation = "PutObject"
	case http.MethodDelete:
		operation = "DeleteObject"
	case http.MethodHead:
		operation = "HeadObject"
	case http.MethodPost:
		operation = "PostObject"
	default:
		operation = r.Method
	}

	// Parse bucket and key from path
	// Path format is typically: /bucket/key or /bucket
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) > 0 {
		bucket = parts[0]
	}
	if len(parts) > 1 {
		key = parts[1]
	}

	return operation, bucket, key
}

// WrapS3Transport wraps an HTTP transport with otelhttp tracing
// and S3-specific attribute extraction.
func WrapS3Transport(rt http.RoundTripper) http.RoundTripper {
	if rt == nil {
		rt = http.DefaultTransport
	}
	// First wrap with our transport to add S3-specific attributes,
	// then wrap with otelhttp. This order ensures otelhttp's span
	// context is available when our RoundTrip runs.
	s3Transport := &s3AttributeTransport{base: rt}
	return otelhttp.NewTransport(s3Transport)
}
