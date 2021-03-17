package python

import (
	"fmt"

	_ "github.com/go-python/gpython/builtin"
	"github.com/go-python/gpython/compile"
	"github.com/go-python/gpython/py"
	"github.com/go-python/gpython/vm"
)

func Run(s string, globals map[string]interface{}) error {
	x, err := compile.Compile(s, "<stdin>", "exec", 0, true)
	if err != nil {
		return err
	}
	m := py.NewModule("__main__", "", nil, dict(globals))
	code, ok := x.(*py.Code)
	if !ok {
		return fmt.Errorf("obj cannot be cast to code")
	}
	_, err = vm.EvalCode(code, m.Globals, nil)
	return err
}
