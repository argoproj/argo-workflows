package controller

import (
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/plugins/controller"
)

func (woc *wfOperationCtx) runPodPreCreatePlugins(tmpl *wfv1.Template, pod *apiv1.Pod) error {
	args := controller.PodPreCreateArgs{Workflow: woc.wf.Reduced(), Template: tmpl, Pod: pod}
	reply := &controller.PodPreCreateReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(controller.PodLifecycleHook); ok {
			if err := plug.PodPreCreate(args, reply); err != nil {
				return err
			} else if reply.Pod != nil {
				*pod = *reply.Pod
			}
		}
	}
	return nil
}

func (woc *wfOperationCtx) runPodPostCreatePlugins(tmpl *wfv1.Template, pod *apiv1.Pod) error {
	args := controller.PodPostCreateArgs{Workflow: woc.wf.Reduced(), Template: tmpl, Pod: pod}
	reply := &controller.PodPostCreateReply{}
	for _, sym := range woc.controller.plugins {
		if plug, ok := sym.(controller.PodLifecycleHook); ok {
			if err := plug.PodPostCreate(args, reply); err != nil {
				return err
			}
		}
	}
	return nil
}
