package template

import (
	"bytes"
	"context"
	"io"

	"github.com/valyala/fasttemplate"

	exprenv "github.com/argoproj/argo-workflows/v4/util/expr/env"
)

const (
	prefix = "{{"
	suffix = "}}"
)

type Template interface {
	Replace(ctx context.Context, replaceMap map[string]any, allowUnresolved bool) (string, error)
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

func (t *impl) Replace(ctx context.Context, replaceMap map[string]any, allowUnresolved bool) (string, error) {
	replacedTmpl := &bytes.Buffer{}
	_, err := t.ExecuteFunc(replacedTmpl, func(w io.Writer, tag string) (int, error) {
		kind, expression := parseTag(tag)
		switch kind {
		case kindExpression:
			env := exprenv.GetFuncMap(replaceMap)
			return expressionReplace(ctx, w, expression, env, allowUnresolved)
		default:
			return simpleReplace(ctx, w, tag, replaceMap, allowUnresolved)
		}
	})
	return replacedTmpl.String(), err
}
