package controller

import (
	controllerplugins "github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
)

func (woc *wfOperationCtx) runParameterSubstitutionPlugins(p map[string]string) error {
	plugs := woc.controller.getControllerPlugins()
	args := controllerplugins.ParameterPreSubstitutionArgs{Workflow: &controllerplugins.Workflow{
		ObjectMeta: woc.wf.ObjectMeta,
	}}
	reply := &controllerplugins.ParameterPreSubstitutionReply{}
	for _, plug := range plugs {
		if plug, ok := plug.(controllerplugins.ParameterSubstitutionPlugin); ok {
			if err := plug.AddParameters(args, reply); err != nil {
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
