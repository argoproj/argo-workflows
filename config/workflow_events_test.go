package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestWorkflowEvents_IsEnabled(t *testing.T) {
	assert.True(t, WorkflowEvents{}.IsEnabled())
	assert.False(t, WorkflowEvents{Enabled: ptr.To(false)}.IsEnabled())
	assert.True(t, WorkflowEvents{Enabled: ptr.To(true)}.IsEnabled())
}
