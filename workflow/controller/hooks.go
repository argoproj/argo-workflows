package controller

import (
	"context"
	"encoding/json"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/template"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
)

func (woc *wfOperationCtx) executeWfLifeCycleHook(ctx context.Context, tmplCtx *templateresolution.Context) (*wfv1.NodeStatus, error) {
	if woc.wf.Spec.Hooks != nil && len(woc.wf.Spec.Hooks) > 0 {
		for hookName, hook := range woc.wf.Spec.Hooks {
			if hook.Expression == "" {
				return nil, nil
			}
			// Replace hook's parameters
			hookBytes, err := json.Marshal(hook)
			if err != nil {
				return nil, errors.InternalWrapError(err)
			}
			newHookStr, err := template.Replace(string(hookBytes), woc.globalParams, true)
			if err != nil {
				return nil, err
			}
			var newHook wfv1.LifecycleHook
			err = json.Unmarshal([]byte(newHookStr), &newHook)

			execute, err := shouldExecute(newHook.Expression)
			if err != nil {
				return nil, err
			}
			if execute {
				hookNodeName := common.GenerateLifeHookNodeName(woc.wf.ObjectMeta.Name, string(hookName))
				woc.log.WithField("lifeCycleHook", hookName).WithField("node", hookNodeName).Infof("Running workflow level hooks")
				_, err := woc.executeTemplate(ctx, hookNodeName, &wfv1.WorkflowStep{Template: hook.Template}, tmplCtx, hook.Arguments, &executeTemplateOpts{})
				if err != nil {
					return nil, err
				}
				woc.addChildNode(woc.wf.ObjectMeta.Name, hookNodeName)
			}
		}
	}
	return nil, nil
}

func (woc *wfOperationCtx) executeLifeCycleHook(ctx context.Context, scope *wfScope, lifeCycleHooks wfv1.LifecycleHooks, parentNode *wfv1.NodeStatus, boundaryID string, tmplCtx *templateresolution.Context, prefix string) (bool, *wfv1.NodeStatus, error) {

	if lifeCycleHooks == nil {
		return false, nil, nil
	}
	for hookName, hook := range lifeCycleHooks {
		if hookName == wfv1.ExitLifecycleEvent {
			return false, nil, nil
		}

		if hook.Expression == "" {
			return false, nil, nil
		}
		// Replace hook's parameters
		hookBytes, err := json.Marshal(hook)
		if err != nil {
			return false, nil, errors.InternalWrapError(err)
		}
		newHookStr, err := template.Replace(string(hookBytes), woc.globalParams.Merge(scope.getParameters()), true)
		if err != nil {
			return false, nil, err
		}
		var newHook wfv1.LifecycleHook
		err = json.Unmarshal([]byte(newHookStr), &newHook)

		execute, err := shouldExecute(newHook.Expression)
		if err != nil {
			return false, nil, err
		}
		if execute {
			outputs := parentNode.Outputs
			if parentNode.Type == wfv1.NodeTypeRetry {
				lastChildNode := getChildNodeIndex(parentNode, woc.wf.Status.Nodes, -1)
				outputs = lastChildNode.Outputs
			}
			hookNodeName := common.GenerateLifeHookNodeName(parentNode.Name, string(hookName))
			woc.log.WithField("lifeCycleHook", hookName).WithField("node", hookNodeName).WithField("hookName", hookName).Info("Running hooks")
			resolvedArgs := newHook.Arguments
			var err error
			if !resolvedArgs.IsEmpty() && outputs != nil {
				resolvedArgs, err = woc.resolveExitTmplArgument(newHook.Arguments, prefix, outputs)
				if err != nil {
					return true, nil, err
				}
			}
			onExitNode, err := woc.executeTemplate(ctx, hookNodeName, &wfv1.WorkflowStep{Template: newHook.Template}, tmplCtx, resolvedArgs, &executeTemplateOpts{
				boundaryID: boundaryID,
			})
			woc.addChildNode(parentNode.Name, hookNodeName)
			return true, onExitNode, err
		}
	}
	return false, nil, nil
}