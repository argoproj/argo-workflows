package commands

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultSecureMode(t *testing.T) {
	// Secure mode by default
	cmd := NewServerCommand()
	require.Equal(t, "true", cmd.Flag("secure").Value.String())
}
