package json

import "encoding/json"

func Jsonify(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	x := make(map[string]any)
	return x, json.Unmarshal(data, &x)
}
