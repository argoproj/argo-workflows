package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
)

func TestNodeEvents_IsEnabled(t *testing.T) {
	assert.True(t, NodeEvents{}.IsEnabled())
	assert.False(t, NodeEvents{Enabled: pointer.Bool(false)}.IsEnabled())
	assert.True(t, NodeEvents{Enabled: pointer.Bool(true)}.IsEnabled())
}
