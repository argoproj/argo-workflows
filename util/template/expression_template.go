package template

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/file"
	"github.com/antonmedv/expr/parser/lexer"
	log "github.com/sirupsen/logrus"
)

func init() {
	if os.Getenv("EXPRESSION_TEMPLATES") != "false" {
		registerKind(kindExpression)
	}
}

func expressionReplace(w io.Writer, expression string, env map[string]interface{}, allowUnresolved bool) (int, error) {
	log := log.WithField("ALEX", "true")
	// The template is JSON-marshaled. This JSON-unmarshals the expression to undo any character escapes.
	var unmarshalledExpression string
	err := json.Unmarshal([]byte(fmt.Sprintf(`"%s"`, expression)), &unmarshalledExpression)
	if err != nil && allowUnresolved {
		log.WithError(err).WithField("allowUnresolved", allowUnresolved).Debug("expression template noop")
		return w.Write([]byte(fmt.Sprintf("{{%s%s}}", kindExpression, expression)))
	}
	if err != nil {
		log.WithError(err).Debug()
		return 0, fmt.Errorf("failed to unmarshall JSON expression: %w", err)
	}

	if _, ok := env["retries"]; !ok && hasRetries(unmarshalledExpression) && allowUnresolved {
		// this is to make sure expressions like `sprig.int(retries)` don't get resolved to 0 when `retries` don't exist in the env
		// See https://github.com/argoproj/argo-workflows/issues/5388
		log.WithError(err).WithField("allowUnresolved", allowUnresolved).WithField("hasRetries", true).Debug("expression template noop")
		return w.Write([]byte(fmt.Sprintf("{{%s%s}}", kindExpression, expression)))
	}
	result, err := expr.Eval(unmarshalledExpression, env)
	if (err != nil || result == nil) && allowUnresolved { //  <nil> result is also un-resolved, and any error can be unresolved
		log.WithError(err).WithField("result", result).WithField("allowUnresolved", allowUnresolved).Debug("expression template noop")
		return w.Write([]byte(fmt.Sprintf("{{%s%s}}", kindExpression, expression)))
	}
	if err != nil {
		log.WithError(err).Debug()
		return 0, fmt.Errorf("failed to evaluate expression: %w", err)
	}
	if result == nil {
		log.Debug("result is nil")
		return 0, fmt.Errorf("failed to evaluate expression %q", expression)
	}
	resultMarshaled, err := json.Marshal(fmt.Sprintf("%v", result))
	if (err != nil || resultMarshaled == nil) && allowUnresolved {
		log.WithError(err).WithField("resultMarshaled", resultMarshaled).WithField("allowUnresolved", allowUnresolved).Debug("expression template noop")
		return w.Write([]byte(fmt.Sprintf("{{%s%s}}", kindExpression, expression)))
	}
	if err != nil {
		log.WithError(err).Debug()
		return 0, fmt.Errorf("failed to marshal evaluated expression: %w", err)
	}
	if resultMarshaled == nil {
		log.Debug("resultMarshaled is nil")
		return 0, fmt.Errorf("failed to marshal evaluated marshaled expression %q", expression)
	}
	// Trim leading and trailing quotes. The value is being inserted into something that's already a string.
	marshaledLength := len(resultMarshaled)
	return w.Write(resultMarshaled[1 : marshaledLength-1])
}

func envMap(replaceMap map[string]string) map[string]interface{} {
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
