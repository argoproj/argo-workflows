package controller

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
)

func (woc *wfOperationCtx) runWorkflowPreOperatePlugins(ctx context.Context) {
	args := controller.WorkflowPreOperateArgs{Workflow: woc.wf}
	reply := &controller.WorkflowPreOperateReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(controller.WorkflowLifecycleHook); ok {
			if err := plug.WorkflowPreOperate(args, reply); err != nil {
				woc.markWorkflowError(ctx, err)
			} else if wf := reply.Workflow; wf != nil {
				logrus.Info("plugin invoked")
				woc.wf = wf
			}
		}
	}
}

func (woc *wfOperationCtx) runWorkflowPostOperatePlugins(ctx context.Context) {
	if !woc.updated {
		return
	}
	args := controller.WorkflowPostOperateArgs{Old: woc.orig, New: woc.wf}
	reply := &controller.WorkflowPostOperateReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(controller.WorkflowLifecycleHook); ok {
			if err := plug.WorkflowPostOperate(args, reply); err != nil {
				woc.markWorkflowError(ctx, err)
			} else if wf := reply.New; wf != nil {
				woc.wf = wf
			}
		}
	}
}
