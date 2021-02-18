package e2e

import (
	"reflect"

	"github.com/go-python/gpython/py"
)

func obj(v interface{}) py.Object {
	switch x := v.(type) {
	case float64:
		return py.Float(x)
	case int:
		return py.Int(x)
	case string:
		return py.String(x)
	case []interface{}:
		return list(x)
	case map[string]interface{}:
		return dict(x)
	default:
		panic(reflect.TypeOf(x).String())
	}
}

func list(v []interface{}) *py.List {
	out := py.NewList()
	for _, x := range v {
		out.Append(obj(x))
	}
	return out
}

func dict(v map[string]interface{}) py.StringDict {
	out := py.NewStringDict()
	for k, x := range v {
		out[k] = obj(x)
	}
	return out
}
