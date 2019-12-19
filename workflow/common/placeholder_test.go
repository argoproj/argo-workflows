package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNextPlaceholder verifies dynamically-generated placeholder strings.
func TestNextPlaceholder(t *testing.T) {
	pg := NewPlaceholderGenerator()
	assert.Equal(t, pg.NextPlaceholder(), "placeholder-0")
	assert.Equal(t, pg.NextPlaceholder(), "placeholder-1")
	assert.Equal(t, pg.NextPlaceholder(), "placeholder-2")
}
