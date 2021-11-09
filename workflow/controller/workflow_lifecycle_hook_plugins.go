package controller

import (
	"context"

	"github.com/sirupsen/logrus"

	plugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
)

func (woc *wfOperationCtx) runWorkflowPreOperatePlugins(ctx context.Context) {
	args := plugins.WorkflowPreOperateArgs{Workflow: woc.wf}
	reply := &plugins.WorkflowPreOperateReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.WorkflowLifecycleHook); ok {
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
	args := plugins.WorkflowPostOperateArgs{Old: woc.orig, New: woc.wf}
	reply := &plugins.WorkflowPostOperateReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.WorkflowLifecycleHook); ok {
			if err := plug.WorkflowPostOperate(args, reply); err != nil {
				woc.markWorkflowError(ctx, err)
			} else if wf := reply.New; wf != nil {
				woc.wf = wf
			}
		}
	}
}
