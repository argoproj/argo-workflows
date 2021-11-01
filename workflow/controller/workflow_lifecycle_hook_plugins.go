package controller

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/workflow/controller/plugins"
)

func (wfc *WorkflowController) runWorkflowPreOperatePlugins(ctx context.Context, woc *wfOperationCtx) {
	args := plugins.WorkflowPreOperateArgs{Workflow: woc.wf}
	reply := &plugins.WorkflowPreOperateReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.WorkflowLifecycleHook); ok {
			if err := plug.WorkflowPreOperate(args, reply); err != nil {
				woc.markWorkflowError(ctx, err)
			} else if wf := reply.Workflow; wf != nil {
				wf.DeepCopyInto(woc.wf)
			}
		}
	}
}

func (woc *wfOperationCtx) runWorkflowPreUpdatePlugins(ctx context.Context) {
	args := plugins.WorkflowPreUpdateArgs{Old: woc.orig, New: woc.wf}
	reply := &plugins.WorkflowPreUpdateReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.WorkflowLifecycleHook); ok {
			if err := plug.WorkflowPreUpdate(args, reply); err != nil {
				woc.markWorkflowError(ctx, err)
			} else if wf := reply.New; wf != nil {
				wf.DeepCopyInto(woc.wf)
			}
		}
	}
}
