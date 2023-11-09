package json

import "encoding/json"

func Jsonify(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	x := make(map[string]interface{})
	return x, json.Unmarshal(data, &x)
}
