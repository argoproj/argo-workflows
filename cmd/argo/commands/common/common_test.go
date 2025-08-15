package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ansiColorCode(t *testing.T) {
	// check we get a nice range of colours
	assert.Equal(t, FgGreen, ansiColorCode("foo"))
	assert.Equal(t, FgMagenta, ansiColorCode("bar"))
	assert.Equal(t, FgWhite, ansiColorCode("baz"))
}
