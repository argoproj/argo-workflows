package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShouldExecuteBareLiterals verifies that shouldExecute correctly handles
// fully-substituted when expressions where bare words are string literals.
// After template substitution, "{{item.evenness}} == even" becomes "odd == even".
// shouldExecute converts VARIABLE tokens to STRING tokens, so "odd" != "even" → false.
func TestShouldExecuteBareLiterals(t *testing.T) {
	tests := []struct {
		name     string
		when     string
		expected bool
	}{
		{
			name:     "different bare words are not equal",
			when:     "odd == even",
			expected: false,
		},
		{
			name:     "same bare words are equal",
			when:     "even == even",
			expected: true,
		},
		{
			name:     "bare word equals quoted string",
			when:     "even == 'even'",
			expected: true,
		},
		{
			name:     "bare word not-equal check",
			when:     "odd != even",
			expected: true,
		},
		{
			name:     "empty when clause",
			when:     "",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := shouldExecute(tc.when)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, res, "shouldExecute(%q)", tc.when)
		})
	}
}
