package controller

import (
	"context"
	"encoding/json"

	"github.com/expr-lang/expr"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/expr/env"
	"github.com/argoproj/argo-workflows/v4/util/template"
	"github.com/argoproj/argo-workflows/v4/util/variables"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// wfScope contains the current scope of variables available when executing a
// template. The underlying *variables.Scope has no exported subscript or
// write helper, so the only way to populate it is via Key.Set — which means
// the only writable variables are those declared in util/variables/keys.
type wfScope struct {
	tmpl  *wfv1.Template
	scope *variables.Scope
}

func createScope(tmpl *wfv1.Template) *wfScope {
	scope := &wfScope{
		tmpl:  tmpl,
		scope: variables.NewScope(),
	}
	if tmpl != nil {
		for _, param := range scope.tmpl.Inputs.Parameters {
			val := scope.tmpl.Inputs.GetParameterByName(param.Name).Value.String()
			varkeys.InputsParameterByName.Set(scope.scope, val, param.Name)
		}
		for _, param := range scope.tmpl.Inputs.Artifacts {
			art := scope.tmpl.Inputs.GetArtifactByName(param.Name)
			varkeys.InputsArtifactByName.Set(scope.scope, art, param.Name)
		}
	}
	return scope
}

// getParameters returns a string-only snapshot of the scope, suitable for
// passing into common.Parameters consumers.
func (s *wfScope) getParameters() common.Parameters {
	return common.Parameters(s.scope.AsStringMap())
}

// resolveVar resolves a parameter or artifact
func (s *wfScope) resolveVar(v string) (any, error) {
	m := s.scope.AsAnyMap()
	if s.tmpl != nil {
		for _, a := range s.tmpl.Inputs.Artifacts {
			m["inputs.artifacts."+a.Name] = a // special case for artifacts
		}
	}
	return template.ResolveVar(v, m)
}

func (s *wfScope) resolveParameter(p *wfv1.ValueFrom) (any, error) {
	if p == nil || (p.Parameter == "" && p.Expression == "") {
		return "", nil
	}
	if p.Expression != "" {
		env := env.GetFuncMap(s.scope.AsAnyMap())
		program, err := expr.Compile(p.Expression, expr.Env(env))
		if err != nil {
			return nil, err
		}
		return expr.Run(program, env)
	}
	return s.resolveVar(p.Parameter)
}

func (s *wfScope) resolveArtifact(ctx context.Context, art *wfv1.Artifact) (*wfv1.Artifact, error) {
	if art == nil || (art.From == "" && art.FromExpression == "") {
		return nil, nil
	}

	var err error
	var val any

	if art.FromExpression != "" {
		envMap := env.GetFuncMap(s.scope.AsAnyMap())
		program, compileErr := expr.Compile(art.FromExpression, expr.Env(envMap))
		if compileErr != nil {
			return nil, compileErr
		}
		val, err = expr.Run(program, envMap)
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
		// If the workflow refers itself input artifacts in fromExpression, the val type is "*wfv1.Artifact"
		ptArt, ok := val.(*wfv1.Artifact)
		if !ok {
			return nil, errors.Errorf(errors.CodeBadRequest, "Variable {{%v}} is not an artifact", art)
		}
		valArt = *ptArt
	}

	if art.SubPath != "" {
		// Copy resolved artifact pointer before adding subpath
		copyArt := valArt.DeepCopy()

		subPathAsJSON, err := json.Marshal(art.SubPath)
		if err != nil {
			return copyArt, errors.New(errors.CodeBadRequest, "failed to marshal artifact subpath for templating")
		}

		resolvedSubPathAsJSON, err := template.Replace(ctx, string(subPathAsJSON), s.getParameters(), true)
		if err != nil {
			return nil, err
		}

		var resolvedSubPath string
		err = json.Unmarshal([]byte(resolvedSubPathAsJSON), &resolvedSubPath)
		if err != nil {
			return copyArt, errors.New(errors.CodeBadRequest, "failed to unmarshal artifact subpath for templating")
		}

		err = copyArt.AppendToKey(resolvedSubPath)
		if err != nil && copyArt.Optional { // Ignore error when artifact optional
			return copyArt, nil
		}
		return copyArt, err
	}

	return &valArt, nil
}
