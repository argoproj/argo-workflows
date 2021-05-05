package controller

import (
	"context"
	"encoding/json"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/template"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

func (woc *wfOperationCtx) runOnExitNode(ctx context.Context, exitHook *wfv1.LifecycleHook, parentDisplayName, parentNodeName, boundaryID string, tmplCtx *templateresolution.Context, prefix string, outputs *wfv1.Outputs) (bool, *wfv1.NodeStatus, error) {
	if exitHook != nil && woc.GetShutdownStrategy().ShouldExecute(true) {
		woc.log.WithField("lifeCycleHook", exitHook).Infof("Running OnExit handler")

		// Previously we used `parentDisplayName` to generate all onExit node names. However, as these can be non-unique
		// we transitioned to using `parentNodeName` instead, which are guaranteed to be unique. In order to not disrupt
		// running workflows during upgrade time, we first check if there is an onExit node that currently exists with the
		// legacy name AND said node is a child of the parent node. If it does, we continue execution with the legacy name.
		// If it doesn't, we use the new (and unique) name for all operations henceforth.
		// TODO: This scaffold code should be removed after a couple of "grace period" version upgrades to allow transitions. It was introduced in v3.0.0
		// When the scaffold code is removed, we should only have the following:
		//
		// 		onExitNodeName := common.GenerateOnExitNodeName(parentNodeName)
		//
		// See more: https://github.com/argoproj/argo-workflows/issues/5502
		onExitNodeName := common.GenerateOnExitNodeName(parentNodeName)
		legacyOnExitNodeName := common.GenerateOnExitNodeName(parentDisplayName)
		if legacyNameNode := woc.wf.GetNodeByName(legacyOnExitNodeName); legacyNameNode != nil && woc.wf.GetNodeByName(parentNodeName).HasChild(legacyNameNode.ID) {
			onExitNodeName = legacyOnExitNodeName
		}
		resolvedArgs := exitHook.Arguments
		var err error
		if !resolvedArgs.IsEmpty() && outputs != nil {
			resolvedArgs, err = woc.resolveExitTmplArgument(exitHook.Arguments, prefix, outputs)
			if err != nil {
				return true, nil, err
			}

		}
		onExitNode, err := woc.executeTemplate(ctx, onExitNodeName, &wfv1.WorkflowStep{Template: exitHook.Template}, tmplCtx, resolvedArgs, &executeTemplateOpts{

			boundaryID:     boundaryID,
			onExitTemplate: true,
		})
		woc.addChildNode(parentNodeName, onExitNodeName)
		return true, onExitNode, err
	}
	return false, nil, nil
}

func (woc *wfOperationCtx) resolveExitTmplArgument(args wfv1.Arguments, prefix string, outputs *wfv1.Outputs) (wfv1.Arguments, error) {
	scope := createScope(nil)
	for _, param := range outputs.Parameters {
		scope.addParamToScope(fmt.Sprintf("%s.outputs.parameters.%s", prefix, param.Name), param.Value.String())
	}
	for _, arts := range outputs.Artifacts {
		scope.addArtifactToScope(fmt.Sprintf("%s.outputs.artifacts.%s", prefix, arts.Name), arts)
	}

	stepBytes, err := json.Marshal(args)
	if err != nil {
		return args, err
	}
	newStepStr, err := template.Replace(string(stepBytes), woc.globalParams.Merge(scope.getParameters()), true)
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
		resolvedArt, err := scope.resolveArtifact(&art)
		if err != nil {
			if art.Optional {
				continue
			}
			return args, fmt.Errorf("unable to resolve references: %s", err)
		}
		resolvedArt.Name = art.Name
		newArgs.Artifacts[j] = *resolvedArt
	}
	return newArgs, nil
}
