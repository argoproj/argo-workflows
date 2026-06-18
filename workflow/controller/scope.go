package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/expr-lang/expr"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/expr/env"
	"github.com/argoproj/argo-workflows/v4/util/template"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
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

// getParametersAny returns the scope's parameters merged over the given globals, preserving nil
// (absent optional) values so expression tags can distinguish absent from empty (e.g. via `??`).
// A simple tag resolving to a nil value is a terminal substitution error; arguments rescued by a
// consumer input default are replaced with a sentinel before substitution (see markAbsentOptionalArgs).
func (s *wfScope) getParametersAny(globals common.Parameters) map[string]any {
	params := make(map[string]any, len(globals)+len(s.scope))
	for key, val := range globals {
		params[key] = val
	}
	for key, val := range s.scope {
		switch val.(type) {
		case string, nil:
			params[key] = val
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

// addSkippedParamToScope registers a parameter key for a skipped/omitted step's output that declared
// no default. The value is stored as nil (an absent optional) so structured consumers and expressions
// can tell it apart from a legitimately empty output (e.g. `ref ?? "fallback"`). A simple tag
// resolving to the nil is a terminal substitution error unless the referencing argument was replaced
// with a sentinel for ProcessArgs to interpret (markAbsentOptionalArgs); this applies uniformly to every
// substitution surface (arguments, when clauses, volumes, artifact subPath, item expansion). nil is
// stored ONLY by this method, so a present-but-nil scope value is the definition of "skipped with
// no default".
func (s *wfScope) addSkippedParamToScope(key string) {
	s.scope[key] = nil
}

// absentOptionalRef reports whether an argument value is a single pure reference (e.g.
// "{{tasks.x.outputs.parameters.y}}") to a key holding an absent optional, i.e. a skipped/omitted
// node output that declared no producer default. A brace-less literal that merely spells out a
// scope key is data, not a reference, and composite values like "x-{{key}}-y" or nested tags are
// not pure references. nil is only ever written by addSkippedParamToScope, so a present-but-nil
// value identifies the skipped placeholder.
func (s *wfScope) absentOptionalRef(v string) bool {
	if !strings.HasPrefix(v, "{{") || !strings.HasSuffix(v, "}}") {
		return false
	}
	key := strings.TrimSpace(v[2 : len(v)-2])
	if strings.Contains(key, "{{") || strings.Contains(key, "}}") {
		return false
	}
	val, ok := s.scope[key]
	return ok && val == nil
}

// markAbsentOptionalArgs replaces arguments that are pure references to a skipped/omitted node's
// output with no producer default (an absent optional, nil in scope) with the
// common.AbsentOptionalArgumentValue sentinel, BEFORE substitution. The sentinel survives textual
// substitution — where the raw absent value would be a terminal error — and is interpreted by
// common.ProcessArgs at consumption time, when the consumed template is fully resolved (even for
// dynamic "{{item.*}}" templateRefs): the argument is treated as unsupplied so the input's own
// default applies, and an input with no default fails terminally. Composite values like
// "x-{{ref}}-y" are not pure references and still fail substitution. Builds a fresh Parameters
// slice: args may alias a step/task still shared with the caller.
func (s *wfScope) markAbsentOptionalArgs(args *wfv1.Arguments) {
	var marked []wfv1.Parameter
	for i, p := range args.Parameters {
		if p.Value != nil && s.absentOptionalRef(p.Value.String()) {
			if marked == nil {
				marked = make([]wfv1.Parameter, len(args.Parameters))
				copy(marked, args.Parameters)
			}
			marked[i].Value = wfv1.AnyStringPtr(common.AbsentOptionalArgumentValue)
		}
	}
	if marked != nil {
		args.Parameters = marked
	}
}

// addSkippedNodeOutputsToScope populates scope with the declared output parameters of a skipped or
// omitted node that produced no outputs, so downstream references — task/step inputs, when-clauses, and
// DAG/steps output aggregation (including ValueFrom.Expression) — resolve to the producer's declared
// default or to an absent (nil) optional instead of requeuing forever; an unhandled absent optional
// fails terminally rather than leaving the workflow stuck. No-op for any node
// that actually produced outputs. includeArtifacts additionally registers empty placeholders for the
// template's declared output artifacts: steps relies on this to keep artifact references resolvable,
// while DAG deliberately leaves them unresolved (resolveDependencyReferences omits optional artifacts
// and errors on required ones).
func (woc *wfOperationCtx) addSkippedNodeOutputsToScope(ctx context.Context, tmplCtx *templateresolution.TemplateContext, scope *wfScope, prefix string, node *wfv1.NodeStatus, tmplHolder wfv1.TemplateReferenceHolder, includeArtifacts bool) {
	if node == nil || node.Outputs != nil {
		return
	}
	if node.Phase != wfv1.NodeSkipped && node.Phase != wfv1.NodeOmitted {
		return
	}
	_, tmpl, _, err := tmplCtx.ResolveTemplate(ctx, tmplHolder)
	if err != nil {
		woc.log.WithError(err).Debug(ctx, "failed to resolve template for skipped node, outputs will not be populated in scope")
		return
	}
	if tmpl == nil {
		return
	}
	for _, param := range tmpl.Outputs.Parameters {
		key := fmt.Sprintf("%s.outputs.parameters.%s", prefix, param.Name)
		scope.addSkippedOutputParamToScope(key, param.ValueFrom)
	}
	if includeArtifacts {
		for _, artifact := range tmpl.Outputs.Artifacts {
			key := fmt.Sprintf("%s.outputs.artifacts.%s", prefix, artifact.Name)
			scope.addArtifactToScope(key, wfv1.Artifact{})
		}
	}
	if tmpl.Outputs.Result != nil {
		scope.addSkippedParamToScope(fmt.Sprintf("%s.outputs.result", prefix))
	}
}

// addSkippedOutputParamToScope registers an output parameter of a skipped/omitted node. If the
// producing template declared a valueFrom.default for the parameter, that default is used as the
// scope value so every consumer (task inputs and template-output aggregation alike) sees it.
// Otherwise it falls back to addSkippedParamToScope's nil (absent optional) behavior.
func (s *wfScope) addSkippedOutputParamToScope(key string, valueFrom *wfv1.ValueFrom) {
	if valueFrom != nil && valueFrom.Default != nil {
		s.addParamToScope(key, valueFrom.Default.String())
		return
	}
	s.addSkippedParamToScope(key)
}

// resolveVar resolves a parameter or artifact
func (s *wfScope) resolveVar(v string) (interface{}, error) {
	m := make(map[string]interface{})
	maps.Copy(m, s.scope)
	if s.tmpl != nil {
		for _, a := range s.tmpl.Inputs.Artifacts {
			m["inputs.artifacts."+a.Name] = a // special case for artifacts
		}
	}
	return template.ResolveVar(v, m)
}

func (s *wfScope) resolveParameter(p *wfv1.ValueFrom) (any, bool, error) {
	if p == nil || (p.Parameter == "" && p.Expression == "") {
		return "", false, nil
	}
	if p.Expression != "" {
		env := env.GetFuncMap(s.scope)
		program, err := expr.Compile(p.Expression, expr.Env(env))
		if err != nil {
			return nil, false, err
		}
		val, err := expr.Run(program, env)
		if err != nil {
			return nil, false, err
		}
		if val == nil {
			// A nil result is an unhandled absent optional (e.g. a skipped node's defaultless output
			// referenced without `??`). Mirror the inline {{= ...}} semantics and treat it as a
			// resolution failure; the caller's error path applies valueFrom.default when declared.
			return nil, false, errors.Errorf(errors.CodeBadRequest, "failed to evaluate expression %q", p.Expression)
		}
		return val, false, nil
	}
	val, err := s.resolveVar(p.Parameter)
	// nil is only ever stored by addSkippedParamToScope, so it identifies a skipped placeholder.
	return val, err == nil && val == nil, err
}

func (s *wfScope) resolveArtifact(ctx context.Context, art *wfv1.Artifact) (*wfv1.Artifact, error) {
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
		// If the workflow refers itself input artifacts in fromExpression, the val type is "*wfv1.Artifact"
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

		subPathAsJSON, err := json.Marshal(art.SubPath)
		if err != nil {
			return copyArt, errors.New(errors.CodeBadRequest, "failed to marshal artifact subpath for templating")
		}

		resolvedSubPathAsJSON, err := template.Replace(ctx, string(subPathAsJSON), s.getParametersAny(nil), true)
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
