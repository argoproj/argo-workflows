package template

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"strings"

	"github.com/valyala/fasttemplate"

	exprenv "github.com/argoproj/argo-workflows/v4/util/expr/env"
)

const (
	prefix = "{{"
	suffix = "}}"
)

type Template interface {
	Replace(ctx context.Context, replaceMap map[string]any, allowUnresolved bool) (string, error)
	ReplaceStrict(ctx context.Context, replaceMap map[string]any, strictPrefixes []string) (string, error)
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

func (t *impl) replace(ctx context.Context, replaceMap map[string]any, simpleStrictRegex, expressionStrictRegex *regexp.Regexp, allowUnresolvedArtifacts bool) (string, error) {
	replacedTmpl := &bytes.Buffer{}
	_, err := t.ExecuteFunc(replacedTmpl, func(w io.Writer, tag string) (int, error) {
		kind, expression := parseTag(tag)
		switch kind {
		case kindExpression:
			env := exprenv.GetFuncMap(replaceMap)
			return expressionReplaceStrict(ctx, w, expression, env, expressionStrictRegex)
		default:
			return simpleReplaceStrict(ctx, w, tag, replaceMap, simpleStrictRegex, allowUnresolvedArtifacts)
		}
	})
	return replacedTmpl.String(), err
}

func (t *impl) Replace(ctx context.Context, replaceMap map[string]any, allowUnresolved bool) (string, error) {
	var regex *regexp.Regexp
	if !allowUnresolved {
		regex = matchAllRegex
	}
	return t.replace(ctx, replaceMap, regex, regex, allowUnresolved)
}

func (t *impl) ReplaceStrict(ctx context.Context, replaceMap map[string]any, strictPrefixes []string) (string, error) {
	var strictRegex *regexp.Regexp
	var expressionStrictRegex *regexp.Regexp
	if len(strictPrefixes) > 0 {
		var patterns []string
		var expressionPatterns []string
		for _, p := range strictPrefixes {
			patterns = append(patterns, regexp.QuoteMeta(p))
			expressionPatterns = append(expressionPatterns, regexp.QuoteMeta(strings.SplitN(p, ".", 2)[0]))
		}
		// Match any string starting with one of the prefixes
		regexStr := "^(" + strings.Join(patterns, "|") + ")"
		strictRegex = regexp.MustCompile(regexStr)

		expressionRegexStr := "^(" + strings.Join(expressionPatterns, "|") + ")$"
		expressionStrictRegex = regexp.MustCompile(expressionRegexStr)
	}

	return t.replace(ctx, replaceMap, strictRegex, expressionStrictRegex, true)
}