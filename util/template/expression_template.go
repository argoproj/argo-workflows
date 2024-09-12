package template

import (
	"encoding/json"
	"fmt"
	"github.com/argoproj/argo-workflows/v3/util/expand"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"
	log "github.com/sirupsen/logrus"
)

func init() {
	if os.Getenv("EXPRESSION_TEMPLATES") != "false" {
		registerKind(kindExpression)
	}
}

func expressionReplace(w io.Writer, expression string, env map[string]interface{}, allowUnresolved bool) (int, error) {
	// The template is JSON-marshaled. This JSON-unmarshals the expression to undo any character escapes.
	var unmarshalledExpression string
	err := json.Unmarshal([]byte(fmt.Sprintf(`"%s"`, expression)), &unmarshalledExpression)
	if err != nil && allowUnresolved {
		log.WithError(err).Debug("unresolved is allowed ")
		return w.Write([]byte(fmt.Sprintf("{{%s%s}}", kindExpression, expression)))
	}
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshall JSON expression: %w", err)
	}

	if _, ok := env["retries"]; !ok && hasRetries(unmarshalledExpression) && allowUnresolved {
		// this is to make sure expressions like `sprig.int(retries)` don't get resolved to 0 when `retries` don't exist in the env
		// See https://github.com/argoproj/argo-workflows/issues/5388
		log.WithError(err).Debug("Retries are present and unresolved is allowed")
		return w.Write([]byte(fmt.Sprintf("{{%s%s}}", kindExpression, expression)))
	}

	// This is to make sure expressions which contains `workflow.status` and `work.failures` don't get resolved to nil
	// when `workflow.status` and `workflow.failures` don't exist in the env.
	// See https://github.com/argoproj/argo-workflows/issues/10393, https://github.com/expr-lang/expr/issues/330
	// This issue doesn't happen to other template parameters since `workflow.status` and `workflow.failures` only exist in the env
	// when the exit handlers complete.
	if ((hasWorkflowStatus(unmarshalledExpression) && !hasVarInEnv(env, "workflow.status")) ||
		(hasWorkflowFailures(unmarshalledExpression) && !hasVarInEnv(env, "workflow.failures"))) &&
		allowUnresolved {
		return w.Write([]byte(fmt.Sprintf("{{%s%s}}", kindExpression, expression)))
	}

	program, err := expr.Compile(unmarshalledExpression, expr.Env(env))
	// This allowUnresolved check is not great
	// it allows for errors that are obviously
	// not failed reference checks to also pass
	if err != nil && !allowUnresolved {
		return 0, fmt.Errorf("failed to evaluate expression: %w", err)
	}
	result, err := expr.Run(program, env)
	if (err != nil || result == nil) && allowUnresolved {
		//  <nil> result is also un-resolved, and any error can be unresolved
		log.WithError(err).Debug("Result and error are unresolved")
		return w.Write([]byte(fmt.Sprintf("{{%s%s}}", kindExpression, expression)))
	}
	if err != nil {
		return 0, fmt.Errorf("failed to evaluate expression: %w", err)
	}
	if result == nil {
		return 0, fmt.Errorf("failed to evaluate expression %q", expression)
	}
	resultMarshaled, err := json.Marshal(result)
	if (err != nil || resultMarshaled == nil) && allowUnresolved {
		log.WithError(err).Debug("resultMarshaled is nil and unresolved is allowed ")
		return w.Write([]byte(fmt.Sprintf("{{%s%s}}", kindExpression, expression)))
	}
	if err != nil {
		return 0, fmt.Errorf("failed to marshal evaluated expression: %w", err)
	}
	if resultMarshaled == nil {
		return 0, fmt.Errorf("failed to marshal evaluated marshaled expression %q", expression)
	}
	marshaledLength := len(resultMarshaled)

	// Trim leading and trailing quotes. The value is being inserted into something that's already a string.
	if len(resultMarshaled) > 1 && resultMarshaled[0] == '"' && resultMarshaled[marshaledLength-1] == '"' {
		return w.Write(resultMarshaled[1 : marshaledLength-1])
	}

	resultQuoted := []byte(strconv.Quote(string(resultMarshaled)))
	return w.Write(resultQuoted[1 : len(resultQuoted)-1])
}

func EnvMap(replaceMap map[string]string) map[string]interface{} {
	envMap := make(map[string]interface{})
	for k, v := range replaceMap {
		envMap[k] = v
	}
	return envMap
}

// hasRetries checks if the variable `retries` exists in the expression template
func hasRetries(expression string) bool {
	tokens, err := lexer.Lex(file.NewSource(expression))
	if err != nil {
		return false
	}
	for _, token := range tokens {
		if token.Kind == lexer.Identifier && token.Value == "retries" {
			return true
		}
	}
	return false
}

// hasWorkflowStatus checks if expression contains `workflow.status`
func hasWorkflowStatus(expression string) bool {
	if !strings.Contains(expression, "workflow.status") {
		return false
	}
	// Even if the expression contains `workflow.status`, it could be the case that it represents a string (`"workflow.status"`),
	// not a variable, so we need to parse it and handle filter the string case.
	tokens, err := lexer.Lex(file.NewSource(expression))
	if err != nil {
		return false
	}
	for i := 0; i < len(tokens)-2; i++ {
		if tokens[i].Value+tokens[i+1].Value+tokens[i+2].Value == "workflow.status" {
			return true
		}
	}
	return false
}

// hasWorkflowFailures checks if expression contains `workflow.failures`
func hasWorkflowFailures(expression string) bool {
	if !strings.Contains(expression, "workflow.failures") {
		return false
	}
	// Even if the expression contains `workflow.failures`, it could be the case that it represents a string (`"workflow.failures"`),
	// not a variable, so we need to parse it and handle filter the string case.
	tokens, err := lexer.Lex(file.NewSource(expression))
	if err != nil {
		return false
	}
	for i := 0; i < len(tokens)-2; i++ {
		if tokens[i].Value+tokens[i+1].Value+tokens[i+2].Value == "workflow.failures" {
			return true
		}
	}
	return false
}

// hasVarInEnv checks if a parameter is in env or not
func hasVarInEnv(env map[string]interface{}, parameter string) bool {
	flattenEnv := expand.Flatten(env)
	_, ok := flattenEnv[parameter]
	return ok
}
