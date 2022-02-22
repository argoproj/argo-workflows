package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSecureMode(t *testing.T) {
	// Secure mode by default
	cmd := NewServerCommand()
	assert.Equal(t, "true", cmd.Flag("secure").Value.String())
}
