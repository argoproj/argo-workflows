package controller

import (
	"context"
	"encoding/json"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v4/util/expr/env"
	"github.com/argoproj/argo-workflows/v4/util/template"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

func (woc *wfOperationCtx) runOnExitNode(ctx context.Context, exitHook *wfv1.LifecycleHook, parentNode *wfv1.NodeStatus, boundaryID string, tmplCtx *templateresolution.TemplateContext, ref varkeys.NodeRefKeys, name string, scope *wfScope) (bool, *wfv1.NodeStatus, error) {
	outputs := parentNode.Outputs
	if lastChildNode := woc.possiblyGetRetryChildNode(parentNode); lastChildNode != nil {
		outputs = lastChildNode.Outputs
	}

	if exitHook != nil && woc.GetShutdownStrategy().ShouldExecute(true) {
		execute := true
		var err error
		if exitHook.Expression != "" {
			// nil-preserving view so expressions can apply `??` fallbacks to skipped/omitted outputs
			execute, err = argoexpr.EvalBool(exitHook.Expression, env.GetFuncMap(scope.getParametersAny(woc.globalParams())))
			if err != nil {
				return true, nil, err
			}
		}
		if execute {
			woc.log.WithField("lifeCycleHook", exitHook).Info(ctx, "Running OnExit handler")
			onExitNodeName := common.GenerateOnExitNodeName(parentNode.Name)
			hookStep := &wfv1.WorkflowStep{Template: exitHook.Template, TemplateRef: exitHook.TemplateRef}
			resolvedArgs := exitHook.Arguments
			if !resolvedArgs.IsEmpty() {
				resolvedArgs, err = woc.resolveExitTmplArgument(ctx, exitHook.Arguments, ref, name, outputs, scope)
				if err != nil {
					return true, nil, err
				}
			}
			onExitNode, err := woc.executeTemplate(ctx, onExitNodeName, hookStep, tmplCtx, resolvedArgs, &executeTemplateOpts{
				boundaryID:     boundaryID,
				onExitTemplate: true,
				nodeFlag:       &wfv1.NodeFlag{Hooked: true},
			})
			woc.addChildNode(ctx, parentNode.Name, onExitNodeName)
			return true, onExitNode, err
		}
	}
	return false, nil, nil
}

func (woc *wfOperationCtx) resolveExitTmplArgument(ctx context.Context, args wfv1.Arguments, ref varkeys.NodeRefKeys, name string, outputs *wfv1.Outputs, scope *wfScope) (wfv1.Arguments, error) {
	if scope == nil {
		scope = createScope(nil)
	}
	if outputs != nil {
		for _, param := range outputs.Parameters {
			value := ""
			if param.Value != nil {
				value = param.Value.String()
			}
			ref.OutputsParameterByName.Set(scope.scope, value, name, param.Name)
		}
		for _, arts := range outputs.Artifacts {
			ref.OutputsArtifactByName.Set(scope.scope, arts, name, arts.Name)
		}
	}

	// Mirror task/step argument handling: a pure reference to a skipped/omitted output with no
	// producer default is replaced with a sentinel BEFORE substitution; common.ProcessArgs treats
	// it as unsupplied so the hook template's own input default applies (or fails terminally).
	scope.markAbsentOptionalArgs(&args)

	stepBytes, err := json.Marshal(args)
	if err != nil {
		return args, err
	}
	// nil-preserving view (and no strict prefixes, preserving the allow-unresolved behavior) so
	// expression tags can apply `??` fallbacks to skipped/omitted outputs, mirroring task/step args
	newStepStr, err := template.ReplaceStrictAny(ctx, string(stepBytes), scope.getParametersAny(woc.globalParams()), nil)
	if err != nil {
		return args, err
	}
	var newArgs wfv1.Arguments
	err = json.Unmarshal([]byte(newStepStr), &newArgs)
	if err != nil {
		return args, err
	}
	// Step 2: replace all artifact references
	for j, art := range newArgs.Artifacts {
		if art.From == "" && art.FromExpression == "" {
			continue
		}
		resolvedArt, err := scope.resolveArtifact(ctx, &art)
		if err != nil {
			if art.Optional {
				continue
			}
			return args, fmt.Errorf("unable to resolve references: %w", err)
		}
		resolvedArt.Name = art.Name
		newArgs.Artifacts[j] = *resolvedArt
	}
	return newArgs, nil
}
