package expand

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpand(t *testing.T) {
	for i := range 1 { // loop 100 times, because map ordering is not determisitic
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			before := map[string]any{
				"a.b":   1,
				"a.c.d": 2,
				"a":     3, // should be deleted
				"ab":    4,
				"abb":   5, // should be kept
			}
			after := Expand(before)
			assert.Len(t, before, 5, "original map unchanged")
			assert.Equal(t, map[string]any{
				"a": map[string]any{
					"b": 1,
					"c": map[string]any{
						"d": 2,
					},
				},
				"ab":  4,
				"abb": 5,
			}, after)
		})
	}
}

func TestExpandParentChildWithInterveningSibling(t *testing.T) {
	// "a-c" contains a hyphen, which sorts between "a" and "a.b". The parent
	// "a" still has a child ("a.b") and must be dropped, otherwise Expand tries
	// to make "a" both a scalar and a nested map and silently drops the child.
	before := map[string]any{
		"a":   3,
		"a-c": 4,
		"a.b": 1,
	}
	after := Expand(before)
	assert.Equal(t, map[string]any{
		"a":   map[string]any{"b": 1},
		"a-c": 4,
	}, after)
}
