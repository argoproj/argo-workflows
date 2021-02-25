package expr

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/oliveagle/jsonpath"
)

func addExprFunctions(env map[string]interface{}) {
	env["number"] = Number
	env["string"] = String
	env["jsonpath"] = JsonPath
}

func Number(val string) interface{} {
	num, err := strconv.ParseFloat(val, 64)
	if err != nil {
		panic(fmt.Sprintf("%q can't convert string to number.", val))
	}
	return num
}

func String(val interface{}) interface{} {
	return fmt.Sprintf("%v", val)
}

func JsonPath(jsonStr string, path string) interface{} {
	var jsonMap interface{}
	err := json.Unmarshal([]byte(jsonStr), &jsonMap)
	if err != nil {
		panic(err)
	}
	value, err := jsonpath.JsonPathLookup(jsonMap, path)
	if err != nil {
		panic(err)
	}
	return value
}
