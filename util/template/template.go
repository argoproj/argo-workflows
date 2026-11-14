package template

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"strings"

	"github.com/valyala/fasttemplate"

	exprenv "github.com/argoproj/argo-workflows/v4/util/expr/env"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
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

func (t *impl) replace(ctx context.Context, replaceMap map[string]any, regex *regexp.Regexp, allowUnresolvedArtifacts bool) (string, error) {
	replacedTmpl := &bytes.Buffer{}
	_, err := t.ExecuteFunc(replacedTmpl, func(w io.Writer, tag string) (int, error) {
		kind, expression := parseTag(tag)
		switch kind {
		case kindExpression:
			env := exprenv.GetFuncMap(replaceMap)
			return expressionReplaceStrict(ctx, w, expression, env, regex)
		default:
			return simpleReplaceStrict(ctx, w, tag, replaceMap, regex, allowUnresolvedArtifacts)
		}
	})
	return replacedTmpl.String(), err
}

func (t *impl) Replace(ctx context.Context, replaceMap map[string]any, allowUnresolved bool) (string, error) {
	var regex *regexp.Regexp
	if !allowUnresolved {
		regex = matchAllRegex
	}
	return t.replace(ctx, replaceMap, regex, allowUnresolved)
}

func getStrictRegex(strictPrefixes []string) *regexp.Regexp {
	if len(strictPrefixes) == 0 {
		return nil
	}
	var patterns []string
	for _, p := range strictPrefixes {
		patterns = append(patterns, regexp.QuoteMeta(p))
	}
	// Match any string starting with one of the prefixes, followed by a dot or the end of the string.
	// This ensures that we match a full dot-segment (e.g., "tasks" matches "tasks.A" but not "tasksA").
	regexStr := "^((" + strings.Join(patterns, "|") + "))(\\.|$)"
	return regexp.MustCompile(regexStr)
}

// isStrictTag reports whether tag matches strictRegex and so must resolve.
// Under strict prefixes, {{tasks.name}} and {{steps.name}} match but name the
// current task or step, which is only set when it is executed: they are not
// references to a dependency, so they may stay unresolved for the pass that
// runs at execution (#17415). Replace with allowUnresolved=false
// (matchAllRegex) requires every tag to resolve, so it has no such exemption.
func isStrictTag(strictRegex *regexp.Regexp, tag string) bool {
	if strictRegex == nil {
		return false
	}
	if strictRegex != matchAllRegex && (tag == varkeys.TasksName.Template() || tag == varkeys.StepsName.Template()) {
		return false
	}
	return strictRegex.MatchString(tag)
}

func (t *impl) ReplaceStrict(ctx context.Context, replaceMap map[string]any, strictPrefixes []string) (string, error) {
	strictRegex := getStrictRegex(strictPrefixes)
	return t.replace(ctx, replaceMap, strictRegex, true)
}
