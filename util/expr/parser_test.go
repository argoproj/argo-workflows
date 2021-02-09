package expr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVariable(t *testing.T) {
	assert := assert.New(t)
	expr := "(steps.flipcoin.outputs.result == 'head')?steps.heads.outputs.result:steps.tails.outputs.result"
	variables := GetVariable(expr)
	assert.Len(variables, 3)
	assert.Contains(variables, "steps.flipcoin.outputs.result")
	assert.Contains(variables, "steps.heads.outputs.result")
	assert.Contains(variables, "steps.tails.outputs.result")
	expr = "(steps.flipcoin.outputs.result == 2)?steps.heads.outputs.result:steps.tails.outputs.result"
	variables = GetVariable(expr)
	assert.Len(variables, 3)
	assert.Contains(variables, "steps.flipcoin.outputs.result")
	assert.Contains(variables, "steps.heads.outputs.result")
	assert.Contains(variables, "steps.tails.outputs.result")

	expr = "(steps.flipcoin.outputs.result == 2)?4:steps.tails.outputs.result"
	variables = GetVariable(expr)
	assert.Len(variables, 2)
	assert.Contains(variables, "steps.flipcoin.outputs.result")
	assert.Contains(variables, "steps.tails.outputs.result")

	expr = `steps.heads.outputs.result+steps.tails.outputs.result`
	variables = GetVariable(expr)
	assert.Len(variables, 2)
	assert.Contains(variables, "steps.heads.outputs.result")
	assert.Contains(variables, "steps.tails.outputs.result")
	expr = `(steps.heads.outputs.result+steps.tails.outputs.result)==steps.flipcoin.outputs.result`
	variables = GetVariable(expr)
	assert.Len(variables, 3)
	assert.Contains(variables, "steps.heads.outputs.result")
	assert.Contains(variables, "steps.tails.outputs.result")
	assert.Contains(variables, "steps.flipcoin.outputs.result")
}
