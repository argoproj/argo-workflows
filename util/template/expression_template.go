package template

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/doublerebel/bellows"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser/lexer"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func init() {
	if os.Getenv("EXPRESSION_TEMPLATES") != "false" {
		registerKind(kindExpression)
	}
}

func anyVarNotInEnv(expression string, variables []string, env map[string]interface{}) bool {
	for _, variable := range variables {
		if hasVariableInExpression(expression, variable) && !hasVarInEnv(env, variable) {
			return true
		}
	}
	return false
}

func expressionReplace(ctx context.Context, w io.Writer, expression string, env map[string]interface{}, allowUnresolved bool) (int, error) {
	log := logging.RequireLoggerFromContext(ctx)
	// The template is JSON-marshaled. This JSON-unmarshals the expression to undo any character escapes.
	var unmarshalledExpression string
	err := json.Unmarshal([]byte(fmt.Sprintf(`"%s"`, expression)), &unmarshalledExpression)
	if err != nil && allowUnresolved {
		log.WithError(err).Debug(ctx, "unresolved is allowed ")
		return fmt.Fprintf(w, "{{%s%s}}", kindExpression, expression)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshall JSON expression: %w", err)
	}

	if anyVarNotInEnv(unmarshalledExpression, []string{"retries"}, env) && allowUnresolved {
		// this is to make sure expressions like `sprig.int(retries)` don't get resolved to 0 when `retries` don't exist in the env
		// See https://github.com/argoproj/argo-workflows/issues/5388
		log.WithError(err).Debug(ctx, "Retries are present and unresolved is allowed")
		return fmt.Fprintf(w, "{{%s%s}}", kindExpression, expression)
	}

	lastRetryVariables := []string{"lastRetry.exitCode", "lastRetry.status", "lastRetry.duration", "lastRetry.message"}
	if anyVarNotInEnv(unmarshalledExpression, lastRetryVariables, env) && allowUnresolved {
		// This is to make sure expressions which contains `lastRetry.*` don't get resolved to nil
		// when they don't exist in the env.
		log.WithError(err).Debug(ctx, "LastRetry variables are present and unresolved is allowed")
		return fmt.Fprintf(w, "{{%s%s}}", kindExpression, expression)
	}

	// This is to make sure expressions which contains `workflow.status` and `work.failures` don't get resolved to nil
	// when `workflow.status` and `workflow.failures` don't exist in the env.
	// See https://github.com/argoproj/argo-workflows/issues/10393, https://github.com/expr-lang/expr/issues/330
	// This issue doesn't happen to other template parameters since `workflow.status` and `workflow.failures` only exist in the env
	// when the exit handlers complete.
	if anyVarNotInEnv(unmarshalledExpression, []string{"workflow.status", "workflow.failures"}, env) && allowUnresolved {
		log.WithError(err).Debug(ctx, "workflow.status or workflow.failures are present and unresolved is allowed")
		return fmt.Fprintf(w, "{{%s%s}}", kindExpression, expression)
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
		log.WithError(err).Debug(ctx, "Result and error are unresolved")
		return fmt.Fprintf(w, "{{%s%s}}", kindExpression, expression)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to evaluate expression: %w", err)
	}
	if result == nil {
		return 0, fmt.Errorf("failed to evaluate expression %q", expression)
	}
	resultMarshaled, err := json.Marshal(result)
	if (err != nil || resultMarshaled == nil) && allowUnresolved {
		log.WithError(err).Debug(ctx, "resultMarshaled is nil and unresolved is allowed ")
		return fmt.Fprintf(w, "{{%s%s}}", kindExpression, expression)
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
	return hasVariableInExpression(expression, "retries")
}

func searchTokens(haystack []lexer.Token, needle []lexer.Token) bool {
	if len(needle) > len(haystack) {
		return false
	}
	if len(needle) == 0 {
		return true
	}
outer:
	for i := 0; i <= len(haystack)-len(needle); i++ {
		for j := 0; j < len(needle); j++ {
			if haystack[i+j].String() != needle[j].String() {
				continue outer
			}
		}
		return true
	}
	return false
}

func filterEOF(toks []lexer.Token) []lexer.Token {
	newToks := []lexer.Token{}
	for _, tok := range toks {
		if tok.Kind != lexer.EOF {
			newToks = append(newToks, tok)
		}
	}
	return newToks
}

// hasVariableInExpression checks if an expression contains a variable.
// This function is somewhat cursed and I have attempted my best to
// remove this curse, but it still exists.
// The strings.Contains is needed because the lexer doesn't do
// any whitespace processing (workflow .status will be seen as workflow.status)
func hasVariableInExpression(expression, variable string) bool {
	if !strings.Contains(expression, variable) {
		return false
	}
	tokens, err := lexer.Lex(file.NewSource(expression))
	if err != nil {
		return false
	}
	variableTokens, err := lexer.Lex(file.NewSource(variable))
	if err != nil {
		return false
	}
	variableTokens = filterEOF(variableTokens)

	return searchTokens(tokens, variableTokens)
}

// hasVarInEnv checks if a parameter is in env or not
func hasVarInEnv(env map[string]interface{}, parameter string) bool {
	flattenEnv := bellows.Flatten(env)
	_, ok := flattenEnv[parameter]
	return ok
}
