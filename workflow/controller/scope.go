package controller

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/expr-lang/expr"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/expr/env"
	"github.com/argoproj/argo-workflows/v4/util/template"
	"github.com/argoproj/argo-workflows/v4/util/variables"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
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
			p := scope.tmpl.Inputs.GetParameterByName(param.Name)
			if p.Value != nil {
				varkeys.InputsParameterByName.Set(scope.scope, p.Value.String(), param.Name)
			} else if p.Default != nil {
				varkeys.InputsParameterByName.Set(scope.scope, p.Default.String(), param.Name)
			}
		}
		for _, param := range scope.tmpl.Inputs.Artifacts {
			art := scope.tmpl.Inputs.GetArtifactByName(param.Name)
			varkeys.InputsArtifactByName.Set(scope.scope, art, param.Name)
		}
	}
	return scope
}

// getParametersAny returns the scope's parameters merged over the given globals, preserving nil
// (absent optional) values so expression tags can distinguish absent from empty (e.g. via `??`).
// A simple tag resolving to a nil value is a terminal substitution error; arguments rescued by a
// consumer input default are replaced with a sentinel before substitution (see markAbsentOptionalArgs).
func (s *wfScope) getParametersAny(globals common.Parameters) map[string]any {
	scopeParams := s.scope.AsAnyMap()
	params := make(map[string]any, len(globals)+len(scopeParams))
	for key, val := range globals {
		params[key] = val
	}
	for key, val := range scopeParams {
		switch val.(type) {
		case string, nil:
			params[key] = val
		}
	}
	return params
}

// getParameters returns the scope's string-valued parameters as a flat string map, DROPPING absent
// optionals (nil values for skipped/omitted outputs with no default) and artifacts. Used by the
// substitution surfaces that operate on a plain string map and cannot represent absence — item /
// withParam / withSequence expansion (via the dag.Substitutor) and retry-node local params. Callers
// that must distinguish absent from empty (arguments, when-clause `??` fallbacks) use getParametersAny.
func (s *wfScope) getParameters() common.Parameters {
	return common.Parameters(s.scope.AsStringMap())
}

// absentOptionalRef reports whether an argument value is a single pure reference (e.g.
// "{{tasks.x.outputs.parameters.y}}") to a key holding an absent optional, i.e. a skipped/omitted
// node output that declared no producer default. A brace-less literal that merely spells out a
// scope key is data, not a reference, and composite values like "x-{{key}}-y" or nested tags are
// not pure references. An absent optional is exactly a key written via Key.SetSkipped with a nil
// value (see addSkippedNodeOutputsToScope), so IsSkipped identifies the placeholder.
func (s *wfScope) absentOptionalRef(v string) bool {
	if !strings.HasPrefix(v, "{{") || !strings.HasSuffix(v, "}}") {
		return false
	}
	key := strings.TrimSpace(v[2 : len(v)-2])
	if strings.Contains(key, "{{") || strings.Contains(key, "}}") {
		return false
	}
	return s.scope.IsSkipped(key)
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
func (woc *wfOperationCtx) addSkippedNodeOutputsToScope(ctx context.Context, tmplCtx *templateresolution.TemplateContext, scope *wfScope, ref varkeys.NodeRefKeys, name string, node *wfv1.NodeStatus, tmplHolder wfv1.TemplateReferenceHolder, includeArtifacts bool) {
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
		// A declared producer default is a real value every consumer (task/step inputs and
		// template-output aggregation alike) should see. Without one, the output is an absent
		// optional: stored as nil and marked skipped so consumers can tell it apart from a
		// legitimately empty output (e.g. `ref ?? "fallback"`).
		if param.ValueFrom != nil && param.ValueFrom.Default != nil {
			ref.OutputsParameterByName.Set(scope.scope, param.ValueFrom.Default.String(), name, param.Name)
		} else {
			ref.OutputsParameterByName.SetSkipped(scope.scope, nil, name, param.Name)
		}
	}
	if includeArtifacts {
		for _, artifact := range tmpl.Outputs.Artifacts {
			ref.OutputsArtifactByName.Set(scope.scope, wfv1.Artifact{}, name, artifact.Name)
		}
	}
	if tmpl.Outputs.Result != nil {
		ref.OutputsResult.SetSkipped(scope.scope, nil, name)
	}
}

// resolveVar resolves a parameter or artifact and additionally returns the unwrapped tag
// (v with the surrounding "{{" / "}}" stripped) so callers can reuse it without re-parsing.
func (s *wfScope) resolveVar(v string) (string, any, error) {
	m := s.scope.AsAnyMap()
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
		env := env.GetFuncMap(s.scope.AsAnyMap())
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
	tag, val, err := s.resolveVar(p.Parameter)
	// IsSkipped is true only for a placeholder written via Key.SetSkipped (a skipped/omitted node
	// output with no producer default), i.e. an absent optional.
	return val, s.scope.IsSkipped(tag), err
}

// resolveArguments resolves argument parameter and artifact references against the scope.
// Parameter values containing {{steps.X.outputs.parameters.Y}} (or tasks.*) references
// are substituted. Artifact arguments with From/FromExpression are resolved to concrete
// storage locations. This should be called before ProcessArgs so that scope-level
// references don't leak into the child template body via SubstituteParams.
func (s *wfScope) resolveArguments(ctx context.Context, args wfv1.Arguments, globalParams common.Parameters) (wfv1.Arguments, error) {
	// nil-preserving view so expression tags can apply `??` fallbacks to skipped/omitted outputs
	mergedParams := s.getParametersAny(globalParams)

	// Replace arguments that are pure references to a skipped/omitted node's output with no producer
	// default with a sentinel BEFORE substitution; common.ProcessArgs interprets it as "unsupplied"
	// at consumption time so the consumed template's input default applies (or fails terminally).
	s.markAbsentOptionalArgs(&args)

	// Resolve parameter value references by JSON-marshaling the arguments,
	// performing template replacement, then unmarshaling. This matches main's
	// resolveDependencyReferences behavior: simpleReplace escapes values for
	// JSON context, and the unmarshal step reverses the escaping. Doing direct
	// string replacement would double-escape values containing quotes.
	argsBytes, err := json.Marshal(args.Parameters)
	if err != nil {
		return args, err
	}
	argsStr := string(argsBytes)
	if strings.Contains(argsStr, "{{") {
		resolved, err := template.Replace(ctx, argsStr, mergedParams, true)
		if err != nil {
			return args, err
		}
		var resolvedParams []wfv1.Parameter
		if err := json.Unmarshal([]byte(resolved), &resolvedParams); err != nil {
			return args, err
		}
		args.Parameters = resolvedParams
	}

	// Resolve artifact from/fromExpression references. Build a fresh slice so
	// the caller's backing array isn't mutated through the slice header.
	if len(args.Artifacts) > 0 {
		resolvedArtifacts := make(wfv1.Artifacts, 0, len(args.Artifacts))
		for i := range args.Artifacts {
			art := args.Artifacts[i]
			if art.From == "" && art.FromExpression == "" {
				resolvedArtifacts = append(resolvedArtifacts, art)
				continue
			}
			resolvedArt, err := s.resolveArtifact(ctx, &art)
			if err != nil {
				if art.Optional {
					// Optional artifact that failed to resolve: drop it from
					// arguments (matches legacy resolveDependencyReferences).
					continue
				}
				return args, err
			}
			if resolvedArt == nil {
				continue
			}
			resolvedArt.Name = art.Name
			resolvedArtifacts = append(resolvedArtifacts, *resolvedArt)
		}
		args.Artifacts = resolvedArtifacts
	}

	return args, nil
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
