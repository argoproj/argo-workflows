package template

import (
	"io"
	"strings"

	"github.com/valyala/fasttemplate"
)

func Validate(s string, validator func(tag string) error) error {
	return eachTag(s, func(tag string) error {
		k, _ := parseTag(tag)
		switch k {
		case kindExpression:
			return nil // we do not validate expression templates
		default:
			return validator(tag)
		}
	})
}

// Variables returns the variables s refers to: each simple tag, trimmed, and
// each identifier and member path, such as pod.name, that an expression tag
// reads. s may be JSON-marshaled. An expression that doesn't parse adds
// nothing.
func Variables(s string) ([]string, error) {
	var vars []string
	err := eachTag(s, func(tag string) error {
		k, expression := parseTag(tag)
		switch k {
		case kindExpression:
			// Undo JSON escapes, as expressionReplaceStrict does.
			if unescaped, err := unescapeExpression(expression); err == nil {
				expression = unescaped
			}
			identifiers, _ := getIdentifiers(expression)
			vars = append(vars, identifiers...)
		default:
			vars = append(vars, strings.TrimSpace(tag))
		}
		return nil
	})
	return vars, err
}

// eachTag calls f with each tag in s, stopping at the first error.
func eachTag(s string, f func(tag string) error) error {
	t, err := fasttemplate.NewTemplate(s, prefix, suffix)
	if err != nil {
		return err
	}
	_, err = t.ExecuteFunc(io.Discard, func(_ io.Writer, tag string) (int, error) {
		return 0, f(tag)
	})
	return err
}
