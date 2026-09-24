package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestReplaceWithEmoji(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	replaceMap := map[string]string{
		"inputs.parameters.flag": "🏴󠁧󠁢󠁳󠁣󠁴󠁿",
	}

	test := toJSONString(`{{inputs.parameters.flag}}`)
	replacement, err := Replace(ctx, test, replaceMap, false)

	require.NoError(t, err, "Should not error on emoji substitution")
	assert.Equal(t, toJSONString("🏴󠁧󠁢󠁳󠁣󠁴󠁿"), replacement, "Should preserve emoji character")
}

func TestReplaceWithSpecialCharacters(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Emoji flag",
			input:    "🏴󠁧󠁢󠁳󠁣󠁴󠁿",
			expected: "🏴󠁧󠁢󠁳󠁣󠁴󠁿",
		},
		{
			name:     "Unicode emoji",
			input:    "Hello 👋 World",
			expected: "Hello 👋 World",
		},
		{
			name:     "Chinese characters",
			input:    "你好世界",
			expected: "你好世界",
		},
		{
			name:     "Arabic characters",
			input:    "مرحبا بالعالم",
			expected: "مرحبا بالعالم",
		},
		{
			name:     "Mixed content",
			input:    "Test 测试 🚀",
			expected: "Test 测试 🚀",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			replaceMap := map[string]string{
				"inputs.parameters.value": tc.input,
			}
			test := toJSONString(`{{inputs.parameters.value}}`)
			replacement, err := Replace(ctx, test, replaceMap, false)

			require.NoError(t, err, "Should not error on special character substitution")
			assert.Equal(t, toJSONString(tc.expected), replacement, "Should preserve special characters")
		})
	}
}
