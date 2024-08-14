package template

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_parseTag(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		kind, tag := parseTag("tag")
		require.Equal(t, kindSimple, kind)
		require.Equal(t, "tag", tag)
	})
	t.Run("Expression", func(t *testing.T) {
		kind, tag := parseTag("=tag")
		require.Equal(t, kindExpression, kind)
		require.Equal(t, "tag", tag)
	})
}
