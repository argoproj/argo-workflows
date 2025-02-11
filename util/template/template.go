package template

import (
	"bytes"
	"io"

	"github.com/valyala/fasttemplate"

	exprenv "github.com/argoproj/argo-workflows/v3/util/expr/env"
)

const (
	prefix = "{{"
	suffix = "}}"
)

type Template interface {
	Replace(replaceMap map[string]string, allowUnresolved bool) (string, error)
}

func NewTemplate(s string) (Template, error) {
	template, err := fasttemplate.NewTemplate(s, prefix, suffix)
	if err != nil {
		return nil, err
	}
	return &impl{template}, nil
}

type impl struct {
	*fasttemplate.Template
}

func (t *impl) Replace(replaceMap map[string]string, allowUnresolved bool) (string, error) {
	replacedTmpl := &bytes.Buffer{}
	_, err := t.Template.ExecuteFunc(replacedTmpl, func(w io.Writer, tag string) (int, error) {
		kind, expression := parseTag(tag)
		switch kind {
		case kindExpression:
			env := exprenv.GetFuncMap(EnvMap(replaceMap))
			return expressionReplace(w, expression, env, allowUnresolved)
		default:
			return simpleReplace(w, tag, replaceMap, allowUnresolved)
		}
	})
	return replacedTmpl.String(), err
}
