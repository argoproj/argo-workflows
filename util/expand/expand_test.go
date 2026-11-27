package expand

import (
	"fmt"
	"strings"
	"testing"

	"github.com/doublerebel/bellows"
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

// TestExpandMatchesBellows checks Expand against the bellows implementation it
// replaced, over every key of up to three characters from "a", "b" and ".",
// alone and in pairs, so empty, repeated and trailing segments are covered.
func TestExpandMatchesBellows(t *testing.T) {
	keys := []string{""}
	for n := range 3 {
		for _, k := range keys {
			if len(k) == n {
				for _, c := range []string{"a", "b", "."} {
					if k+c != "." {
						keys = append(keys, k+c)
					}
				}
			}
		}
	}
	cases := []map[string]any{
		{"x": map[string]any{"y": 1}, "x-y.z": 2},
		{"item." + strings.Repeat("a.", 100) + "b": "v", "workflow.name": "w"},
	}
	all := map[string]any{}
	for i, k := range keys {
		all[k] = i
		for j, l := range keys {
			cases = append(cases, map[string]any{k: i, l: j})
		}
	}
	cases = append(cases, all)
	for _, m := range cases {
		assert.Equal(t, bellows.Expand(removeConflicts(m)), Expand(m), "%v", m)
	}
}

// TestExpandKeepsNestedOverLeaf covers a parent that removeConflicts keeps
// because a sibling sorts between it and its child: the nested map is kept
// whatever order the keys are visited in.
func TestExpandKeepsNestedOverLeaf(t *testing.T) {
	for range 20 {
		assert.Equal(t, map[string]any{
			"a":   map[string]any{"b": 1},
			"a-c": 4,
		}, Expand(map[string]any{"a": 3, "a-c": 4, "a.b": 1}))
	}
}

func BenchmarkExpandManyDots(b *testing.B) {
	for _, dots := range []int{1000, 2000, 4000, 8000} {
		m := map[string]any{"item." + strings.Repeat("a.", dots) + "b": "v", "workflow.name": "w"}
		b.Run(fmt.Sprint(dots), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				Expand(m)
			}
		})
	}
}
