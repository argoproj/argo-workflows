package controller

import (
	"strings"

	"github.com/valyala/fasttemplate"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
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

func (s *wfScope) resolveParameter(v string) (string, error) {
	val, err := s.resolveVar(v)
	if err != nil {
		return "", err
	}
	valStr, ok := val.(string)
	if !ok {
		return "", errors.Errorf(errors.CodeBadRequest, "Variable {{%s}} is not a string", v)
	}
	return valStr, nil
}

func (s *wfScope) resolveArtifact(v string, subPath string) (*wfv1.Artifact, error) {
	val, err := s.resolveVar(v)
	if err != nil {
		return nil, err
	}
	valArt, ok := val.(wfv1.Artifact)
	if !ok {
		return nil, errors.Errorf(errors.CodeBadRequest, "Variable {{%s}} is not an artifact", v)
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
