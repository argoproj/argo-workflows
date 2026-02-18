package telemetry

import (
	"slices"

	apiv1 "k8s.io/api/core/v1"

	"go.opentelemetry.io/otel/propagation"
)

// EnvvarCarrier is a TextMapCarrier that uses an apiv1.EnvVar list held in memory as a storage
// medium for propagated key-value pairs.
type EnvvarCarrier struct {
	EnvVars *[]apiv1.EnvVar
}

// Compile time check that MapCarrier implements the TextMapCarrier.
var _ propagation.TextMapCarrier = EnvvarCarrier{}

// Get returns the value associated with the passed key.
func (c EnvvarCarrier) Get(key string) string {
	i := slices.IndexFunc(*c.EnvVars, func(envVar apiv1.EnvVar) bool {
		return envVar.Name == toEnvKey(key)
	})
	if i >= 0 {
		return (*c.EnvVars)[i].Value
	}
	return ""
}

// Set stores the key-value pair.
func (c EnvvarCarrier) Set(key, value string) {
	i := slices.IndexFunc(*c.EnvVars, func(envVar apiv1.EnvVar) bool {
		return envVar.Name == toEnvKey(key)
	})
	if i >= 0 {
		(*c.EnvVars)[i].Value = value
		return
	}
	*c.EnvVars = append(*c.EnvVars, apiv1.EnvVar{
		Name:  toEnvKey(key),
		Value: value,
	})
}

// Keys lists the keys stored in this carrier.
func (c EnvvarCarrier) Keys() []string {
	vals := make([]string, 0)
	for _, envVar := range *c.EnvVars {
		if isEnvKey(envVar.Name) {
			vals = append(vals, fromEnvKey(envVar.Name))
		}
	}
	return vals
}
