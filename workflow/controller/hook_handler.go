package controller

import (
	"context"
	"fmt"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v4/util/expr/env"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/dag"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

// hookHandler drives the lifecycle hooks and exit hooks of a DAG's tasks or
// a Steps template's steps for the Engine.
type hookHandler struct {
	woc        *wfOperationCtx
	tmplCtx    *templateresolution.TemplateContext
	boundaryID string
	ref        varkeys.NodeRefKeys // sibling-node variable keys ("tasks" or "steps")
	log        logging.Logger
}

func newHookHandler(woc *wfOperationCtx, tmplCtx *templateresolution.TemplateContext, boundaryID string, tmpl *wfv1.Template, log logging.Logger) *hookHandler {
	ref := varkeys.TasksNodeRef
	if tmpl.GetType() == wfv1.TemplateTypeSteps {
		ref = varkeys.StepsNodeRef
	}
	return &hookHandler{woc: woc, tmplCtx: tmplCtx, boundaryID: boundaryID, ref: ref, log: log}
}

// hasHooks reports whether the task has any lifecycle or exit hook to drive.
// A task without one needs no scope: building it walks every ancestor.
func (h *hookHandler) hasHooks(task dag.Task) bool {
	return len(task.GetHooks()) > 0 || task.GetExitHook(h.woc.execWf.Spec.Arguments) != nil
}

// DriveTaskHooks drives the hooks of node, task's node: its lifecycle hooks
// and, once they are done and node has completed, its exit hook (see
// driveExitHook). refName is the name the hooks refer to the task by
// ({{tasks.<refName>.status}}). It reports whether every hook is done. A hook
// error is returned, except back-pressure (parallelism, rate limit, operation
// deadline), which is not the task's failure: the hook stays not done and is
// tried again. The Engine's walk calls it once per task per reconcile, so an
// exit hook is driven at most once per reconcile: driving it again would run
// checkParallelism against a pod count this reconcile just bumped (#14392).
func (h *hookHandler) DriveTaskHooks(ctx context.Context, task dag.Task, refName string, node *wfv1.NodeStatus, scope *wfScope) (done bool, err error) {
	h.ref.Status.Set(scope.scope, string(node.Phase), refName)
	done, err = h.woc.executeTmplLifeCycleHook(ctx, scope, task.GetHooks(), node, h.boundaryID, h.tmplCtx, h.ref, refName)
	if err != nil || !done {
		return false, h.ignoreThrottle(ctx, node, err)
	}
	done, err = h.driveExitHook(ctx, task, refName, node, scope)
	return done, h.ignoreThrottle(ctx, node, err)
}

// driveExitHook creates or re-enters the exit hook node of node, once node has
// completed, and reports whether that hook is done. It follows the workflow
// onExit's rule: an existing hook node is always re-entered, to advance it
// (also under Terminate) and, once it has finished, to release its lock and
// emit its metrics (handleNodeFulfilled). A new one is created only when the
// shutdown strategy lets exit handlers run and the hook's expression is true.
func (h *hookHandler) driveExitHook(ctx context.Context, task dag.Task, refName string, node *wfv1.NodeStatus, scope *wfScope) (bool, error) {
	exitHook := task.GetExitHook(h.woc.execWf.Spec.Arguments)
	if exitHook == nil || !node.Fulfilled() || !node.Completed() {
		return true, nil
	}
	onExitNodeName := common.GenerateOnExitNodeName(node.Name)
	if onExitNode, _ := h.woc.wf.GetNodeByName(onExitNodeName); onExitNode == nil {
		if !h.woc.GetShutdownStrategy().ShouldExecute(true) {
			return true, nil
		}
		if exitHook.Expression != "" {
			// nil-preserving view so expressions can apply `??` fallbacks to skipped/omitted outputs
			execute, err := argoexpr.EvalBool(exitHook.Expression, env.GetFuncMap(scope.getParametersAny(h.woc.globalParams())))
			if err != nil {
				return false, h.woc.errorHookNode(ctx, onExitNodeName, exitHook, node, h.boundaryID, h.tmplCtx, err)
			}
			if !execute {
				return true, nil
			}
		}
		h.log.Info(ctx, fmt.Sprintf("Running OnExit node for %s", node.Name))
	}
	onExitNode, err := h.woc.reconcileHookNode(ctx, onExitNodeName, exitHook, node, true, h.boundaryID, h.tmplCtx, h.ref, refName, scope)
	if err != nil {
		return false, err
	}
	if shutdown := h.woc.GetShutdownStrategy(); shutdown.Enabled() && shutdown.ShouldExecute(true) && !onExitNode.Fulfilled() {
		// operate skips task-set reconciliation while shutting down, but
		// under Stop an exit hook still runs: hand its HTTP/plugin nodes to
		// the agent here, as the workflow's onExit does. Under Terminate
		// nothing new is handed to the agent.
		h.woc.reconcileTaskSetFor(ctx, onExitNode)
	}
	return onExitNode.Fulfilled(), nil
}

// reenterHooks advances the hook nodes of node that had not finished when this
// reconcile started, lifecycle and exit alike, and creates none. It reports
// whether they have all finished. It serves the hook nodes an older controller
// created where this one creates none (main ran a DAG task's lifecycle hooks
// on its TaskGroup), and those of a node whose hooks' scope cannot be built:
// a hook node that is not a pod (a nested template, a suspend) only advances
// when re-entered, and re-entry releases its lock and emits its metrics once
// it has finished.
func (h *hookHandler) reenterHooks(ctx context.Context, task dag.Task, refName string, node *wfv1.NodeStatus, scope *wfScope) (bool, error) {
	if scope != nil {
		h.ref.Status.Set(scope.scope, string(node.Phase), refName)
	}
	done := true
	for _, child := range h.hookNodesToReenter(node) {
		hook, onExit := task.GetExitHook(h.woc.execWf.Spec.Arguments), true
		if child.Name != common.GenerateOnExitNodeName(node.Name) {
			hook, onExit = nil, false
			if lifecycleHook, ok := task.GetHooks()[wfv1.LifecycleEvent(strings.TrimPrefix(child.Name, node.Name+".hooks."))]; ok {
				hook = &lifecycleHook
			}
		}
		if hook == nil {
			continue
		}
		hookNode, err := h.woc.reconcileHookNode(ctx, child.Name, hook, node, onExit, h.boundaryID, h.tmplCtx, h.ref, refName, scope)
		if err != nil {
			return false, h.ignoreThrottle(ctx, node, err)
		}
		done = done && hookNode.Fulfilled()
	}
	return done, nil
}

// hookNodesToReenter lists node's hook nodes that had not finished when this
// reconcile started.
func (h *hookHandler) hookNodesToReenter(node *wfv1.NodeStatus) []*wfv1.NodeStatus {
	var out []*wfv1.NodeStatus
	for _, childID := range node.Children {
		child, err := h.woc.wf.Status.Nodes.Get(childID)
		if err != nil || child.NodeFlag == nil || !child.NodeFlag.Hooked {
			continue
		}
		if prev, ok := h.woc.preExecutionNodeStatuses[child.ID]; !ok || !prev.Fulfilled() {
			out = append(out, child)
		}
	}
	return out
}

// ignoreThrottle drops deliberate back-pressure (parallelism, rate limit,
// operation deadline) on node's hooks: it is not the task's failure, and the
// hook is tried again later.
func (h *hookHandler) ignoreThrottle(ctx context.Context, node *wfv1.NodeStatus, err error) error {
	if isThrottleErr(err) {
		h.log.WithError(err).WithField("node", node.Name).Debug(ctx, "task hook held back; will retry")
		return nil
	}
	return err
}

func toTemplateReferenceHolder(lifecycleHook *wfv1.LifecycleHook) wfv1.TemplateReferenceHolder {
	return &wfv1.WorkflowStep{
		Template:    lifecycleHook.Template,
		TemplateRef: lifecycleHook.TemplateRef,
	}
}
