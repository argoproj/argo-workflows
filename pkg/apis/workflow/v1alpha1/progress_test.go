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

func TestProgressV2(t *testing.T) {
	p, ok := ParseProgress("SFKRP")
	assert.True(t, ok)
	assert.Equal(t, 4, p.N())
	assert.Equal(t, 5, p.M())
	assert.Equal(t, NodeSucceeded, p.Status(0))
	assert.Equal(t, NodeFailed, p.Status(1))
	assert.Equal(t, NodeSkipped, p.Status(2))
	assert.Equal(t, NodeRunning, p.Status(3))
	assert.Equal(t, NodePending, p.Status(4))
	assert.Equal(t, NodePending, p.Status(5))
	assert.True(t, p.Failure())
	p = p.WithStatus(1, NodeSucceeded)
	assert.False(t, p.Failure())
}
