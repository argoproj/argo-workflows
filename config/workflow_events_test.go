package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowEvents_IsEnabled(t *testing.T) {
	assert.True(t, WorkflowEvents{}.IsEnabled())
	assert.False(t, WorkflowEvents{Enabled: new(false)}.IsEnabled())
	assert.True(t, WorkflowEvents{Enabled: new(true)}.IsEnabled())
}
