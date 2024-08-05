package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestNodeEvents_IsEnabled(t *testing.T) {
	assert.True(t, NodeEvents{}.IsEnabled())
	assert.False(t, NodeEvents{Enabled: ptr.To(false)}.IsEnabled())
	assert.True(t, NodeEvents{Enabled: ptr.To(true)}.IsEnabled())
}
