package controller

import (
	"context"
	"fmt"
	"strings"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v3/util/expr/env"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

func IsExitHook(hookName wfv1.LifecycleEvent) bool {
	return hookName == wfv1.ExitLifecycleEvent
}

func (woc *wfOperationCtx) executeWfLifeCycleHook(ctx context.Context, tmplCtx *templateresolution.Context) error {
	for hookName, hook := range woc.wf.Spec.Hooks {
		//exit hook will be execute in runOnExitNode
		if IsExitHook(hookName) {
			continue
		}
		execute, err := shouldExecuteHook(hook.Expression, getEnv(woc.globalParams))
		if err != nil {
			return err
		}
		if execute {
			hookNodeName := generateLifeHookNodeName(woc.wf.ObjectMeta.Name, string(hookName))
			woc.log.WithField("lifeCycleHook", hookName).WithField("node", hookNodeName).Infof("Running workflow level hooks")
			_, err := woc.executeTemplate(ctx, hookNodeName, &wfv1.WorkflowStep{Template: hook.Template}, tmplCtx, hook.Arguments, &executeTemplateOpts{})
			if err != nil {
				return err
			}
			woc.addChildNode(woc.wf.Name, hookNodeName)
		}
	}

	return nil
}

func (woc *wfOperationCtx) executeTmplLifeCycleHook(ctx context.Context, scope *wfScope, lifeCycleHooks wfv1.LifecycleHooks, parentNode *wfv1.NodeStatus, boundaryID string, tmplCtx *templateresolution.Context, prefix string) error {

	for hookName, hook := range lifeCycleHooks {
		//exit hook will be execute in runOnExitNode
		if IsExitHook(hookName) {
			continue
		}
		execute, err := shouldExecuteHook(hook.Expression, getEnv(woc.globalParams.Merge(scope.getParameters())))
		if err != nil {
			return err
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
				resolvedArgs, err = woc.resolveExitTmplArgument(hook.Arguments, prefix, outputs)
				if err != nil {
					return err
				}
			}
			_, err = woc.executeTemplate(ctx, hookNodeName, &wfv1.WorkflowStep{Template: hook.Template}, tmplCtx, resolvedArgs, &executeTemplateOpts{
				boundaryID: boundaryID,
			})
			woc.addChildNode(parentNode.Name, hookNodeName)
			return err
		}
	}
	return nil
}

func getEnv(parameters common.Parameters) map[string]interface{} {
	params := make(map[string]interface{})
	for key, value := range parameters {
		params[strings.Replace(key, "-", "_", -1)] = value
	}
	return env.GetFuncMap(params)
}

func shouldExecuteHook(expression string, env map[string]interface{}) (bool, error) {
	return argoexpr.EvalBool(strings.Replace(expression, "-", "_", -1), env)
}

func generateLifeHookNodeName(parentNodeName string, hookName string) string {
	return fmt.Sprintf("%s.hooks.%s", parentNodeName, hookName)
}
