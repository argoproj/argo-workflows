package parameterization

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Diff compares semantic values, preserving absent/null and explicit collection identities.
func Diff(before, after any, rules []IdentityRule) []Change {
	identities := map[string]string{}
	for _, r := range rules {
		identities[r.Path] = r.KeyPath
	}
	patterns := make([]string, 0, len(identities))
	for pattern := range identities {
		patterns = append(patterns, pattern)
	}
	sort.Slice(patterns, func(i, j int) bool {
		a, b := strings.Count(patterns[i], "/*"), strings.Count(patterns[j], "/*")
		if a == b {
			return patterns[i] < patterns[j]
		}
		return a < b
	})
	result := []Change{}
	var walk func(any, any, string)
	walk = func(a, b any, path string) {
		if reflect.DeepEqual(a, b) {
			return
		}
		am, ao := a.(map[string]any)
		bm, bo := b.(map[string]any)
		if ao && bo {
			keys := map[string]bool{}
			for k := range am {
				keys[k] = true
			}
			for k := range bm {
				keys[k] = true
			}
			sorted := []string{}
			for k := range keys {
				sorted = append(sorted, k)
			}
			sort.Strings(sorted)
			for _, k := range sorted {
				av, ae := am[k]
				bv, be := bm[k]
				p := path + "/" + Escape(k)
				switch {
				case !ae:
					result = append(result, Change{Op: "add", Path: p, After: bv})
				case !be:
					result = append(result, Change{Op: "remove", Path: p, Before: av})
				default:
					walk(av, bv, p)
				}
			}
			return
		}
		aa, ao := a.([]any)
		ba, bo := b.([]any)
		identity := identities[path]
		if identity == "" {
			for _, pattern := range patterns {
				if MatchPath(pattern, path) {
					identity = identities[pattern]
					break
				}
			}
		}
		if ao && bo && identity != "" && validIdentities(aa, identity) && validIdentities(ba, identity) {
			key := identity
			index := map[string]int{}
			for i, item := range aa {
				v, _ := Pointer(item, key)
				index[fmt.Sprint(v)] = i
			}
			matched := map[int]bool{}
			for i, item := range ba {
				v, _ := Pointer(item, key)
				id := fmt.Sprint(v)
				p := fmt.Sprintf("%s/%d", path, i)
				j, ok := index[id]
				if !ok {
					result = append(result, Change{Op: "add", Path: p, After: item, Identity: id})
					continue
				}
				matched[j] = true
				if i != j {
					result = append(result, Change{Op: "move", Path: p, From: fmt.Sprintf("%s/%d", path, j), Identity: id})
				}
				walk(aa[j], item, p)
			}
			for i, item := range aa {
				if !matched[i] {
					v, _ := Pointer(item, key)
					result = append(result, Change{Op: "remove", Path: fmt.Sprintf("%s/%d", path, i), Before: item, Identity: fmt.Sprint(v)})
				}
			}
			return
		}
		op := "replace"
		if reflect.TypeOf(a) != reflect.TypeOf(b) {
			op = "type"
		}
		result = append(result, Change{Op: op, Path: path, Before: a, After: b})
	}
	walk(before, after, "")
	return result
}

// Invalid historical identities fall back to whole-array diff; never collapse entries.
func validIdentities(items []any, path string) bool {
	seen := map[string]bool{}
	for _, item := range items {
		v, ok := Pointer(item, path)
		id, typed := v.(string)
		if !ok || !typed || id == "" || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
