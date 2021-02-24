package json

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Fix(t *testing.T) {
	assert.Equal(t, "<", Fix("\\u003c"))
	assert.Equal(t, ">", Fix("\\u003e"))
	assert.Equal(t, "&", Fix("\\u0026"))
}
