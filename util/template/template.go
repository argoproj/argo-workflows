package template

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	exprenv "github.com/argoproj/argo-workflows/v3/util/expr/env"
	"github.com/pkg/errors"
)

const (
	prefix = "{{"
	suffix = "}}"
)

type Template interface {
	Replace(replaceMap map[string]string, allowUnresolved bool) (string, error)
}

func replaceTemplate(x string) string {
	x = strings.ReplaceAll(x, "{{", "")
	x = strings.ReplaceAll(x, "}}", "")
	return fmt.Sprintf(`{{execFunc .X .Y %s}}`, strconv.Quote(x))
}

func NewTemplate(s string) (Template, error) {
	funcMap := template.FuncMap{
		"execFunc": execFunc,
	}
	re := regexp.MustCompile(`{{(.*)}}`)
	tmpl, err := template.New("NewTemplate").Funcs(funcMap).Parse(re.ReplaceAllStringFunc(s, replaceTemplate))
	if err != nil {
		return nil, err
	}
	return &impl{tmpl}, nil
}

func execFunc(replaceMap map[string]string, allowUnresolved bool, tag string) (string, error) {
	w := &bytes.Buffer{}
	kind, expression := parseTag(tag)
	switch kind {
	case kindExpression:
		env := exprenv.GetFuncMap(EnvMap(replaceMap))
		_, err := expressionReplace(w, expression, env, allowUnresolved)
		if err != nil {
			return "", err
		}
		return w.String(), nil
	default:
		_, err := simpleReplace(w, tag, replaceMap, allowUnresolved)
		if err != nil {
			return "", err
		}
		return w.String(), nil
	}
}

type impl struct {
	*template.Template
}

func (t *impl) Replace(replaceMap map[string]string, allowUnresolved bool) (string, error) {
	replacedTmpl := &bytes.Buffer{}
	data := struct {
		X map[string]string
		Y bool
	}{
		replaceMap,
		allowUnresolved,
	}

	err := t.Execute(replacedTmpl, data)
	if err != nil {
		return "", errors.Unwrap(errors.Unwrap(err))
	}
	return replacedTmpl.String(), err
}
