package controller

import (
	"context"
	"encoding/json"

	jsonpatch "github.com/evanphx/json-patch"
)

func (woc *wfOperationCtx) runWorkflowPreOperatePlugins() error {
	args := plugins.WorkflowPreOperateArgs{Workflow: woc.wf}
	reply := &plugins.WorkflowPreOperateReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.WorkflowLifecycleHook); ok {
			if err := plug.WorkflowPreOperate(args, reply); err != nil {
				return err
			} else if reply.Workflow != nil {
				if err := woc.patchObj(woc.wf, reply.Workflow); err != nil {
					return err
				}
				woc.updated = true
			}
		}
	}
	return nil
}

func (woc *wfOperationCtx) patchObj(old interface{}, patch interface{}) error {
	orig, err := json.Marshal(old)
	if err != nil {
		return err
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}
	mergePatch, err := jsonpatch.CreateMergePatch([]byte("{}"), patchBytes)
	if err != nil {
		return err
	}
	data, err := jsonpatch.MergePatch(orig, mergePatch)
	if err != nil {
		return nil
	}
	err = json.Unmarshal(data, old)
	if err != nil {
		return err
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
				woc.markWorkflowError(ctx, err)
			}
		}
	}
}
