package controller

import (
	"fmt"
	"strings"

	"github.com/valyala/fasttemplate"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/expr"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// wfScope contains the current scope of variables available when executing a template
type wfScope struct {
	tmpl  *wfv1.Template
	scope map[string]interface{}
}

func CreateScope(tmpl *wfv1.Template) *wfScope {
	scope := &wfScope{
		tmpl:  tmpl,
		scope: make(map[string]interface{}),
	}
	scope.includeTmplParamsArts()
	return scope
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

// includeTmplParamsArts include template input parameters and artifacts in scope
func (s *wfScope) includeTmplParamsArts() {
	if s.tmpl == nil {
		return
	}
	for _, param := range s.tmpl.Inputs.Parameters {
		key := fmt.Sprintf("inputs.parameters.%s", param.Name)
		s.scope[key] = s.tmpl.Inputs.GetParameterByName(param.Name).Value.String()
	}
	for _, param := range s.tmpl.Inputs.Artifacts {
		key := fmt.Sprintf("inputs.artifacts.%s", param.Name)
		s.scope[key] = s.tmpl.Inputs.GetArtifactByName(param.Name)
	}
}

// resolveVar resolves a parameter or artifact
func (s *wfScope) resolveVar(v string) (interface{}, error) {
	v = strings.TrimPrefix(v, "{{")
	v = strings.TrimSuffix(v, "}}")
	if val, ok := s.scope[v]; ok {
		return val, nil
	}
	return nil, errors.Errorf(errors.CodeBadRequest, "Unable to resolve: {{%s}}", v)
}

func (s *wfScope) resolveParameter(p *wfv1.ValueFrom) (string, error) {
	if p == nil {
		return "", nil
	}
	if p.Parameter == "" && p.Expression == "" {
		return "", nil
	}
	var val interface{}
	var err error
	if p.Expression != "" {
		val, err = expr.Eval(p.Expression, s.scope)
		return val.(string), err
	} else {
		val, err = s.resolveVar(p.Parameter)
		return val.(string), err
	}
}

func (s *wfScope) resolveArtifact(art *wfv1.Artifact) (*wfv1.Artifact, error) {
	if art == nil || (art.From == "" && art.FromExpression == "") {
		return nil, nil
	}

	var err error
	var val interface{}

	if art.FromExpression != "" {
		val, err = expr.Eval(art.FromExpression, s.scope)
	} else {
		val, err = s.resolveVar(art.From)
	}

	if err != nil {
		return nil, err
	}
	valArt, ok := val.(wfv1.Artifact)
	if !ok {
		return nil, errors.Errorf(errors.CodeBadRequest, "Variable {{%v}} is not an artifact", art)
	}

	if art.SubPath != "" {
		fstTmpl := fasttemplate.New(art.SubPath, "{{", "}}")
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
