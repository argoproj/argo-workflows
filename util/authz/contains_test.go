package authz

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_containsFunc(t *testing.T) {
	assert.True(t, contains(strings.Split("*", ","), "foo"))
	assert.True(t, contains(strings.Split("foo", ","), "foo"))
}
