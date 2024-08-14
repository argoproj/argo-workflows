package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModes_Add(t *testing.T) {
	t.Run("InvalidMode", func(t *testing.T) {
		require.Error(t, Modes{}.Add(""))
	})
	t.Run("Client", func(t *testing.T) {
		m := Modes{}
		require.NoError(t, m.Add("client"))
		require.Contains(t, m, Client)
	})
	t.Run("Hybrid", func(t *testing.T) {
		m := Modes{}
		require.NoError(t, m.Add("hybrid"))
		require.Contains(t, m, Client)
		require.Contains(t, m, Server)
	})
	t.Run("Server", func(t *testing.T) {
		m := Modes{}
		require.NoError(t, m.Add("server"))
		require.Contains(t, m, Server)
	})
	t.Run("SSO", func(t *testing.T) {
		m := Modes{}
		require.NoError(t, m.Add("sso"))
		require.Contains(t, m, SSO)
	})
}

func TestModes_GetMode(t *testing.T) {
	m := Modes{
		Client: true,
		SSO:    true,
		Server: true,
	}
	t.Run("Client", func(t *testing.T) {
		mode, valid := m.GetMode("Bearer ")
		if require.True(t, valid) {
			require.Equal(t, Client, mode)
		}
	})
	t.Run("Server", func(t *testing.T) {
		mode, valid := m.GetMode("")
		if require.True(t, valid) {
			require.Equal(t, Server, mode)
		}
	})
	t.Run("SSO", func(t *testing.T) {
		mode, valid := m.GetMode("Bearer v2:")
		if require.True(t, valid) {
			require.Equal(t, SSO, mode)
		}
	})

	m = Modes{
		Client: false,
		SSO:    false,
		Server: true,
	}
	t.Run("Server and Auth", func(t *testing.T) {
		mode, valid := m.GetMode("Bearer ")
		if require.True(t, valid) {
			require.Equal(t, Server, mode)
		}
	})
}
