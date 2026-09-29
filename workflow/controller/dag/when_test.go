package dag

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldExecute(t *testing.T) {
	trueExpressions := []string{
		"foo == foo",
		"'ref/branch/master' == 'ref/branch/master'",
		"foo != bar",
		"1 == 1",
		"1 != 2",
		"1 < 2",
		"1 <= 1",
		"1/2 == 0.5",
		"a < b",
		"(foo == bar) || (foo == foo)",
		"(1 > 0) && (1 < 2)",
		"Error in (Failed, Error)",
		"!(Succeeded in (Failed, Error))",
		"true == true",
	}
	for _, trueExp := range trueExpressions {
		res, err := ShouldExecute(trueExp)
		require.NoError(t, err)
		assert.True(t, res)
	}

	falseExpressions := []string{
		"foo != foo",
		"'ref/branch/master' != 'ref/branch/master'",
		"foo == bar",
		"1 != 1",
		"1 == 2",
		"1 > 2",
		"1 <= 0",
		"1/2 != 0.5",
		"a > b",
		"(foo == bar) || (bar == foo)",
		"(1 > 0) && (11 < 2)",
		"Succeeded in (Failed, Error)",
		"!(Error in (Failed, Error))",
		"false == true",
	}
	for _, falseExp := range falseExpressions {
		res, err := ShouldExecute(falseExp)
		require.NoError(t, err)
		assert.False(t, res)
	}
}

// TestShouldExecuteBareLiterals verifies that ShouldExecute correctly handles
// fully-substituted when expressions where bare words are string literals.
// After template substitution, "{{item.evenness}} == even" becomes "odd == even".
// ShouldExecute converts VARIABLE tokens to STRING tokens, so "odd" != "even" → false.
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
			res, err := ShouldExecute(tc.when)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, res, "ShouldExecute(%q)", tc.when)
		})
	}
}

// TestShouldExecuteInvalidWhenHint verifies the invalid-when error message
// keeps main's full hint, including the closing quote (C83): moving this
// evaluator into engine.go in 8d178132c dropped the ` ("))` suffix.
func TestShouldExecuteInvalidWhenHint(t *testing.T) {
	_, err := ShouldExecute("heads == @tails")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `(hint: try wrapping the affected expression in quotes ("))`)
}
