package jws

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClaimSet_Sub(t *testing.T) {
	assert.Empty(t, ClaimSet{}.Sub())
	assert.Empty(t, ClaimSet{"sub": false}.Sub())
	assert.NotEmpty(t, ClaimSet{"sub": "ok"}.Sub())
}

func TestClaimSet_Iss(t *testing.T) {
	assert.Empty(t, ClaimSet{}.Iss())
	assert.Empty(t, ClaimSet{"iss": false}.Iss())
	assert.NotEmpty(t, ClaimSet{"iss": "ok"}.Iss())
}

func TestClaimSet_Groups(t *testing.T) {
	assert.Empty(t, ClaimSet{}.Groups())
	assert.Empty(t, ClaimSet{"groups": false}.Groups())
	assert.NotEmpty(t, ClaimSet{"groups": []string{"ok"}}.Groups())
}
