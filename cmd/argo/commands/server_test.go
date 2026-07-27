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

func TestLogoutRedirectURLFlag(t *testing.T) {
	cmd := NewServerCommand()
	assert.Equal(t, "", cmd.Flag("logout-redirect-url").Value.String())
	assert.NoError(t, cmd.Flags().Set("logout-redirect-url", "https://example.com/logged-out"))
	assert.Equal(t, "https://example.com/logged-out", cmd.Flag("logout-redirect-url").Value.String())
}
