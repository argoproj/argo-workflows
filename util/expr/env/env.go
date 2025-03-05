package env

import (
	"encoding/json"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/evilmonkeyinc/jsonpath"
	"github.com/expr-lang/expr/builtin"

	"github.com/argoproj/argo-workflows/v3/util/expand"
)

var sprigFuncMap = sprig.GenericFuncMap() // a singleton for better performance

func init() {
	delete(sprigFuncMap, "env")
	delete(sprigFuncMap, "expandenv")
}

func GetFuncMap(m map[string]interface{}) map[string]interface{} {
	env := expand.Expand(m)
	// Alias for the built-in `int` function, for backwards compatibility.
	env["asInt"] = builtin.Int
	// Alias for the built-in `float` function, for backwards compatibility.
	env["asFloat"] = builtin.Float
	env["jsonpath"] = jsonPath
	env["toJson"] = toJson
	env["sprig"] = sprigFuncMap
	return env
}

func toJson(v interface{}) string {
	output, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(output)
}

func jsonPath(jsonStr string, path string) interface{} {
	var jsonMap interface{}
	err := json.Unmarshal([]byte(jsonStr), &jsonMap)
	if err != nil {
		panic(err)
	}
	value, err := jsonpath.Query(path, jsonMap)
	if err != nil {
		panic(err)
	}
	return value
}
