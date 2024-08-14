package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ansiColorCode(t *testing.T) {
	// check we get a nice range of colours
	require.Equal(t, FgYellow, ansiColorCode("foo"))
	require.Equal(t, FgGreen, ansiColorCode("bar"))
	require.Equal(t, FgYellow, ansiColorCode("baz"))
	require.Equal(t, FgRed, ansiColorCode("qux"))
}
