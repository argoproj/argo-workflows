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
func TestGetVariable(t *testing.T) {
	assert := assert.New(t)
	expr := "(steps.flipcoin.outputs.result == 'head')?steps.heads.outputs.result:steps.tails.outputs.result"
	variables := GetExprIdentifers(expr)
	assert.Len(variables, 3)
	assert.Contains(variables, "steps.flipcoin.outputs.result")
	assert.Contains(variables, "steps.heads.outputs.result")
	assert.Contains(variables, "steps.tails.outputs.result")
	expr = "(steps.flipcoin.outputs.result == 2)?steps.heads.outputs.result:steps.tails.outputs.result"
	variables = GetExprIdentifers(expr)
	assert.Len(variables, 3)
	assert.Contains(variables, "steps.flipcoin.outputs.result")
	assert.Contains(variables, "steps.heads.outputs.result")
	assert.Contains(variables, "steps.tails.outputs.result")

	expr = "(steps.flipcoin.outputs.result == 2)?4:steps.tails.outputs.result"
	variables = GetExprIdentifers(expr)
	assert.Len(variables, 2)
	assert.Contains(variables, "steps.flipcoin.outputs.result")
	assert.Contains(variables, "steps.tails.outputs.result")

	expr = `steps.heads.outputs.result+steps.tails.outputs.result`
	variables = GetExprIdentifers(expr)
	assert.Len(variables, 2)
	assert.Contains(variables, "steps.heads.outputs.result")
	assert.Contains(variables, "steps.tails.outputs.result")
	expr = `(steps.heads.outputs.result+steps.tails.outputs.result)==steps.flipcoin.outputs.result`
	variables = GetExprIdentifers(expr)
	assert.Len(variables, 3)
	assert.Contains(variables, "steps.heads.outputs.result")
	assert.Contains(variables, "steps.tails.outputs.result")
	assert.Contains(variables, "steps.flipcoin.outputs.result")
}
