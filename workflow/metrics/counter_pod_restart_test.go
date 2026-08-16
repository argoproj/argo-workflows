package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractConditionFromMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "DiskPressure condition",
			message:  "The node had condition: [DiskPressure]",
			expected: "DiskPressure",
		},
		{
			name:     "MemoryPressure condition",
			message:  "The node had condition: [MemoryPressure]",
			expected: "MemoryPressure",
		},
		{
			name:     "PIDPressure condition",
			message:  "The node had condition: [PIDPressure]",
			expected: "PIDPressure",
		},
		{
			name:     "no condition in message",
			message:  "Pod was evicted",
			expected: "",
		},
		{
			name:     "empty message",
			message:  "",
			expected: "",
		},
		{
			name:     "multiple conditions takes first",
			message:  "The node had condition: [DiskPressure] and [MemoryPressure]",
			expected: "DiskPressure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractConditionFromMessage(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}
