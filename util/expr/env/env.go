package env

import (
	"encoding/json"

	sprig "github.com/Masterminds/sprig/v3"
	"github.com/evilmonkeyinc/jsonpath"
	"github.com/expr-lang/expr/builtin"

	"github.com/argoproj/argo-workflows/v4/util/expand"
)

var sprigFuncMap = sprig.GenericFuncMap() // a singleton for better performance

func init() {
	delete(sprigFuncMap, "env")
	delete(sprigFuncMap, "expandenv")
}

func GetFuncMap(m map[string]any) map[string]any {
	env := expand.Expand(m)
	// Alias for the built-in `int` function, for backwards compatibility.
	env["asInt"] = builtin.Int
	// Alias for the built-in `float` function, for backwards compatibility.
	env["asFloat"] = builtin.Float
	env["jsonpath"] = jsonPath
	env["toJson"] = toJSON
	env["sprig"] = sprigFuncMap
	return env
}

func toJSON(v any) string {
	output, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(output)
}

func jsonPath(jsonStr string, path string) any {
	var jsonMap any
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
