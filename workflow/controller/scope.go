package controller

import (
	"fmt"
	"strings"

	"github.com/Knetic/govaluate"
	"github.com/valyala/fasttemplate"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

// wfScope contains the current scope of variables available when executing a template
type wfScope struct {
	tmpl  *wfv1.Template
	scope map[string]interface{}
}

// getParameters returns a map of strings intended to be used simple string substitution
func (s *wfScope) getParameters() common.Parameters {
	params := make(common.Parameters)
	for key, val := range s.scope {
		valStr, ok := val.(string)
		if ok {
			params[key] = valStr
		}
	}
	return params
}

func (s *wfScope) addParamToScope(key, val string) {
	s.scope[key] = val
}

func (s *wfScope) addArtifactToScope(key string, artifact wfv1.Artifact) {
	s.scope[key] = artifact
}

// resolveVar resolves a parameter or artifact
func (s *wfScope) resolveVar(v string) (interface{}, error) {
	v = strings.TrimPrefix(v, "{{")
	v = strings.TrimSuffix(v, "}}")
	parts := strings.Split(v, ".")
	prefix := parts[0]
	switch prefix {
	case "steps", "tasks", "workflow":
		val, ok := s.scope[v]
		if ok {
			return val, nil
		}
	case "inputs":
		art := s.tmpl.Inputs.GetArtifactByName(parts[2])
		if art != nil {
			return *art, nil
		}
	}
	return nil, errors.Errorf(errors.CodeBadRequest, "Unable to resolve: {{%s}}", v)
}

func (s *wfScope) resolveParameter(p *wfv1.ValueFrom) (string, error) {
	if p == nil {
		return "", nil
	}
	if p.Parameter == "" && p.FromExpression == "" {
		return "", nil
	}
	param := p.Parameter
	var err error
	if p.FromExpression != "" {
		param, err = s.evaluateExpression(p.FromExpression)
		if err != nil {
			return "", fmt.Errorf("unable to resolve expression: %s", err)
		}
		return param, nil
	}
	val, err := s.resolveVar(param)
	if err != nil {
		return "", err
	}
	valStr, ok := val.(string)
	if !ok {
		return "", errors.Errorf(errors.CodeBadRequest, "Variable {{%s}} is not a string", param)
	}
	return valStr, nil
}

func (s *wfScope) evaluateExpression(fromExpression string) (string, error) {
	if fromExpression == "" {
		return "", nil
	}
	fstTmpl := fasttemplate.New(fromExpression, "{{", "}}")
	updateExp, err := common.Replace(fstTmpl, s.getParameters(), true)
	if err != nil {
		return "", err
	}

	updateExp = strings.Replace(updateExp, "{{", "\"{{", -1)
	updateExp = strings.Replace(updateExp, "}}", "}}\"", -1)
	expression, err := govaluate.NewEvaluableExpression(updateExp)
	if err != nil {
		if strings.Contains(err.Error(), "Invalid token") {
			return "", errors.Errorf(errors.CodeBadRequest, "Invalid 'fromExpression' '%s': %v (hint: try wrapping the affected expression in quotes (\"))", expression, err)
		}
		return "", errors.Errorf(errors.CodeBadRequest, "Invalid 'fromExpression' '%s': %v", expression, err)
	}
	// The following loop converts govaluate variables (which we don't use), into strings. This
	// allows us to have expressions like: "foo != bar" without requiring foo and bar to be quoted.
	tokens := expression.Tokens()
	for i, tok := range tokens {
		switch tok.Kind {
		case govaluate.VARIABLE:
			tok.Kind = govaluate.STRING
		default:
			continue
		}
		tokens[i] = tok
	}
	expression, err = govaluate.NewEvaluableExpressionFromTokens(tokens)
	if err != nil {
		return "", errors.InternalWrapErrorf(err, "Failed to parse 'fromExpression''%s': %v", expression, err)
	}
	result, err := expression.Evaluate(nil)
	if err != nil {
		return "", errors.InternalWrapErrorf(err, "Failed to evaluate 'fromExpression' '%s': %v", expression, err)
	}
	if result == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", result), nil
}

func (s *wfScope) resolveArtifact(art *wfv1.Artifact, subPath string) (*wfv1.Artifact, error) {
	if art == nil {
		return nil, nil
	}
	if art.From == "" && art.FromExpression == "" {
		return nil, nil
	}
	artReference := art.From
	var err error
	if art.FromExpression != "" {
		artReference, err = s.evaluateExpression(art.FromExpression)
		if err != nil {
			if art.Optional {
				return nil, nil
			}
			return nil, fmt.Errorf("unable to resolve expression: %s", err)
		}
	}

	if artReference == "" {
		return nil, nil
	}
	val, err := s.resolveVar(artReference)

	if err != nil {
		return nil, err
	}
	valArt, ok := val.(wfv1.Artifact)
	if !ok {
		return nil, errors.Errorf(errors.CodeBadRequest, "Variable {{%s}} is not an artifact", artReference)
	}

	if subPath != "" {
		fstTmpl := fasttemplate.New(subPath, "{{", "}}")
		resolvedSubPath, err := common.Replace(fstTmpl, s.getParameters(), true)
		if err != nil {
			return nil, err
		}

		// Copy resolved artifact pointer before adding subpath
		copyArt := valArt.DeepCopy()
		return copyArt, copyArt.AppendToKey(resolvedSubPath)
	}

	return &valArt, nil
}
