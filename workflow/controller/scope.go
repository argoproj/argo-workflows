package controller

import (
	"encoding/json"
	"fmt"

	"github.com/expr-lang/expr"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/expr/env"
	"github.com/argoproj/argo-workflows/v3/util/template"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

// wfScope contains the current scope of variables available when executing a template
type wfScope struct {
	tmpl  *wfv1.Template
	scope map[string]interface{}
}

func createScope(tmpl *wfv1.Template) *wfScope {
	scope := &wfScope{
		tmpl:  tmpl,
		scope: make(map[string]interface{}),
	}
	if tmpl != nil {
		for _, param := range scope.tmpl.Inputs.Parameters {
			key := fmt.Sprintf("inputs.parameters.%s", param.Name)
			scope.scope[key] = scope.tmpl.Inputs.GetParameterByName(param.Name).Value.String()
		}
		for _, param := range scope.tmpl.Inputs.Artifacts {
			key := fmt.Sprintf("inputs.artifacts.%s", param.Name)
			scope.scope[key] = scope.tmpl.Inputs.GetArtifactByName(param.Name)
		}
	}
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

// resolveVar resolves a parameter or artifact
func (s *wfScope) resolveVar(v string) (interface{}, error) {
	m := make(map[string]interface{})
	for k, v := range s.scope {
		m[k] = v
	}
	if s.tmpl != nil {
		for _, a := range s.tmpl.Inputs.Artifacts {
			m["inputs.artifacts."+a.Name] = a // special case for artifacts
		}
	}
	return template.ResolveVar(v, m)
}

func (s *wfScope) resolveParameter(p *wfv1.ValueFrom) (interface{}, error) {
	if p == nil || (p.Parameter == "" && p.Expression == "") {
		return "", nil
	}
	if p.Expression != "" {
		env := env.GetFuncMap(s.scope)
		program, err := expr.Compile(p.Expression, expr.Env(env))
		if err != nil {
			return nil, err
		}
		return expr.Run(program, env)
	} else {
		return s.resolveVar(p.Parameter)
	}
}

func (s *wfScope) resolveArtifact(art *wfv1.Artifact) (*wfv1.Artifact, error) {
	if art == nil || (art.From == "" && art.FromExpression == "") {
		return nil, nil
	}

	var err error
	var val interface{}

	if art.FromExpression != "" {
		env := env.GetFuncMap(s.scope)
		program, err := expr.Compile(art.FromExpression, expr.Env(env))
		if err != nil {
			return nil, err
		}
		val, err = expr.Run(program, env)
		if err != nil {
			return nil, err
		}

	} else {
		val, err = s.resolveVar(art.From)
	}

	if err != nil {
		return nil, err
	}
	valArt, ok := val.(wfv1.Artifact)
	if !ok {
		//If the workflow refers itself input artifacts in fromExpression, the val type is "*wfv1.Artifact"
		ptArt, ok := val.(*wfv1.Artifact)
		if ok {
			valArt = *ptArt
		} else {
			return nil, errors.Errorf(errors.CodeBadRequest, "Variable {{%v}} is not an artifact", art)
		}
	}

	if art.SubPath != "" {
		// Copy resolved artifact pointer before adding subpath
		copyArt := valArt.DeepCopy()

		subPathAsJson, err := json.Marshal(art.SubPath)
		if err != nil {
			return copyArt, errors.New(errors.CodeBadRequest, "failed to marshal artifact subpath for templating")
		}

		resolvedSubPathAsJson, err := template.Replace(string(subPathAsJson), s.getParameters(), true)
		if err != nil {
			return nil, err
		}

		var resolvedSubPath string
		err = json.Unmarshal([]byte(resolvedSubPathAsJson), &resolvedSubPath)
		if err != nil {
			return copyArt, errors.New(errors.CodeBadRequest, "failed to unmarshal artifact subpath for templating")
		}

		err = copyArt.AppendToKey(resolvedSubPath)
		if err != nil && copyArt.Optional { //Ignore error when artifact optional
			return copyArt, nil
		}
		return copyArt, err
	}

	return &valArt, nil
}
