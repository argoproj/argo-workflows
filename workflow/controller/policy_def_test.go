package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_policyDef_matches(t *testing.T) {
	assert.False(t, policyDef{}.matches("", "", "", roleRead))
	assert.True(t, policyDef{role: roleRead}.matches("", "", "", roleRead))
	assert.True(t, policyDef{role: roleRead ^ roleWrite}.matches("", "", "", roleRead))
}
