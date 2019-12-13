package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPlaceholder(t *testing.T) {
	pg := NewPlaceholderGenerator()
	assert.Equal(t, pg.GetPlaceholder(), "placeholder-0")
	assert.Equal(t, pg.GetPlaceholder(), "placeholder-1")
	assert.Equal(t, pg.GetPlaceholder(), "placeholder-2")
}
