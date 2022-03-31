package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseMethod(t *testing.T) {
	verb, resource := parseMethod("ListCronWorkflows")
	assert.Equal(t, "list", verb)
	assert.Equal(t, "cronworkflows", resource)
}
