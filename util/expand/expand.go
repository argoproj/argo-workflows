package expand

import (
	"sort"
	"strings"

	"github.com/doublerebel/bellows"
)

func Expand(m map[string]interface{}) map[string]interface{} {
	return bellows.Expand(removeConflicts(m))
}

// It is possible for the map to contain conflicts:
// {"a.b": 1, "a": 2}
// What should the result be? We remove the less-specific key.
// {"a.b": 1, "a": 2} -> {"a.b": 1, "a": 2}
func removeConflicts(m map[string]interface{}) map[string]interface{} {
	var keys []string
	n := map[string]interface{}{}
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
