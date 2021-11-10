package controller

import (
	"context"

	plugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/patch"
)

func (woc *wfOperationCtx) runWorkflowPreOperatePlugins() error {
	args := plugins.WorkflowPreOperateArgs{Workflow: woc.wf}
	reply := &plugins.WorkflowPreOperateReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.WorkflowLifecycleHook); ok {
			if err := plug.WorkflowPreOperate(args, reply); err != nil {
				return err
			} else if reply.Workflow != nil {
				if err := patch.Obj(woc.wf, reply.Workflow); err != nil {
					return err
				}
				woc.updated = true
			}
		}
	}
	return nil
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
				if !errorsutil.IsTransientErr(err) {
					woc.markWorkflowError(ctx, err)
				}
			}
		}
	}
}
