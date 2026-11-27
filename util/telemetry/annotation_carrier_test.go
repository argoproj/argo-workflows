package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnnotationCarrier(t *testing.T) {
	annotations := map[string]string{
		"opentelemetry.io/traceparent": "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
		"opentelemetry.io/":            "ignored",
		"other.io/traceparent":         "ignored",
	}
	carrier := AnnotationCarrier{Prefix: "opentelemetry.io/", Annotations: annotations}

	assert.Equal(t, "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01", carrier.Get("traceparent"))
	assert.Equal(t, "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01", carrier.Get("Traceparent"))
	assert.Empty(t, carrier.Get("tracestate"))
	assert.Equal(t, []string{"traceparent"}, carrier.Keys())

	carrier.Set("Baggage", "k=v")
	assert.Equal(t, "k=v", annotations["opentelemetry.io/baggage"])
}

func TestAnnotationCarrierNilMap(t *testing.T) {
	carrier := AnnotationCarrier{Prefix: "opentelemetry.io/"}
	assert.Empty(t, carrier.Get("traceparent"))
	assert.Empty(t, carrier.Keys())
	carrier.Set("traceparent", "x") // must not panic
}
