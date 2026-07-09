package controller

import (
	"context"
	"fmt"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/expr/argoexpr"
	"github.com/argoproj/argo-workflows/v4/util/expr/env"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	varkeys "github.com/argoproj/argo-workflows/v4/util/variables/keys"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/common/dag"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
)

// hookHandler encapsulates all lifecycle hook and exit handler logic for the Engine.
// It keeps hook orchestration in one place rather than scattered across Execute and executeTask.
type hookHandler struct {
	woc        *wfOperationCtx
	tmplCtx    *templateresolution.TemplateContext
	boundaryID string
	prefix     string              // "tasks" or "steps"
	ref        varkeys.NodeRefKeys // sibling-node variable keys matching prefix
	log        logging.Logger
	// exitDriven records tasks whose exit handler has been driven this operate
	// cycle. ProcessAllTaskHooks runs twice per cycle; the second time a completed
	// task is seen, its onExit node is inspected by name rather than re-run, so the
	// exit handler is driven at most once per cycle (#14392 / PR #16088). The
	// hookHandler is created per cycle in NewEngine, so this map is cycle-scoped.
	exitDriven map[string]bool
}

func newHookHandler(woc *wfOperationCtx, tmplCtx *templateresolution.TemplateContext, boundaryID string, tmpl *wfv1.Template, log logging.Logger) *hookHandler {
	prefix, ref := "tasks", varkeys.TasksNodeRef
	if tmpl.GetType() == wfv1.TemplateTypeSteps {
		prefix, ref = "steps", varkeys.StepsNodeRef
	}
	return &hookHandler{
		woc:        woc,
		tmplCtx:    tmplCtx,
		boundaryID: boundaryID,
		prefix:     prefix,
		ref:        ref,
		log:        log,
		exitDriven: map[string]bool{},
	}
}

// ExecuteLifecycleHooks runs non-exit lifecycle hooks for a task node.
// It delegates to woc.executeTmplLifeCycleHook and returns whether all hooks have completed.
func (h *hookHandler) ExecuteLifecycleHooks(ctx context.Context, scope *wfScope, hooks wfv1.LifecycleHooks, taskNode *wfv1.NodeStatus, displayName string) (bool, error) {
	return h.woc.executeTmplLifeCycleHook(ctx, scope, hooks, taskNode, h.boundaryID, h.tmplCtx, h.ref, displayName)
}

// ExecuteExitHandler evaluates and runs the exit handler for a completed task.
// Returns (hasExitNode, exitNode, error). If exitHook is nil, returns (false, nil, nil).
func (h *hookHandler) ExecuteExitHandler(ctx context.Context, exitHook *wfv1.LifecycleHook, taskNode *wfv1.NodeStatus, displayName string, scope *wfScope) (bool, *wfv1.NodeStatus, error) {
	if exitHook == nil {
		return false, nil, nil
	}

	if !h.woc.GetShutdownStrategy().ShouldExecute(true) {
		return false, nil, nil
	}

	if exitHook.Expression != "" {
		// nil-preserving view so expressions can apply `??` fallbacks to skipped/omitted outputs
		execute, err := argoexpr.EvalBool(exitHook.Expression,
			env.GetFuncMap(scope.getParametersAny(h.woc.globalParams())))
		if err != nil {
			return true, nil, err
		}
		if !execute {
			return false, nil, nil
		}
	}

	onExitNodeName := common.GenerateOnExitNodeName(taskNode.Name)
	onExitNode, err := h.woc.wf.GetNodeByName(onExitNodeName)
	creating := err != nil
	// Create the exit handler node, or re-enter reconcileTemplate to advance an
	// unfulfilled one (e.g., process pod completions in a nested Steps template).
	if creating || !onExitNode.Fulfilled() {
		if creating {
			h.log.Info(ctx, fmt.Sprintf("Running OnExit node for %s", taskNode.Name))
		}
		resolvedArgs := exitHook.Arguments
		// Resolve even when the task produced no outputs: exit-hook args may reference a SIBLING's
		// skipped/omitted output (an absent optional), which resolveExitTmplArgument rescues via the
		// scope. The task's own outputs (possibly nil) are merged in by resolveExitTmplArgument.
		if !resolvedArgs.IsEmpty() {
			resolvedArgs, err = h.woc.resolveExitTmplArgument(ctx,
				exitHook.Arguments, h.ref, displayName, taskNode.Outputs, scope)
			if err != nil {
				return true, nil, err
			}
		}
		onExitNode, err = h.woc.reconcileTemplate(ctx, onExitNodeName, toTemplateReferenceHolder(exitHook), h.tmplCtx, resolvedArgs, &executeTemplateOpts{
			boundaryID:     h.boundaryID,
			onExitTemplate: true,
			nodeFlag:       &wfv1.NodeFlag{Hooked: true},
			scope:          scope,
		})
		if err != nil {
			return true, nil, err
		}
		if creating && onExitNode != nil {
			h.woc.addChildNode(ctx, taskNode.Name, onExitNode.Name)
		}
	}
	return true, onExitNode, nil
}

// ProcessAllTaskHooks runs lifecycle hooks and exit handlers for all tasks
// that have nodes. Per-task errors are isolated: the failing task's hook
// error is forwarded to onError (which marks the offending task node Errored),
// and iteration continues so unrelated siblings' hooks and the engine's
// converge pass are not blocked. Returns onExitCompleted=false if any
// exit handler is still pending OR any hook errored, and always returns
// a nil error: callers must NOT treat per-task hook errors as boundary-fatal.
//
// This mirrors the legacy controller's executeDAGTask behavior, where a hook
// error on one task did not prevent sibling tasks from running their hooks.
//
// ProcessAllTaskHooks runs twice per operate cycle (before and after scheduling).
// A task's exit handler is driven (reconcileTemplate) at most once per cycle,
// tracked by h.exitDriven: the second time a completed task is seen, its onExit
// node is only looked up by name to gate completion, never re-run. This mirrors
// PR #16088 (#14392): re-running the exit handler would re-run checkParallelism
// against a pod count this cycle just bumped, spuriously failing the handler.
func (h *hookHandler) ProcessAllTaskHooks(ctx context.Context, tasks []dag.Task, getTaskNode func(ctx context.Context, taskName string) *wfv1.NodeStatus, buildScope func(ctx context.Context, task dag.Task) (*wfScope, error), onError func(ctx context.Context, taskNode *wfv1.NodeStatus, err error)) (onExitCompleted bool, err error) {
	onExitCompleted = true
	for _, task := range tasks {
		taskName := task.GetName()
		taskNode := getTaskNode(ctx, taskName)
		if taskNode == nil {
			continue
		}

		scope, scopeErr := buildScope(ctx, task)
		if scopeErr != nil {
			h.log.WithError(scopeErr).WithField("task", taskName).Error(ctx, "failed to build scope for task hooks; isolating to this task")
			onError(ctx, taskNode, scopeErr)
			onExitCompleted = false
			continue
		}
		h.ref.Status.Set(scope.scope, string(taskNode.Phase), task.GetDisplayName())

		hookCompleted, hookErr := h.ExecuteLifecycleHooks(ctx, scope, task.GetHooks(), taskNode, task.GetDisplayName())
		if hookErr != nil {
			h.log.WithError(hookErr).WithField("task", taskName).Error(ctx, "task lifecycle hook errored; isolating to this task")
			onError(ctx, taskNode, hookErr)
			onExitCompleted = false
			continue
		}
		if !hookCompleted {
			onExitCompleted = false
			continue
		}

		if taskNode.Fulfilled() && taskNode.Completed() {
			if h.exitDriven[taskName] {
				// Already driven this cycle (earlier pass). Do not re-run the exit
				// handler — look up the onExit node by name to gate completion, exactly
				// as PR #16088's executeDAG target loop does. Re-running reconcileTemplate
				// here would re-run checkParallelism against a pod count this cycle just
				// bumped, spuriously failing the handler (#14392).
				onExitNodeName := common.GenerateOnExitNodeName(taskNode.Name)
				if onExitNode, lookupErr := h.woc.wf.GetNodeByName(onExitNodeName); lookupErr == nil && onExitNode != nil && !onExitNode.Fulfilled() {
					onExitCompleted = false
				}
				continue
			}
			hasOnExitNode, onExitNode, exitErr := h.ExecuteExitHandler(ctx, task.GetExitHook(h.woc.execWf.Spec.Arguments), taskNode, task.GetDisplayName(), scope)
			h.exitDriven[taskName] = true
			if exitErr != nil {
				h.log.WithError(exitErr).WithField("task", taskName).Error(ctx, "task exit handler errored; isolating to this task")
				onError(ctx, taskNode, exitErr)
				onExitCompleted = false
				continue
			}
			if hasOnExitNode && (onExitNode == nil || !onExitNode.Fulfilled()) {
				onExitCompleted = false
			}
		}
	}
	return onExitCompleted, nil
}

func toTemplateReferenceHolder(lifecycleHook *wfv1.LifecycleHook) wfv1.TemplateReferenceHolder {
	return &wfv1.WorkflowStep{
		Template:    lifecycleHook.Template,
		TemplateRef: lifecycleHook.TemplateRef,
	}
}
