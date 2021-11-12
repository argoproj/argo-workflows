package controller

import (
	"context"

	controllerplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/patch"
)

func (woc *wfOperationCtx) runWorkflowPreOperatePlugins() error {
	plugs, err := woc.controller.getControllerPlugins()
	if err != nil {
		return err
	}
	args := controllerplugins.WorkflowPreOperateArgs{Workflow: woc.wf}
	reply := &controllerplugins.WorkflowPreOperateReply{}
	for _, plug := range plugs {
		if plug, ok := plug.(controllerplugins.WorkflowLifecycleHook); ok {
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
func (woc *wfOperationCtx) runWorkflowPostOperatePlugins(ctx context.Context) error {
	if !woc.updated {
		return nil
	}
	plugs, err := woc.controller.getControllerPlugins()
	if err != nil {
		return err
	}
	args := controllerplugins.WorkflowPostOperateArgs{Old: woc.orig, New: woc.wf}
	reply := &controllerplugins.WorkflowPostOperateReply{}
	for _, plug := range plugs {
		if plug, ok := plug.(controllerplugins.WorkflowLifecycleHook); ok {
			if err := plug.WorkflowPostOperate(args, reply); err != nil {
				if !errorsutil.IsTransientErr(err) {
					woc.markWorkflowError(ctx, err)
				}
			}
		}
	}
	return nil
}
