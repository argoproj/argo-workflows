package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNextPlaceholder verifies dynamically-generated placeholder strings.
func TestNextPlaceholder(t *testing.T) {
	pg := NewPlaceholderGenerator()
	assert.Equal(t, "__argo__internal__placeholder-0", pg.NextPlaceholder())
	assert.Equal(t, "__argo__internal__placeholder-1", pg.NextPlaceholder())
	assert.Equal(t, "__argo__internal__placeholder-2", pg.NextPlaceholder())
}
