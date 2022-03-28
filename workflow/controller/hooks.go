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

func (woc *wfOperationCtx) executeWfLifeCycleHook(ctx context.Context, tmplCtx *templateresolution.Context) error {
	for hookName, hook := range woc.execWf.Spec.Hooks {
		//exit hook will be execute in runOnExitNode
		if hookName == wfv1.ExitLifecycleEvent {
			continue
		}
		execute, err := argoexpr.EvalBool(hook.Expression, env.GetFuncMap(template.EnvMap(woc.globalParams)))
		if err != nil {
			return err
		}
		if execute {
			hookNodeName := generateLifeHookNodeName(woc.wf.ObjectMeta.Name, string(hookName))
			woc.log.WithField("lifeCycleHook", hookName).WithField("node", hookNodeName).Infof("Running workflow level hooks")
			_, err := woc.executeTemplate(ctx, hookNodeName, &wfv1.WorkflowStep{Template: hook.Template, TemplateRef: hook.TemplateRef}, tmplCtx, hook.Arguments, &executeTemplateOpts{})
			if err != nil {
				return err
			}
			woc.addChildNode(woc.wf.Name, hookNodeName)
		}
	}

	return nil
}

func (woc *wfOperationCtx) executeTmplLifeCycleHook(ctx context.Context, envMap map[string]string, lifeCycleHooks wfv1.LifecycleHooks, parentNode *wfv1.NodeStatus, boundaryID string, tmplCtx *templateresolution.Context, prefix string) (bool, error) {
	var hookNodes []*wfv1.NodeStatus
	completed := false
	for hookName, hook := range lifeCycleHooks {
		//exit hook will be execute in runOnExitNode
		if hookName == wfv1.ExitLifecycleEvent {
			continue
		}
		execute, err := argoexpr.EvalBool(hook.Expression, env.GetFuncMap(template.EnvMap(envMap)))
		if err != nil {
			return completed, err
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
					return completed, err
				}
			}
			hookNode, err := woc.executeTemplate(ctx, hookNodeName, &wfv1.WorkflowStep{Template: hook.Template, TemplateRef: hook.TemplateRef}, tmplCtx, resolvedArgs, &executeTemplateOpts{
				boundaryID: boundaryID,
			})
			if err != nil {
				return completed, err
			}
			woc.addChildNode(parentNode.Name, hookNodeName)
			hookNodes = append(hookNodes, hookNode)

		}
	}
	completed = true

	// All hook nodes are completed
	for _, hookNode := range hookNodes {
		if !hookNode.Fulfilled() {
			completed = false
			break
		}
	}
	return completed, nil
}

func generateLifeHookNodeName(parentNodeName string, hookName string) string {
	return fmt.Sprintf("%s.hooks.%s", parentNodeName, hookName)
}
