package template

import (
	"testing"

	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"
	"github.com/stretchr/testify/assert"
)

func Test_hasVariableInExpression(t *testing.T) {
	assert.True(t, hasVariableInExpression("retries", "retries"))
	assert.True(t, hasVariableInExpression("retries + 1", "retries"))
	assert.True(t, hasVariableInExpression("sprig(retries)", "retries"))
	assert.True(t, hasVariableInExpression("sprig(retries + 1) * 64", "retries"))
	assert.False(t, hasVariableInExpression("foo", "retries"))
	assert.False(t, hasVariableInExpression("retriesCustom + 1", "retries"))
	assert.True(t, hasVariableInExpression("item", "item"))
	assert.False(t, hasVariableInExpression("foo", "item"))
	assert.True(t, hasVariableInExpression("sprig.upper(item)", "item"))
}

func Test_hasWorkflowParameters(t *testing.T) {
	expression := `workflow.status == "Succeeded" ? "SUCCESSFUL" : "FAILED"`
	exprToks, _ := lexer.Lex(file.NewSource(expression))
	variableToks, _ := lexer.Lex(file.NewSource("workflow.status"))
	variableToks = filterEOF(variableToks)
	assert.True(t, searchTokens(exprToks, variableToks))
	assert.True(t, hasVariableInExpression(expression, "workflow.status"))

	assert.False(t, hasVariableInExpression(expression, "workflow status"))
	assert.False(t, hasVariableInExpression(expression, "workflow .status"))

	expression = `"workflow.status" == "Succeeded" ? "SUCCESSFUL" : "FAILED"`
	assert.False(t, hasVariableInExpression(expression, "workflow .status"))
}
