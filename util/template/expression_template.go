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

var variablesToCheck = []string{
	"item",
	"retries",
	"lastRetry.exitCode",
	"lastRetry.status",
	"lastRetry.duration",
	"lastRetry.message",
	"workflow.status",
	"workflow.failures",
}

func anyVarNotInEnv(expression string, env map[string]any) *string {
	for _, variable := range variablesToCheck {
		if hasVariableInExpression(expression, variable) && !hasVarInEnv(env, variable) {
			return &variable
		}
	}
	return nil
}

func expressionReplace(ctx context.Context, w io.Writer, expression string, env map[string]any, allowUnresolved bool) (int, error) {
	log := logging.RequireLoggerFromContext(ctx)
	// The template is JSON-marshaled. This JSON-unmarshals the expression to undo any character escapes.
	var unmarshalledExpression string
	err := json.Unmarshal(fmt.Appendf(nil, `"%s"`, expression), &unmarshalledExpression)
	if err != nil && allowUnresolved {
		log.WithError(err).Debug(ctx, "unresolved is allowed")
		return fmt.Fprintf(w, "{{%s%s}}", kindExpression, expression)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to unmarshall JSON expression: %w", err)
	}

	varNameNotInEnv := anyVarNotInEnv(unmarshalledExpression, env)
	if varNameNotInEnv != nil && allowUnresolved {
		// this is to make sure expressions don't get resolved to nil or an empty string when certain variables
		// don't exist in the env during the "global" replacement.
		// See https://github.com/argoproj/argo-workflows/issues/5388, https://github.com/argoproj/argo-workflows/issues/15008,
		// https://github.com/argoproj/argo-workflows/issues/10393, https://github.com/expr-lang/expr/issues/330
		log.WithField("variable", *varNameNotInEnv).Debug(ctx, "variable not in env but unresolved is allowed")
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

func EnvMap(replaceMap map[string]string) map[string]any {
	envMap := make(map[string]any)
	for k, v := range replaceMap {
		envMap[k] = v
	}
	return envMap
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
		for j := range needle {
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
func hasVarInEnv(env map[string]any, parameter string) bool {
	flattenEnv := bellows.Flatten(env)
	_, ok := flattenEnv[parameter]
	return ok
}
