// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"os"
	"strings"

	"go.opentelemetry.io/otel/propagation"
)

// Carrier is a TextMapCarrier that uses the environment variables as a
// storage medium for propagated key-value pairs. The keys are uppercased
// before being used to access the environment variables.
// This is useful for propagating values that are set in the environment
// and need to be accessed by different processes or services.
// The keys are uppercased to avoid case sensitivity issues across different
// operating systems and environments.
type Carrier struct {
	// SetEnvFunc is a function that sets the environment variable.
	// Usually, you want to set the environment variable for processes
	// that are spawned by the current process.
	// By default implementation, it does nothing.
	// Using os.Setenv here is discouraged as the environment should
	// be immutable:
	// https://opentelemetry.io/docs/specs/otel/context/env-carriers/#environment-variable-immutability
	SetEnvFunc func(key, value string)
}

// Compile time check that Carrier implements the TextMapCarrier.
var _ propagation.TextMapCarrier = Carrier{}

// Get returns the value associated with the passed key.
// The key is uppercased before being used to access the environment variable.
func (Carrier) Get(key string) string {
	k := strings.ToUpper(key)
	return os.Getenv(k)
}

// Set stores the key-value pair in the environment variable.
// The key is uppercased before being used to set the environment variable.
// If SetEnvFunc is not set, this method does nothing.
func (e Carrier) Set(key, value string) {
	if e.SetEnvFunc == nil {
		return
	}
	k := strings.ToUpper(key)
	e.SetEnvFunc(k, value)
}

// Keys lists the keys stored in this carrier.
// This returns all the keys in the environment variables.
func (Carrier) Keys() []string {
	keys := make([]string, 0, len(os.Environ()))
	for _, kv := range os.Environ() {
		kvPair := strings.SplitN(kv, "=", 2)
		if len(kvPair) < 1 {
			continue
		}
		keys = append(keys, strings.ToLower(kvPair[0]))
	}
	return keys
}
