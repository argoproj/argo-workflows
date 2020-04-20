package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModes_Add(t *testing.T) {
	assert.Error(t, Modes{}.Add(""))
	assert.Equal(t, Modes{Client: true}, Modes{}.Add(Client))
	assert.Equal(t, Modes{Client: true, Server: true}, Modes{}.Add(Hybrid))
	assert.Equal(t, Modes{Server: true}, Modes{}.Add(Server))
	assert.Equal(t, Modes{SSO: true}, Modes{}.Add(SSO))
}
