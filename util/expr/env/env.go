package env

import (
	"encoding/json"

	"github.com/Masterminds/sprig"
	exprpkg "github.com/argoproj/pkg/expr"
	"github.com/doublerebel/bellows"
)

func GetFuncMap(m map[string]interface{}) map[string]interface{} {
	env := bellows.Expand(m)
	for k, v := range exprpkg.GetExprEnvFunctionMap() {
		env[k] = v
	}
	env["toJson"] = toJson
	env["sprig"] = sprig.GenericFuncMap()
	return env
}

func toJson(v interface{}) string {
	output, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(output)
}
