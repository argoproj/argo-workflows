package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ansiColorCode(t *testing.T) {
	// check we get a nice range of colours
	assert.Equal(t, FgYellow, ANSIColorCode("foo"))
	assert.Equal(t, FgGreen, ANSIColorCode("bar"))
	assert.Equal(t, FgYellow, ANSIColorCode("baz"))
	assert.Equal(t, FgRed, ANSIColorCode("qux"))
}
