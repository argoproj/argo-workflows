package util

import "encoding/json"

func MustUnmarshallJSON(text string, v interface{}) {
	err := json.Unmarshal([]byte(text), v)
	if err != nil {
		panic(err)
	}
}
