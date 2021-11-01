package controller

import (
	"github.com/argoproj/argo-workflows/v3/workflow/controller/plugins"
)

func (woc *wfOperationCtx) runParameterSubstitutionPlugins(p map[string]string) error {
	args := plugins.ParameterPreSubstitutionArgs{Workflow: woc.wf}
	reply := &plugins.ParameterPreSubstitutionReply{Parameters: p}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.ParameterSubstitutionPlugin); ok {
			if err := plug.ParameterPreSubstitution(args, reply); err != nil {
				return err
			}
		}
	}
	return nil
}
