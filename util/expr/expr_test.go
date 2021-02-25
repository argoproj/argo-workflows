package expr

import (
	"testing"

	assert "github.com/stretchr/testify/assert"
)

func TestEval(t *testing.T) {
	assert := assert.New(t)
	code := `inputs.parameters.num0 == 1? 1:2`
	env := map[string]interface{}{
		"steps.fibonaccihelper.id":         "fibonacci-xb6xr-397095958",
		"steps.fibonaccihelper.startedAt":  "2021-02-04T23:38:32Z",
		"steps.fibonaccihelper.finishedAt": "2021-02-04T23:38:32Z",
		"steps.fibonaccihelper.status":     "Skipped",
		"outputs.parameters.fib":           nil,
	}
	env["inputs.parameters.num0"] = "2"
	result, err := Eval(code, env)
	assert.NoError(err)
	assert.Equal(2, result)
}

func TestExprFunctions(t *testing.T) {
	assert := assert.New(t)
	env := map[string]interface{}{
		"a.n":    1,
		"a.s":    "1",
		"a.json": "{ \"empId\": \"1\", \"empName\" : \"test\"}",
	}
	t.Run("ExprNumberFunction", func(t *testing.T) {
		exp := "number(a.s) == 1? true: false"
		result, err := Eval(exp, env)
		assert.NoError(err)
		assert.Equal(true, result)
	})
	t.Run("ExprStringFunction", func(t *testing.T) {
		exp := "string(a.n) == '1'? true: false"
		result, err := Eval(exp, env)
		assert.NoError(err)
		assert.Equal(true, result)
	})
	t.Run("ExprJsonFunction", func(t *testing.T) {
		exp := "jsonpath(a.json, '$.empId') == '1'? true: false"
		result, err := Eval(exp, env)
		assert.NoError(err)
		assert.Equal(true, result)
	})
}
