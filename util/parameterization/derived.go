package parameterization

import (
	"fmt"
	"reflect"
	"strings"
)

func markSources(v any, path string, sources map[string]string) {
	switch value := v.(type) {
	case map[string]any:
		for key, child := range value {
			markSources(child, path+"/"+Escape(key), sources)
		}
	case []any:
		for i, child := range value {
			markSources(child, fmt.Sprintf("%s/%d", path, i), sources)
		}
	default:
		sources[path] = "user"
	}
}

// Derived targets are read-only and must not have a conflicting submitted value.
// The original derived value can be submitted unchanged when loading a revision.
func applyDerived(rules []Derivation, values any, sources map[string]string) error {
	targets := map[string]Derivation{}
	visiting := map[string]bool{}
	done := map[string]bool{}
	for _, rule := range rules {
		if rule.Path == "" || !strings.HasPrefix(rule.Path, "/") || rule.Path == rule.From {
			return fmt.Errorf("invalid derived target %s", rule.Path)
		}
		if _, ok := targets[rule.Path]; ok {
			return fmt.Errorf("duplicate derived target %s", rule.Path)
		}
		targets[rule.Path] = rule
	}
	var resolve func(string) error
	resolve = func(path string) error {
		if done[path] {
			return nil
		}
		if visiting[path] {
			return fmt.Errorf("derived dependency cycle at %s", path)
		}
		visiting[path] = true
		rule := targets[path]
		for dep := range targets {
			if rule.From == dep || strings.HasPrefix(rule.From, dep+"/") {
				if err := resolve(dep); err != nil {
					return err
				}
			}
		}
		value, ok := Pointer(values, rule.From)
		if !ok {
			return fmt.Errorf("derived source %s is absent", rule.From)
		}
		old, exists := Pointer(values, path)
		if exists && !reflect.DeepEqual(old, value) {
			return fmt.Errorf("derived target %s conflicts with its source", path)
		}
		split := strings.LastIndex(path, "/")
		container, found := Pointer(values, path[:split])
		if !found {
			return fmt.Errorf("derived parent is absent at %s", path)
		}
		m, ok := container.(map[string]any)
		if !ok {
			return fmt.Errorf("derived target must be an object property")
		}
		key := strings.ReplaceAll(strings.ReplaceAll(path[split+1:], "~1", "/"), "~0", "~")
		m[key] = Clone(value)
		sources[path] = "derived:" + rule.From
		visiting[path] = false
		done[path] = true
		return nil
	}
	for path := range targets {
		if err := resolve(path); err != nil {
			return err
		}
	}
	return nil
}
