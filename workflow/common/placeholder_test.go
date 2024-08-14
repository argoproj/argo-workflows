package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNextPlaceholder verifies dynamically-generated placeholder strings.
func TestNextPlaceholder(t *testing.T) {
	pg := NewPlaceholderGenerator()
	require.Equal(t, "placeholder-0", pg.NextPlaceholder())
	require.Equal(t, "placeholder-1", pg.NextPlaceholder())
	require.Equal(t, "placeholder-2", pg.NextPlaceholder())
}
