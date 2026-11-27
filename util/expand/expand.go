package expand

import (
	"sort"
	"strings"
)

// Expand turns dotted keys into nested maps: {"a.b": 1} -> {"a": {"b": 1}}.
// Each key is walked once, so the work is linear in the total key length.
func Expand(m map[string]any) map[string]any {
	root := newNode()
	for k, v := range removeConflicts(m) {
		n := root
		for {
			seg, rest, nested := strings.Cut(k, ".")
			if !nested {
				// A key that is also a parent keeps its nested map.
				if _, ok := n.children[seg]; !ok {
					n.values[seg] = v
				}
				break
			}
			child, ok := n.children[seg]
			if !ok {
				child = newNode()
				n.children[seg] = child
				n.values[seg] = child.values
			}
			n, k = child, rest
		}
	}
	return root.values
}

// node is one level of the expanded map. children holds the entries of values
// that Expand created as nested maps, so they are told apart from leaf values
// that happen to be maps themselves.
type node struct {
	values   map[string]any
	children map[string]*node
}

func newNode() *node {
	return &node{values: map[string]any{}, children: map[string]*node{}}
}

// It is possible for the map to contain conflicts:
// {"a.b": 1, "a": 2}
// What should the result be? We remove the less-specific key.
// {"a.b": 1, "a": 2} -> {"a.b": 1, "a": 2}
func removeConflicts(m map[string]any) map[string]any {
	var keys []string
	n := map[string]any{}
	for k, v := range m {
		keys = append(keys, k)
		n[k] = v
	}
	sort.Strings(keys)
	for i := 0; i < len(keys)-1; i++ {
		k := keys[i]
		// remove any parent that has a child
		if strings.HasPrefix(keys[i+1], k+".") {
			delete(n, k)
		}
	}
	return n
}
