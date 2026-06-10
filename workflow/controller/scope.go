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
	scope map[string]any
}

func createScope(tmpl *wfv1.Template) *wfScope {
	scope := &wfScope{
		tmpl:  tmpl,
		scope: make(map[string]any),
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
		switch v := val.(type) {
		case string:
			params[key] = v
		case nil:
			// Skipped/omitted optional output: nil for expression consumers, but "" here so plain
			// string substitution can still resolve the reference and keep the node live.
			params[key] = ""
		}
	}
	return params
}

// getParametersAny returns the scope's parameters merged over the given globals, preserving nil
// (absent optional) values so expression tags can distinguish absent from empty (e.g. via `??`).
// A simple tag resolving to a nil value is a terminal substitution error; arguments rescued by a
// consumer input default must be dropped before substitution (see dropSkippedDefaultedArgs).
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
// resolving to the nil is a terminal substitution error unless the referencing argument was dropped
// in favor of a consumer input default (dropSkippedDefaultedArgs); getParameters still flattens it
// to "" for the legacy string-map paths (volumes, item expansion). nil is stored ONLY by this
// method, so a present-but-nil scope value is the definition of "skipped with no default".
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

// dropSkippedDefaultedArgs removes arguments that are pure references to a skipped/omitted node's
// output with no producer default (an absent optional, nil in scope) WHEN the consumed template
// declares its own input default for that parameter. It must run BEFORE substitution: dropping the
// argument makes it indistinguishable from an unsupplied one, so the consumer's input default
// applies through the normal path, while any absent-optional reference left in place fails
// substitution with a terminal error.
func (woc *wfOperationCtx) dropSkippedDefaultedArgs(ctx context.Context, tmplCtx *templateresolution.TemplateContext, holder wfv1.TemplateReferenceHolder, args *wfv1.Arguments, scope *wfScope) error {
	candidates := make(map[string]bool)
	for _, p := range args.Parameters {
		if p.Value != nil && scope.absentOptionalRef(p.Value.String()) {
			candidates[p.Name] = true
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	// Resolve the consumed template once to learn which of those inputs have their own default. The
	// template may not be resolvable at this point (e.g. a "{{item.*}}" templateRef that only resolves
	// at expansion); this handling is best-effort sugar, so skip it rather than failing here — the
	// surviving reference will fail substitution with a terminal absent-optional error.
	_, tmpl, _, err := tmplCtx.ResolveTemplate(ctx, holder)
	if err != nil || tmpl == nil {
		woc.log.WithError(err).WithField("template", holder.GetTemplateName()).
			Debug(ctx, "could not resolve consumed template; skipping skipped-arg default handling")
		return nil
	}
	if err := woc.mergedTemplateDefaultsInto(tmpl); err != nil {
		return err
	}
	// Build a fresh slice: args may alias a step/task still shared with the caller.
	kept := make([]wfv1.Parameter, 0, len(args.Parameters))
	for _, p := range args.Parameters {
		if candidates[p.Name] {
			if in := tmpl.Inputs.GetParameterByName(p.Name); in != nil && in.Default != nil {
				continue // drop: the argument becomes unsupplied, the consumer's own input default applies
			}
		}
		kept = append(kept, p)
	}
	args.Parameters = kept
	return nil
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

// resolveVar resolves a parameter or artifact and additionally returns the unwrapped tag
// (v with the surrounding "{{" / "}}" stripped) so callers can reuse it without re-parsing.
func (s *wfScope) resolveVar(v string) (string, any, error) {
	m := make(map[string]any)
	maps.Copy(m, s.scope)
	if s.tmpl != nil {
		for _, a := range s.tmpl.Inputs.Artifacts {
			m["inputs.artifacts."+a.Name] = a // special case for artifacts
		}
	}
	return template.ResolveVar(v, m)
}

// resolveParameter resolves a ValueFrom and additionally reports whether the source was a
// skipped/omitted step's placeholder output (only meaningful for the Parameter form).
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
	_, val, err := s.resolveVar(p.Parameter)
	// nil is only ever stored by addSkippedParamToScope, so it identifies a skipped placeholder.
	return val, err == nil && val == nil, err
}

func (s *wfScope) resolveArtifact(ctx context.Context, art *wfv1.Artifact) (*wfv1.Artifact, error) {
	if art == nil || (art.From == "" && art.FromExpression == "") {
		return nil, nil
	}

	var err error
	var val any

	if art.FromExpression != "" {
		envMap := env.GetFuncMap(s.scope)
		program, compileErr := expr.Compile(art.FromExpression, expr.Env(envMap))
		if compileErr != nil {
			return nil, compileErr
		}
		val, err = expr.Run(program, envMap)
		if err != nil {
			return nil, err
		}
	} else {
		_, val, err = s.resolveVar(art.From)
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
