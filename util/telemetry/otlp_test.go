package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResolveOTLPEndpoint verifies endpoint precedence and handling of empty values.
func TestResolveOTLPEndpoint(t *testing.T) {
	tests := []struct {
		name                   string
		commonEndpoint         string
		signalSpecificEndpoint string
		expectedEndpoint       string
	}{
		{
			name: "empty values",
		},
		{
			name:             "common endpoint",
			commonEndpoint:   "common:4317",
			expectedEndpoint: "common:4317",
		},
		{
			name:                   "signal-specific endpoint",
			signalSpecificEndpoint: "signal:4317",
			expectedEndpoint:       "signal:4317",
		},
		{
			name:                   "signal-specific endpoint takes precedence",
			commonEndpoint:         "common:4317",
			signalSpecificEndpoint: "signal:4317",
			expectedEndpoint:       "signal:4317",
		},
		{
			name:                   "whitespace-only values",
			commonEndpoint:         " ",
			signalSpecificEndpoint: "\t",
		},
		{
			name:                   "endpoints with surrounding whitespace",
			commonEndpoint:         " common:4317 ",
			signalSpecificEndpoint: " signal:4317 ",
			expectedEndpoint:       "signal:4317",
		},
		{
			name:                   "whitespace-only signal-specific endpoint falls back to common endpoint",
			commonEndpoint:         "common:4317",
			signalSpecificEndpoint: " ",
			expectedEndpoint:       "common:4317",
		},
	}

	for _, signalEndpointEnv := range []string{otlpMetricsEndpointEnv, otlpTracesEndpointEnv} {
		t.Run(signalEndpointEnv, func(t *testing.T) {
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					t.Setenv(otlpEndpointEnv, test.commonEndpoint)
					t.Setenv(signalEndpointEnv, test.signalSpecificEndpoint)

					require.Equal(t, test.expectedEndpoint, resolveOTLPEndpoint(signalEndpointEnv))
				})
			}
		})
	}
}
