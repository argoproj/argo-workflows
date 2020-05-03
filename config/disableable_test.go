package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
)

func TestDisableable_IsEnabled(t *testing.T) {
	var d *Disableable
	assert.True(t, d.IsEnabled())
	assert.True(t, (&Disableable{}).IsEnabled())
	assert.True(t, (&Disableable{Enabled: pointer.BoolPtr(true)}).IsEnabled())
	assert.False(t, (&Disableable{Enabled: pointer.BoolPtr(false)}).IsEnabled())
}