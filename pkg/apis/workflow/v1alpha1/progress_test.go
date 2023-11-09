package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProgress(t *testing.T) {
	t.Run("ParseProgress", func(t *testing.T) {
		_, ok := ParseProgress("")
		assert.False(t, ok)
		progress, ok := ParseProgress("0/1")
		assert.True(t, ok)
		assert.Equal(t, Progress("0/1"), progress)
	})
	t.Run("IsValid", func(t *testing.T) {
		assert.False(t, Progress("").IsValid())
		assert.False(t, Progress("/0").IsValid())
		assert.False(t, Progress("0/").IsValid())
		assert.False(t, Progress("0/0").IsValid())
		assert.False(t, Progress("1/0").IsValid())
		assert.True(t, Progress("0/1").IsValid())
	})
	t.Run("Add", func(t *testing.T) {
		assert.Equal(t, Progress("1/2"), Progress("0/0").Add("1/2"))
	})
	t.Run("Complete", func(t *testing.T) {
		assert.Equal(t, Progress("100/100"), Progress("0/100").Complete())
	})
}
