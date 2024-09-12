package expand

import (
	"reflect"
	"sort"
	"strings"
)

func Expand(m map[string]interface{}) map[string]interface{} {
	m = removeConflicts(m)
	result := make(map[string]interface{})
	for k, v := range m {
		expandKey(result, k, v)
	}
	return result
}

func expandKey(result map[string]interface{}, key string, value interface{}) {
	current := result
	start := 0
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			part := key[start:i]
			next, ok := current[part]
			if !ok {
				next = make(map[string]interface{})
				current[part] = next
			} else if _, ok := next.(map[string]interface{}); !ok {
				next = make(map[string]interface{})
				current[part] = next
			}
			current = next.(map[string]interface{})
			start = i + 1
		}
	}
	current[key[start:]] = value
}

func Flatten(value interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	flattenRecursive(reflect.ValueOf(value), "", result)
	return result
}

func flattenRecursive(v reflect.Value, prefix string, result map[string]interface{}) {
	switch v.Kind() {
	case reflect.Interface:
		flattenRecursive(v.Elem(), prefix, result)
	case reflect.Ptr:
		flattenRecursive(v.Elem(), prefix, result)
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return
		}
		for _, key := range v.MapKeys() {
			flattenRecursive(v.MapIndex(key), joinKey(prefix, key.String()), result)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			flattenRecursive(v.Field(i), joinKey(prefix, v.Type().Field(i).Name), result)
		}
	default:
		if prefix != "" {
			result[prefix] = v.Interface()
		}
	}
}

func joinKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// It is possible for the map to contain conflicts:
// {"a.b": 1, "a": 2}
// What should the result be? We remove the less-specific key.
// {"a.b": 1, "a": 2} -> {"a.b": 1, "a": 2}
func removeConflicts(m map[string]interface{}) map[string]interface{} {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make(map[string]interface{})
	for i, k := range keys {
		if i < len(keys)-1 && strings.HasPrefix(keys[i+1], k+".") {
			continue // Skip this key as it conflicts with a more specific one
		}
		result[k] = m[k]
	}
	return result
}
