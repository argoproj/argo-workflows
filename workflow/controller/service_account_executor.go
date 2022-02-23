package controller

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (woc *wfOperationCtx) getExecutorServiceAccountName(tmpl *wfv1.Template) string {
	if tmpl.Executor != nil && tmpl.Executor.ServiceAccountName != "" {
		return tmpl.Executor.ServiceAccountName
	}
	if woc.execWf.Spec.Executor != nil && woc.execWf.Spec.Executor.ServiceAccountName != "" {
		return woc.execWf.Spec.Executor.ServiceAccountName
	}
	return woc.controller.Config.GetDefaultExecutorServiceAccountName()
}

func (woc *wfOperationCtx) getExecutorServiceAccountTokenVolume(secretName string) apiv1.Volume {
	return woc.getServiceAccountTokenVolume(woc.getExecutorServiceAccountTokenVolumeName(), secretName)
}

func (woc *wfOperationCtx) getExecutorServiceAccountTokenVolumeMount() apiv1.VolumeMount {
	return apiv1.VolumeMount{
		Name:      woc.getExecutorServiceAccountTokenVolumeName(),
		MountPath: common.ServiceAccountTokenMountPath,
		ReadOnly:  true,
	}
}

func (woc *wfOperationCtx) getExecutorServiceAccountTokenVolumeName() string {
	return fmt.Sprintf("exec-sa-token-%s", woc.serviceAccountNonce)
}
