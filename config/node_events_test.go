package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNodeEvents_IsEnabled(t *testing.T) {
	assert.True(t, NodeEvents{}.IsEnabled())
	assert.False(t, NodeEvents{Enabled: new(false)}.IsEnabled())
	assert.True(t, NodeEvents{Enabled: new(true)}.IsEnabled())
}
