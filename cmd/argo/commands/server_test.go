package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSecureMode(t *testing.T) {
	// Secure mode by default
	cmd := NewServerCommand()
	assert.Equal(t, "true", cmd.Flag("secure").Value.String())
}

func TestLogoutRedirectURLFlag(t *testing.T) {
	cmd := NewServerCommand()
	assert.Empty(t, cmd.Flag("logout-redirect-url").Value.String())
	require.NoError(t, cmd.Flags().Set("logout-redirect-url", "https://example.com/logged-out"))
	assert.Equal(t, "https://example.com/logged-out", cmd.Flag("logout-redirect-url").Value.String())
}
