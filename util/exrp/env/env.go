package env

import (
	"github.com/Masterminds/sprig"
	"github.com/doublerebel/bellows"
)

func New(m map[string]interface{}) map[string]interface{} {
	env := bellows.Expand(m)
	env["sprig"] = sprig.GenericFuncMap()
	return env
}
