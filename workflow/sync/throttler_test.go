package sync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoParallelismSamePriority(t *testing.T) {
	r := NewThrottler(0, func(key string) {})

	r.Add(&val{"c", 1})
	r.Add(&val{"b", 2})
	r.Add(&val{"a", 3})

	assert.True(t, r.Next("a"))
	assert.True(t, r.Next("b"))
	assert.True(t, r.Next("c"))
}

func TestWithParallelismLimitAndPriority(t *testing.T) {
	queue := ""
	r := NewThrottler(2, func(key string) { queue = key })

	r.Add(&val{"a", 1})
	r.Add(&val{"b", 2})
	r.Add(&val{"c", 3})
	r.Add(&val{"d", 4})

	assert.False(t, r.Next("a"))
	assert.False(t, r.Next("b"))
	assert.False(t, r.Next("c"))
	assert.True(t, r.Next("d"))
	assert.True(t, r.Next("c"))

	assert.Equal(t, "", queue)

	r.Remove("c")
}
