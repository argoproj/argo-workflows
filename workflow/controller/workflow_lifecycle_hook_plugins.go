package controller

import (
	"context"

	controllerplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
	errorsutil "github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/patch"
)

func (woc *wfOperationCtx) runWorkflowPreOperatePlugins() error {
	plugs := woc.controller.getControllerPlugins()
	args := controllerplugins.WorkflowPreOperateArgs{Workflow: &controllerplugins.Workflow{ObjectMeta: woc.wf.ObjectMeta}}
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
func (woc *wfOperationCtx) runWorkflowPostOperatePlugins(ctx context.Context) {
	plugs := woc.controller.getControllerPlugins()
	args := controllerplugins.WorkflowPostOperateArgs{Old: &controllerplugins.Workflow{ObjectMeta: woc.orig.ObjectMeta}, New: &controllerplugins.Workflow{ObjectMeta: woc.wf.ObjectMeta}}
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
}
