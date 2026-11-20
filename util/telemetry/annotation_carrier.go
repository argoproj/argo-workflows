package telemetry

import (
	"strings"

	"go.opentelemetry.io/otel/propagation"
)

// AnnotationCarrier is a TextMapCarrier over Kubernetes object annotations.
// Each propagated field is stored under its own annotation, named Prefix
// followed by the field name, e.g. "opentelemetry.io/traceparent".
type AnnotationCarrier struct {
	Prefix      string
	Annotations map[string]string
}

// Compile time check that AnnotationCarrier implements the TextMapCarrier.
var _ propagation.TextMapCarrier = AnnotationCarrier{}

// Get returns the value of the annotation for the passed key.
func (c AnnotationCarrier) Get(key string) string {
	return c.Annotations[c.Prefix+strings.ToLower(key)]
}

// Set stores the key-value pair as an annotation.
// It does nothing if the carrier has no annotation map.
func (c AnnotationCarrier) Set(key, value string) {
	if c.Annotations == nil {
		return
	}
	c.Annotations[c.Prefix+strings.ToLower(key)] = value
}

// Keys lists the keys stored in this carrier, with the prefix removed.
func (c AnnotationCarrier) Keys() []string {
	keys := make([]string, 0)
	for k := range c.Annotations {
		if key, ok := strings.CutPrefix(k, c.Prefix); ok && key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}
