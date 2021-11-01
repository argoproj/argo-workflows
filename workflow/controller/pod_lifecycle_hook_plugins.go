package controller

import (
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/controller/plugins"
)

func (woc *wfOperationCtx) runPodPreCreatePlugins(tmpl *wfv1.Template, pod *apiv1.Pod) error {
	args := plugins.PodPreCreateArgs{Workflow: woc.wf, Template: tmpl}
	reply := &plugins.PodPreCreateReply{Pod: pod}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.PodLifecycleHook); ok {
			if err := plug.PodPreCreate(args, reply); err != nil {
				return err
			}
		}
	}
	return nil
}

func (woc *wfOperationCtx) runPodPostCreatePlugins(tmpl *wfv1.Template, pod *apiv1.Pod) error {
	args := plugins.PodPostCreateArgs{Workflow: woc.wf, Template: tmpl, Pod: pod}
	reply := &plugins.PodPostCreateReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(plugins.PodLifecycleHook); ok {
			if err := plug.PodPostCreate(args, reply); err != nil {
				return err
			}
		}
	}
	return nil
}
