package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ansiColorCode(t *testing.T) {
	// check we get a nice range of colours
	assert.Equal(t, FgYellow, ansiColorCode("foo"))
	assert.Equal(t, FgGreen, ansiColorCode("bar"))
	assert.Equal(t, FgYellow, ansiColorCode("baz"))
	assert.Equal(t, FgRed, ansiColorCode("qux"))
}
