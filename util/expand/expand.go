package expand

import (
	"sort"
	"strings"

	"github.com/doublerebel/bellows"
)

func Expand(m map[string]any) map[string]any {
	return bellows.Expand(removeConflicts(m))
}

// It is possible for the map to contain conflicts:
// {"a.b": 1, "a": 2}
// What should the result be? We remove the less-specific key, so that the
// value nested under "a" is not clobbered: {"a.b": 1, "a": 2} -> {"a.b": 1}.
func removeConflicts(m map[string]any) map[string]any {
	var keys []string
	n := map[string]any{}
	for k, v := range m {
		keys = append(keys, k)
		n[k] = v
	}
	sort.Strings(keys)
	for i := 0; i < len(keys); i++ {
		k := keys[i]
		// Remove any parent that has a child. A child shares the prefix k+".",
		// but is not necessarily the immediate successor of k once sorted: a
		// sibling such as "a-c" sorts between "a" and "a.b". Scan forward over
		// every key that still has k as a prefix until a child is found.
		for j := i + 1; j < len(keys) && strings.HasPrefix(keys[j], k); j++ {
			if strings.HasPrefix(keys[j], k+".") {
				delete(n, k)
				break
			}
		}
	}
	return n
}
