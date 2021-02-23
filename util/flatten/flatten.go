package flatten

import (
	"fmt"
	"reflect"
	"strings"

	jsonutil "github.com/argoproj/argo-workflows/v3/util/json"
)

func toMap(in interface{}) map[string]interface{} {
	v, _ := jsonutil.Jsonify(in)
	return v
}

func flattenWithPrefix(in map[string]interface{}, out map[string]string, prefix string) {
	for k, v := range in {
		if v == nil {
			continue
		}
		switch reflect.TypeOf(v).Kind() {
		case reflect.Map:
			flattenWithPrefix(toMap(v), out, prefix+k+".")
		default:
			out[prefix+k] = fmt.Sprintf("%v", v)
		}
	}
}

// Flatten converts a struct into a map[string]string using dot-notation.
// E.g. listOptions.continue="10"
func Flatten(in interface{}) map[string]string {
	out := make(map[string]string)
	flattenWithPrefix(toMap(in), out, "")
	return out
}

// Expand converts a expand map key to nested keys
// E.g. map["listOptions.continue": 10] to map["listOptions": map["continue":10]]
func Expand(value map[string]interface{}) map[string]interface{} {
	return ExpandPrefixed(value, "")
}

func ExpandPrefixed(value map[string]interface{}, prefix string) map[string]interface{} {
	m := make(map[string]interface{})
	ExpandPrefixedToResult(value, prefix, m)
	return m
}

func ExpandPrefixedToResult(value map[string]interface{}, prefix string, result map[string]interface{}) {
	if prefix != "" {
		prefix += "."
	}
	for k, val := range value {
		if !strings.HasPrefix(k, prefix) {
			continue
		}

		key := k[len(prefix):]
		idx := strings.Index(key, ".")
		if idx != -1 {
			key = key[:idx]
		}
		if _, ok := result[key]; ok {
			continue
		}
		if idx == -1 {
			result[key] = val
			continue
		}

		// It contains a period, so it is a more complex structure
		result[key] = ExpandPrefixed(value, k[:len(prefix)+len(key)])
	}
}
