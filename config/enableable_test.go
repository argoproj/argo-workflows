package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnableable_IsEnabled(t *testing.T) {
	var d *Enableable
	assert.False(t, d.IsEnabled())
	assert.False(t, (&Enableable{}).IsEnabled())
	assert.False(t, (&Enableable{Enabled: false}).IsEnabled())
	assert.True(t, (&Enableable{Enabled: true}).IsEnabled())
}
