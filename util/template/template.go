package template

import (
	"bytes"
	log "github.com/sirupsen/logrus"
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
		log.WithFields(log.Fields{
			"tag":             tag,
			"replaceMap":      replaceMap,
			"kind":            kind,
			"expression":      expression,
			"allowUnresolved": allowUnresolved,
		}).Debug("template replace")
		switch kind {
		case kindExpression:
			env := exprenv.GetFuncMap(envMap(replaceMap))
			return expressionReplace(w, expression, env, allowUnresolved)
		default:
			return simpleReplace(w, tag, replaceMap, allowUnresolved)
		}
	})
	return replacedTmpl.String(), err
}
