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
	wfutil "github.com/argoproj/argo-workflows/v4/workflow/util"
)

func (woc *wfOperationCtx) executeWfLifeCycleHook(ctx context.Context, tmplCtx *templateresolution.TemplateContext) (bool, error) {
	var hookNodes []*wfv1.NodeStatus
	heldBack := false
	entryNode, _ := woc.wf.GetNodeByName(woc.wf.Name)
	for hookName, hook := range woc.execWf.Spec.Hooks {
		// the exit hook runs as the workflow's onExit node, once the entry node has finished (operate)
		if hookName == wfv1.ExitLifecycleEvent {
			continue
		}
		hookNodeName := generateLifeHookNodeName(woc.wf.Name, string(hookName))
		// To check a node was triggered.
		hookedNode, _ := woc.wf.GetNodeByName(hookNodeName)
		if hook.Expression == "" {
			// No node exists yet for this hook (reconcileTemplate never ran),
			// so record the error on its own hook node, as errorHookNode
			// does for a template-level hook: the entry node may
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
			hookNode, err := woc.reconcileTemplate(ctx, hookNodeName, toTemplateReferenceHolder(&hook), tmplCtx, hook.Arguments,
				&executeTemplateOpts{nodeFlag: &wfv1.NodeFlag{Hooked: true}, recordPendingOnThrottle: true},
			)
			// Linked whenever it exists, errored or not, as reconcileHookNode links.
			if woc.wf.Status.Nodes.Has(woc.wf.ResolveNodeID(hookNodeName)) {
				woc.addChildNode(ctx, woc.wf.Name, hookNodeName)
			}
			if err != nil {
				if isThrottleErr(err) {
					// Back-pressure (parallelism, rate limit, operation deadline), not a
					// failure: hold this hook back so the workflow stays Running and the
					// hook is retried once a slot frees up, as template-level hooks do
					// (hookHandler.ignoreThrottle). Sibling hooks are still driven. See #17511.
					woc.log.WithError(err).WithField("lifeCycleHook", hookName).Debug(ctx, "workflow level hook held back; will retry")
					heldBack = true
					continue
				}
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
	// A hook held back by back-pressure has no node in hookNodes to wait on,
	// but its recorded node keeps it pending: the workflow is not done yet.
	if heldBack {
		return false, nil
	}

	return true, nil
}

func (woc *wfOperationCtx) executeTmplLifeCycleHook(ctx context.Context, scope *wfScope, lifeCycleHooks wfv1.LifecycleHooks, parentNode *wfv1.NodeStatus, boundaryID string, tmplCtx *templateresolution.TemplateContext, ref varkeys.NodeRefKeys, name string) (bool, error) {
	var hookNodes []*wfv1.NodeStatus
	for hookName, hook := range lifeCycleHooks {
		// the exit hook runs once these hooks are done (hookHandler.driveExitHook)
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

// recordPendingHookNode records a workflow-level hook that back-pressure held
// back as a node of its own, so the hook's trigger is not lost: the trigger
// lives in the node, and once the entry node finishes the hook's expression
// stops matching, so without a node the hook would be silently dropped instead
// of waiting for a free slot (#17511). Linking it to the workflow is the
// caller's job, as it is for every other hook node.
//
// The node is created exactly as its own dispatch would create it — same type,
// scope, inputs and phase — but without a MemoizationStatus: handleMemoization
// runs after the constraints, so the cache has not been consulted yet and must
// not be recorded as checked before the hook can actually run.
func (woc *wfOperationCtx) recordPendingHookNode(ctx context.Context, nodeName string, processedTmpl *wfv1.Template, tmplCtx *templateresolution.TemplateContext, orgTmpl wfv1.TemplateReferenceHolder, opts *executeTemplateOpts) {
	if _, err := woc.wf.GetNodeByName(nodeName); err == nil {
		return // already recorded by an earlier operation
	}
	nodeType := wfutil.GetNodeType(processedTmpl)
	// Composite and retry nodes initialize straight to Running (executeSteps,
	// executeDAG, handleRetries); every other dispatch re-enters a Pending node.
	phase := wfv1.NodePending
	if nodeType == wfv1.NodeTypeSteps || nodeType == wfv1.NodeTypeDAG || nodeType == wfv1.NodeTypeRetry {
		phase = wfv1.NodeRunning
	}
	// tmplCtx is the context the hook's reference was made from, so its scope
	// is the one reconcileTemplate records for this task (dt.TemplateScope) —
	// the scope an unthrottled dispatch would give the node, even when the
	// hook resolves its template through a templateRef.
	_, node := woc.initializeExecutableNode(ctx, nodeName, nodeType, tmplCtx.GetTemplateScope(), processedTmpl, orgTmpl, opts.boundaryID, phase, opts.nodeFlag, processedTmpl.IsDaemon())
	if node.MemoizationStatus != nil {
		node.MemoizationStatus = nil
		woc.wf.Status.Nodes.Set(ctx, node.ID, *node)
		woc.updated = true
	}
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
