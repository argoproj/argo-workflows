package controller

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/workflow/controller/plugins"
)

func (wfc *WorkflowController) runWorkflowPreOperatePlugins(ctx context.Context, woc *wfOperationCtx) {
	args := plugins.WorkflowPreOperateArgs{}
	reply := &plugins.WorkflowPreOperateReply{Workflow: woc.wf}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.WorkflowLifecycleHook); ok {
			if err := plug.WorkflowPreOperate(args, reply); err != nil {
				woc.markWorkflowError(ctx, err)
			}
		}
	}
}

func (woc *wfOperationCtx) runWorkflowPreUpdatePlugins(ctx context.Context) {
	args := plugins.WorkflowPreUpdateArgs{Old: woc.orig}
	reply := &plugins.WorkflowPreUpdateReply{New: woc.wf}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.WorkflowLifecycleHook); ok {
			if err := plug.WorkflowPreUpdate(args, reply); err != nil {
				woc.markWorkflowError(ctx, err)
			}
		}
	}
}
