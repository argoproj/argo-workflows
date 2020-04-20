package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModes_Add(t *testing.T) {
	t.Run("InvalidMode", func(t *testing.T) {
		assert.Error(t, Modes{}.Add(""))
	})
	t.Run("Client", func(t *testing.T) {
		m := Modes{}
		if assert.NoError(t, m.Add(Client)) {
			assert.Contains(t, m, Client)
		}
	})
	t.Run("Hybrid", func(t *testing.T) {
		m := Modes{}
		if assert.NoError(t, m.Add(Hybrid)) {
			assert.Contains(t, m, Client)
			assert.Contains(t, m, Server)
		}
	})
	t.Run("Server", func(t *testing.T) {
		m := Modes{}
		if assert.NoError(t, m.Add(Server)) {
			assert.Contains(t, m, Server)
		}
	})
	t.Run("SSO", func(t *testing.T) {
		m := Modes{}
		if assert.NoError(t, m.Add(SSO)) {
			assert.Contains(t, m, SSO)
		}
	})
}
