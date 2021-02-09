package expr

import (
	"fmt"
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
	key := fmt.Sprintf("inputs.parameters.%s", "num0")
	env[key] = 2
	result, err := Eval(code, env)
	assert.NoError(err)
	assert.Equal(1, result)
}
