package controller

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v4/errors"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v4/util/expr/env"
	"github.com/argoproj/argo-workflows/v4/util/template"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

func (woc *wfOperationCtx) executeWfLifeCycleHook(ctx context.Context, tmplCtx *templateresolution.TemplateContext) (bool, error) {
	var hookNodes []*wfv1.NodeStatus
	entryNode, _ := woc.wf.GetNodeByName(woc.wf.Name)
	for hookName, hook := range woc.execWf.Spec.Hooks {
		// exit hook will be executed in runOnExitNode
		if hookName == wfv1.ExitLifecycleEvent {
			continue
		}
		hookNodeName := generateLifeHookNodeName(woc.wf.Name, string(hookName))
		// To check a node was triggered.
		hookedNode, _ := woc.wf.GetNodeByName(hookNodeName)
		if hook.Expression == "" {
			// No node exists yet for this hook (reconcileTemplate never ran),
			// so record the error on its own hook node, as errorHookNode
			// does for a template-level hook (C92): the entry node may
			// already be Succeeded, and the strict node phase state machine
			// refuses to flip it to Error.
			return true, woc.errorHookNode(ctx, hookNodeName, &hook, entryNode, "", tmplCtx, errors.Errorf(errors.CodeBadRequest, "Expression required for hook %s", hookNodeName))
		}
		execute, err := argoexpr.EvalBool(hook.Expression, env.GetFuncMap(template.EnvMap(woc.globalParams())))
		if err != nil {
			return true, woc.errorHookNode(ctx, hookNodeName, &hook, entryNode, "", tmplCtx, err)
		}
		// executeTemplated should be invoked when hookedNode != nil, because we should reexecute the function to check mutex condition, etc.
		if execute || hookedNode != nil {
			woc.log.WithField("lifeCycleHook", hookName).WithField("node", hookNodeName).Info(ctx, "Running workflow level hooks")
			hookNode, err := woc.reconcileTemplate(ctx, hookNodeName, &wfv1.WorkflowStep{Template: hook.Template, TemplateRef: hook.TemplateRef}, tmplCtx, hook.Arguments,
				&executeTemplateOpts{nodeFlag: &wfv1.NodeFlag{Hooked: true}},
			)
			// Linked whenever it exists, errored or not, as reconcileHookNode links.
			if woc.wf.Status.Nodes.Has(woc.wf.ResolveNodeID(hookNodeName)) {
				woc.addChildNode(ctx, woc.wf.Name, hookNodeName)
			}
			if err != nil {
				return true, err
			}
			hookNodes = append(hookNodes, hookNode)
			woc.reconcileTaskSetFor(ctx, hookNode)
		}
	}
	for _, hookNode := range hookNodes {
		if !hookNode.Fulfilled() {
			return false, nil
		}
	}

	return true, nil
}

func (woc *wfOperationCtx) executeTmplLifeCycleHook(ctx context.Context, scope *wfScope, lifeCycleHooks wfv1.LifecycleHooks, parentNode *wfv1.NodeStatus, boundaryID string, tmplCtx *templateresolution.TemplateContext, ref varkeys.NodeRefKeys, name string) (bool, error) {
	var hookNodes []*wfv1.NodeStatus
	for hookName, hook := range lifeCycleHooks {
		// exit hook will be executed in runOnExitNode
		if hookName == wfv1.ExitLifecycleEvent {
			continue
		}
		hookNodeName := generateLifeHookNodeName(parentNode.Name, string(hookName))
		// To check a node was triggered
		hookedNode, _ := woc.wf.GetNodeByName(hookNodeName)
		if hook.Expression == "" {
			return false, woc.errorHookNode(ctx, hookNodeName, &hook, parentNode, boundaryID, tmplCtx, errors.Errorf(errors.CodeBadRequest, "Expression required for hook %s", hookNodeName))
		}
		// nil-preserving view so expressions can apply `??` fallbacks to skipped/omitted outputs
		execute, err := argoexpr.EvalBool(hook.Expression, env.GetFuncMap(scope.getParametersAny(woc.globalParams())))
		if err != nil {
			return false, woc.errorHookNode(ctx, hookNodeName, &hook, parentNode, boundaryID, tmplCtx, err)
		}
		// executeTemplated should be invoked when hookedNode != nil, because we should reexecute the function to check mutex condition, etc.
		if execute || hookedNode != nil {
			woc.log.WithField("lifeCycleHook", hookName).WithField("node", hookNodeName).WithField("hookName", hookName).Info(ctx, "Running hooks")
			hookNode, err := woc.reconcileHookNode(ctx, hookNodeName, &hook, parentNode, false, boundaryID, tmplCtx, ref, name, scope)
			if err != nil {
				return false, err
			}
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

// reconcileHookNode creates or advances nodeName, the node running hook (an
// exit hook when onExit) for parentNode, and links it under parentNode
// whenever it exists, also when it errored on creation, so retry and
// resubmit find its parent. An exit hook's arguments are always resolved
// against scope and the parent's outputs (they may name a sibling's absent
// optional output); a lifecycle hook's only once the parent has outputs.
func (woc *wfOperationCtx) reconcileHookNode(ctx context.Context, nodeName string, hook *wfv1.LifecycleHook, parentNode *wfv1.NodeStatus, onExit bool, boundaryID string, tmplCtx *templateresolution.TemplateContext, ref varkeys.NodeRefKeys, name string, scope *wfScope) (*wfv1.NodeStatus, error) {
	args := hook.Arguments
	outputs := parentNode.Outputs
	if lastChildNode := woc.possiblyGetRetryChildNode(parentNode); lastChildNode != nil {
		outputs = lastChildNode.Outputs
	}
	if !args.IsEmpty() && (onExit || outputs != nil) {
		var err error
		if args, err = woc.resolveExitTmplArgument(ctx, hook.Arguments, ref, name, outputs, scope); err != nil {
			return nil, woc.errorHookNode(ctx, nodeName, hook, parentNode, boundaryID, tmplCtx, err)
		}
	}
	hookNode, err := woc.reconcileTemplate(ctx, nodeName, toTemplateReferenceHolder(hook), tmplCtx, args, &executeTemplateOpts{
		boundaryID:     boundaryID,
		onExitTemplate: onExit,
		nodeFlag:       &wfv1.NodeFlag{Hooked: true},
	})
	if woc.wf.Status.Nodes.Has(woc.wf.ResolveNodeID(nodeName)) {
		woc.addChildNode(ctx, parentNode.Name, nodeName)
	}
	return hookNode, err
}

// errorHookNode records err, a hook's error, on its node nodeName as Error,
// creating the node, linked under parentNode, if the hook never got one; a
// finished hook node keeps its phase. It returns err.
func (woc *wfOperationCtx) errorHookNode(ctx context.Context, nodeName string, hook *wfv1.LifecycleHook, parentNode *wfv1.NodeStatus, boundaryID string, tmplCtx *templateresolution.TemplateContext, err error) error {
	if node, _ := woc.wf.GetNodeByName(nodeName); node == nil || !node.Fulfilled() {
		woc.initializeNodeOrMarkError(ctx, node, nodeName, tmplCtx.GetTemplateScope(), toTemplateReferenceHolder(hook), boundaryID, &wfv1.NodeFlag{Hooked: true}, err)
		woc.addChildNode(ctx, parentNode.Name, nodeName)
	}
	return err
}

// reconcileTaskSetFor dispatches the HTTP/plugin nodes at or under node to
// the agent now, for a hook node driven where operate's own task-set
// reconciliation does not reach it.
func (woc *wfOperationCtx) reconcileTaskSetFor(ctx context.Context, node *wfv1.NodeStatus) {
	if node != nil && woc.nodeRequiresTaskSetReconciliation(ctx, node.Name) {
		woc.taskSetReconciliation(ctx)
	}
}

func generateLifeHookNodeName(parentNodeName string, hookName string) string {
	return fmt.Sprintf("%s.hooks.%s", parentNodeName, hookName)
}
