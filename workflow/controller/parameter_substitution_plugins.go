package controller

import (
	controller2 "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
)

func (woc *wfOperationCtx) runParameterSubstitutionPlugins(p map[string]string) error {
	args := controller2.ParameterPreSubstitutionArgs{Workflow: woc.wf.Reduced()}
	reply := &controller2.ParameterPreSubstitutionReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(controller2.ParameterSubstitutionPlugin); ok {
			if err := plug.ParameterPreSubstitution(args, reply); err != nil {
				return err
			} else if reply.Parameters != nil {
				for k, v := range reply.Parameters {
					p[k] = v
				}
			}
		}
	}
	return nil
}
