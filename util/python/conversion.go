package python

import (
	"github.com/go-python/gpython/py"

	"github.com/argoproj/argo-workflows/v3/util/json"
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
		v, _ := json.Jsonify(v)
		return obj(v)
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
