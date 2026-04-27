package controller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/expr-lang/expr"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/expr/env"
	"github.com/argoproj/argo-workflows/v4/util/exprtrace"
	"github.com/argoproj/argo-workflows/v4/util/template"
	"github.com/argoproj/argo-workflows/v4/util/variables"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// wfScope contains the current scope of variables available when executing a template.
//
// MIGRATION: scope.scope (*exprtrace.Map) is the legacy string-keyed container;
// scope.typed (*variables.Scope) is the new catalog-gated view. addParamToScope
// and addArtifactToScope still write the legacy map for compatibility with code
// that has not yet been migrated; newer writers call addParam / addArtifact,
// which dual-write to both. When all writers go through Keys, the legacy map
// disappears.
type wfScope struct {
	tmpl  *wfv1.Template
	scope *exprtrace.Map
	typed *variables.Scope
}

func createScope(tmpl *wfv1.Template) *wfScope {
	scope := &wfScope{
		tmpl:  tmpl,
		scope: exprtrace.New(),
		typed: variables.NewScope(),
	}
	if tmpl != nil {
		for _, param := range scope.tmpl.Inputs.Parameters {
			val := scope.tmpl.Inputs.GetParameterByName(param.Name).Value.String()
			key := fmt.Sprintf("inputs.parameters.%s", param.Name)
			scope.scope.Set(key, val)
			varkeys.InputsParameterByName.Set(scope.typed, val, param.Name)
		}
		for _, param := range scope.tmpl.Inputs.Artifacts {
			art := scope.tmpl.Inputs.GetArtifactByName(param.Name)
			key := fmt.Sprintf("inputs.artifacts.%s", param.Name)
			scope.scope.Set(key, art)
			varkeys.InputsArtifactByName.Set(scope.typed, art, param.Name)
		}
	}
	return scope
}

// addParam is the catalog-gated write path. It writes via key to the typed
// scope AND mirrors into the legacy map so consumers that still read the old
// map remain correct during migration.
func (s *wfScope) addParam(key *variables.Key, value any, args ...string) {
	s.scope.SetFromCaller(key.Concretize(args...), value, 1)
	key.Set(s.typed, value, args...)
}

// getParameters returns a map of strings intended to be used simple string substitution
func (s *wfScope) getParameters() common.Parameters {
	params := make(common.Parameters)
	for key, entry := range s.scope.Entries() {
		valStr, ok := entry.Value.(string)
		if ok {
			params[key] = valStr
		}
	}
	return params
}

func (s *wfScope) addParamToScope(key, val string) {
	s.scope.SetFromCaller(key, val, 1)
}

func (s *wfScope) addArtifactToScope(key string, artifact wfv1.Artifact) {
	s.scope.SetFromCaller(key, artifact, 1)
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
		_, _ = s.scope.DumpD2(exprtrace.DumpTarget{
			Expression: p.Expression,
			Label:      "resolveParameter",
		})
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
		_, _ = s.scope.DumpD2(exprtrace.DumpTarget{
			Expression: art.FromExpression,
			Label:      "resolveArtifact",
		})
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
