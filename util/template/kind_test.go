package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseTag(t *testing.T) {
	t.Run("Simple", func(t *testing.T) {
		kind, tag := parseTag("tag")
		assert.Equal(t, kindSimple, kind)
		assert.Equal(t, "tag", tag)
	})
	t.Run("Expression", func(t *testing.T) {
		kind, tag := parseTag("=tag")
		assert.Equal(t, kindExpression, kind)
		assert.Equal(t, "tag", tag)
	})
}
