package template

import (
	"testing"

	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"
	"github.com/stretchr/testify/assert"
)

func Test_hasRetries(t *testing.T) {
	t.Run("hasRetiresInExpression", func(t *testing.T) {
		assert.True(t, hasRetries("retries"))
		assert.True(t, hasRetries("retries + 1"))
		assert.True(t, hasRetries("sprig(retries)"))
		assert.True(t, hasRetries("sprig(retries + 1) * 64"))
		assert.False(t, hasRetries("foo"))
		assert.False(t, hasRetries("retriesCustom + 1"))
	})
}

func Test_hasWorkflowParameters(t *testing.T) {
	t.Run("hasVariableInExpression", func(t *testing.T) {
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
	})
}
