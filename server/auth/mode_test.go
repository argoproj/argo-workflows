package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModes_Add(t *testing.T) {
	t.Run("InvalidMode", func(t *testing.T) {
		require.Error(t, Modes{}.Add(""))
	})
	t.Run("Client", func(t *testing.T) {
		m := Modes{}
		require.NoError(t, m.Add("client"))
		assert.Contains(t, m, Client)
	})
	t.Run("Hybrid", func(t *testing.T) {
		m := Modes{}
		require.NoError(t, m.Add("hybrid"))
		assert.Contains(t, m, Client)
		assert.Contains(t, m, Server)
	})
	t.Run("Server", func(t *testing.T) {
		m := Modes{}
		require.NoError(t, m.Add("server"))
		assert.Contains(t, m, Server)
	})
	t.Run("SSO", func(t *testing.T) {
		m := Modes{}
		require.NoError(t, m.Add("sso"))
		assert.Contains(t, m, SSO)
	})
}
