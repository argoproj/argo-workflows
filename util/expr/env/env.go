package env

import (
	"encoding/json"

	sprig "github.com/Masterminds/sprig/v3"
	exprpkg "github.com/argoproj/pkg/expr"

	"github.com/argoproj/argo-workflows/v3/util/expand"
)

var sprigFuncMap = sprig.GenericFuncMap() // a singleton for better performance

func init() {
	delete(sprigFuncMap, "env")
	delete(sprigFuncMap, "expandenv")
}

func GetFuncMap(m map[string]interface{}) map[string]interface{} {
	env := expand.Expand(m)
	for k, v := range exprpkg.GetExprEnvFunctionMap() {
		env[k] = v
	}
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
