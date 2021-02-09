package controller

import (
	"fmt"
	"strings"

	"github.com/valyala/fasttemplate"

	"github.com/argoproj/argo/v3/errors"
	wfv1 "github.com/argoproj/argo/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/v3/util/expr"
	"github.com/argoproj/argo/v3/workflow/common"
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
		fmt.Println(val, ok)
		if ok {
			fmt.Println(val, ok)
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
	if p.Parameter == "" && p.Expression == "" {
		return "", nil
	}
	var val interface{}
	var err error
	if p.Expression != "" {
		val, err = expr.Eval(p.Expression, s.getAllParamArtifact())
	} else {
		val, err = s.resolveVar(p.Parameter)
	}

	if err != nil {
		return "", fmt.Errorf("unable to resolve expression: %s", err)
	}
	if val == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", val), nil
}

func (s *wfScope) getAllParamArtifact() map[string]interface{} {

	paramArtMap := make(map[string]interface{})
	for key, val := range s.scope {
		if _, ok := val.(*wfv1.AnyString); ok {
			paramArtMap[strings.TrimSpace(key)] = val.(*wfv1.AnyString).Value()
		} else {
			paramArtMap[strings.TrimSpace(key)] = val
		}
	}
	for _, param := range s.tmpl.Inputs.Parameters {
		key := fmt.Sprintf("inputs.parameters.%s", strings.TrimSpace(param.Name))
		paramArtMap[strings.TrimSpace(key)] = s.tmpl.Inputs.GetParameterByName(param.Name).Value.Value()
	}
	for _, param := range s.tmpl.Inputs.Artifacts {
		key := fmt.Sprintf("inputs.artifacts.%s", param.Name)
		paramArtMap[strings.TrimSpace(key)] = s.tmpl.Inputs.GetArtifactByName(param.Name)
	}
	return paramArtMap
}

func (s *wfScope) resolveArtifact(art *wfv1.Artifact, subPath string) (*wfv1.Artifact, error) {
	if art == nil {
		return nil, nil
	}
	if art.From == "" && art.FromExpression == "" {
		return nil, nil
	}
	var err error
	var val interface{}
	if art.FromExpression != "" {
		val, err = expr.Eval(art.FromExpression, s.getAllParamArtifact())
	} else {
		val, err = s.resolveVar(art.From)
	}
	if err != nil {
		return nil, fmt.Errorf("unable to resolve artifact: %s", err)
	}

	valArt, ok := val.(wfv1.Artifact)
	if !ok {
		return nil, errors.Errorf(errors.CodeBadRequest, "Variable {{%v}} is not an artifact", art)
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
