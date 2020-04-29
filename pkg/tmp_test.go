package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_foo(t *testing.T) {
	assert.Equal(t, "bar", foo())
}
