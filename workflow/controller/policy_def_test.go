package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_policyDef_matches(t *testing.T) {
	assert.False(t, policyDef{}.matches("", "", "", actRead))
	assert.True(t, policyDef{act: actRead}.matches("", "", "", actRead))
	assert.True(t, policyDef{act: actRead ^ actWrite}.matches("", "", "", actRead))
}
