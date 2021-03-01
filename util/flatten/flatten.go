package flatten

import (
	"fmt"
	"reflect"

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
