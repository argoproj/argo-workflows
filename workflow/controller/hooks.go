package controller

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v3/util/expr/env"
	"github.com/argoproj/argo-workflows/v3/util/template"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

func (woc *wfOperationCtx) executeWfLifeCycleHook(ctx context.Context, tmplCtx *templateresolution.Context) (bool, error) {
	var hookNodes []*wfv1.NodeStatus
	for hookName, hook := range woc.execWf.Spec.Hooks {
		//exit hook will be executed in runOnExitNode
		if hookName == wfv1.ExitLifecycleEvent {
			continue
		}
		execute, err := argoexpr.EvalBool(hook.Expression, env.GetFuncMap(template.EnvMap(woc.globalParams)))
		if err != nil {
			return true, err
		}
		if execute {
			hookNodeName := generateLifeHookNodeName(woc.wf.ObjectMeta.Name, string(hookName))
			woc.log.WithField("lifeCycleHook", hookName).WithField("node", hookNodeName).Infof("Running workflow level hooks")
			hookNode, err := woc.executeTemplate(ctx, hookNodeName, &wfv1.WorkflowStep{Template: hook.Template, TemplateRef: hook.TemplateRef}, tmplCtx, hook.Arguments, &executeTemplateOpts{})
			if err != nil {
				return true, err
			}
			woc.addChildNode(woc.wf.Name, hookNodeName)
			hookNodes = append(hookNodes, hookNode)
			// If the hookNode node is HTTP template, it requires HTTP reconciliation, do it here
			if hookNode != nil && woc.nodeRequiresTaskSetReconciliation(hookNode.Name) {
				woc.taskSetReconciliation(ctx)
			}
		}
	}
	for _, hookNode := range hookNodes {
		if !hookNode.Fulfilled() {
			return false, nil
		}
	}

	return true, nil
}

func (woc *wfOperationCtx) executeTmplLifeCycleHook(ctx context.Context, scope *wfScope, lifeCycleHooks wfv1.LifecycleHooks, parentNode *wfv1.NodeStatus, boundaryID string, tmplCtx *templateresolution.Context, prefix string) (bool, error) {
	var hookNodes []*wfv1.NodeStatus
	for hookName, hook := range lifeCycleHooks {
		//exit hook will be executed in runOnExitNode
		if hookName == wfv1.ExitLifecycleEvent {
			continue
		}
		execute, err := argoexpr.EvalBool(hook.Expression, env.GetFuncMap(template.EnvMap(woc.globalParams.Merge(scope.getParameters()))))
		if err != nil {
			return false, err
		}
		if execute {
			outputs := parentNode.Outputs
			if parentNode.Type == wfv1.NodeTypeRetry {
				lastChildNode := getChildNodeIndex(parentNode, woc.wf.Status.Nodes, -1)
				outputs = lastChildNode.Outputs
			}
			hookNodeName := generateLifeHookNodeName(parentNode.Name, string(hookName))
			woc.log.WithField("lifeCycleHook", hookName).WithField("node", hookNodeName).WithField("hookName", hookName).Info("Running hooks")
			resolvedArgs := hook.Arguments
			var err error
			if !resolvedArgs.IsEmpty() && outputs != nil {
				resolvedArgs, err = woc.resolveExitTmplArgument(hook.Arguments, prefix, outputs, scope)
				if err != nil {
					return false, err
				}
			}
			hookNode, err := woc.executeTemplate(ctx, hookNodeName, &wfv1.WorkflowStep{Template: hook.Template, TemplateRef: hook.TemplateRef}, tmplCtx, resolvedArgs, &executeTemplateOpts{
				boundaryID: boundaryID,
			})
			if err != nil {
				return false, err
			}
			woc.addChildNode(parentNode.Name, hookNodeName)
			hookNodes = append(hookNodes, hookNode)
		}
	}

	// Check if all hook nodes are completed
	for _, hookNode := range hookNodes {
		if !hookNode.Fulfilled() {
			return false, nil
		}
	}
	return true, nil
}

func generateLifeHookNodeName(parentNodeName string, hookName string) string {
	return fmt.Sprintf("%s.hooks.%s", parentNodeName, hookName)
}
