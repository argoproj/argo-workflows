package template

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/ast"
	"github.com/expr-lang/expr/file"
	"github.com/expr-lang/expr/parser"
	"github.com/expr-lang/expr/parser/lexer"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/maps"
)

func init() {
	if os.Getenv("EXPRESSION_TEMPLATES") != "false" {
		registerKind(kindExpression)
	}
}

var (
	variablesToCheck = []string{
		"item",
		"retries",
		"lastRetry.exitCode",
		"lastRetry.status",
		"lastRetry.duration",
		"lastRetry.message",
		"workflow.status",
		"workflow.failures",
	}
)

func anyVarNotInEnv(expression string, env map[string]any) *string {
	for _, variable := range variablesToCheck {
		if hasVariableInExpression(expression, variable) && !hasVarInEnv(env, variable) {
			return &variable
		}
	}
	return nil
}

func expressionReplaceStrict(ctx context.Context, w io.Writer, expression string, env map[string]any, strictRegex *regexp.Regexp) (int, error) {
	// The template is JSON-marshaled. This JSON-unmarshals the expression to undo any character escapes.
	var unmarshalledExpression string
	err := json.Unmarshal(fmt.Appendf(nil, `"%s"`, expression), &unmarshalledExpression)
	if err != nil {
		// If we can't unmarshal, we can't parse. Fallback to expressionReplaceCore to handle it (likely error).
		return expressionReplaceCore(ctx, w, expression, env, false)
	}

	identifiers, err := getIdentifiers(unmarshalledExpression)
	if err != nil {
		// If we can't parse, we can't check variables. Fallback to expressionReplaceCore(false) to report syntax error.
		return expressionReplaceCore(ctx, w, expression, env, false)
	}

	missingIdentifiers := []string{}
	for _, id := range identifiers {
		if !hasVarInEnv(env, id) {
			missingIdentifiers = append(missingIdentifiers, id)
		}
	}

	for _, id := range missingIdentifiers {
		if strictRegex != nil && strictRegex.MatchString(id) {
			return 0, fmt.Errorf("failed to evaluate expression: %s is missing", id)
		}
	}

	// If we have missing identifiers but they are NOT strict, we allow unresolved.
	// If we have NO missing identifiers, we enforce resolution (to catch runtime errors).
	allowUnresolved := len(missingIdentifiers) > 0
	return expressionReplaceCore(ctx, w, expression, env, allowUnresolved)
}

type identifierVisitor struct {
	identifiers []string
	seen        map[string]bool
}

func (v *identifierVisitor) Visit(node *ast.Node) {
	if n, ok := (*node).(*ast.IdentifierNode); ok {
		if !v.seen[n.Value] {
			v.identifiers = append(v.identifiers, n.Value)
			v.seen[n.Value] = true
		}
	}
	if n, ok := (*node).(*ast.MemberNode); ok {
		path, ok := getMemberPath(n)
		if ok {
			if !v.seen[path] {
				v.identifiers = append(v.identifiers, path)
				v.seen[path] = true
			}
		}
	}
}

func getMemberPath(node *ast.MemberNode) (string, bool) {
	var parts []string
	curr := node
	for {
		prop, ok := curr.Property.(*ast.StringNode)
		if !ok {
			return "", false
		}
		parts = append([]string{prop.Value}, parts...)

		if id, ok := curr.Node.(*ast.IdentifierNode); ok {
			parts = append([]string{id.Value}, parts...)
			return strings.Join(parts, "."), true
		}

		if next, ok := curr.Node.(*ast.MemberNode); ok {
			curr = next
		} else {
			return "", false
		}
	}
}

func getIdentifiers(expression string) ([]string, error) {
	tree, err := parser.Parse(expression)
	if err != nil {
		return nil, err
	}
	visitor := &identifierVisitor{
		seen: make(map[string]bool),
	}
	ast.Walk(&tree.Node, visitor)
	return visitor.identifiers, nil
}

func expressionReplace(ctx context.Context, w io.Writer, expression string, env map[string]any, allowUnresolved bool) (int, error) {
	var strictRegex *regexp.Regexp
	if !allowUnresolved {
		strictRegex = matchAllRegex
	}
	return expressionReplaceStrict(ctx, w, expression, env, strictRegex)
}

func expressionReplaceCore(ctx context.Context, w io.Writer, expression string, env map[string]any, allowUnresolved bool) (int, error) {

	shouldAllowFailure := false

	maps.VisitMap(env, func(key string, value any) bool {
		rv := reflect.Indirect(reflect.ValueOf(value))
		if rv.Kind() == reflect.String {
			placeholder := strings.HasPrefix(rv.String(), "__argo__internal__placeholder")
			if placeholder {
				shouldAllowFailure = true
				return false
			}
		}
		return true
	})

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
	if err != nil && !allowUnresolved && !shouldAllowFailure {
		return 0, fmt.Errorf("failed to evaluate expression: %w", err)
	}
	result, err := expr.Run(program, env)
	if (err != nil || result == nil) && (allowUnresolved || shouldAllowFailure) {
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
	if _, ok := env[parameter]; ok {
		return true
	}

	parts := strings.Split(parameter, ".")
	var current any
	found := false
	remainingParts := parts

	// Try to find the longest matching prefix in env
	for i := len(parts); i > 0; i-- {
		prefix := strings.Join(parts[:i], ".")
		if val, ok := env[prefix]; ok {
			current = val
			remainingParts = parts[i:]
			found = true
			break
		}
	}

	if !found {
		// If no prefix found, start from env itself (if env is the root object)
		// But in our case env is a map[string]any, so if no key matched, we probably can't traverse.
		// However, let's keep existing behavior: start traversing from env as if it's the root.
		current = env
		remainingParts = parts
	}

	// Traverse the remaining parts
	for i, part := range remainingParts {
		if current == nil {
			return false
		}

		rVal := reflect.ValueOf(current)
		for rVal.Kind() == reflect.Ptr {
			if rVal.IsNil() {
				return false
			}
			rVal = rVal.Elem()
		}

		switch rVal.Kind() {
		case reflect.Map:
			val := rVal.MapIndex(reflect.ValueOf(part))
			if !val.IsValid() {
				return false
			}
			current = val.Interface()
		case reflect.Struct:
			field := rVal.FieldByName(part)
			if !field.IsValid() {
				// Search anonymous fields manually to ensure we find embedded fields
				for j := 0; j < rVal.NumField(); j++ {
					fType := rVal.Type().Field(j)
					if fType.Anonymous {
						embeddedValue := rVal.Field(j)
						// Handle pointer to embedded struct
						for embeddedValue.Kind() == reflect.Ptr {
							if embeddedValue.IsNil() {
								break
							}
							embeddedValue = embeddedValue.Elem()
						}
						if embeddedValue.Kind() == reflect.Struct {
							// If we are looking for the embedded type itself (e.g. "Time" in metav1.Time)
							if fType.Name == part {
								field = rVal.Field(j)
								break
							}
							if foundField := embeddedValue.FieldByName(part); foundField.IsValid() {
								field = foundField
								break
							}
						}
					}
				}
			}

			if !field.IsValid() {
				return false
			}
			if !field.CanInterface() {
				return false
			}
			current = field.Interface()
		default:
			return false
		}

		// If this was the last part, we found it
		if i == len(remainingParts)-1 {
			return true
		}
	}

	return found && len(remainingParts) == 0
}
