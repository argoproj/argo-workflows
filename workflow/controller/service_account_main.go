package controller

import (
	"fmt"

	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (woc *wfOperationCtx) getAutomountMainServiceAccountToken(tmpl *wfv1.Template) bool {
	if tmpl.AutomountServiceAccountToken != nil {
		return *tmpl.AutomountServiceAccountToken
	}
	if woc.execWf.Spec.AutomountServiceAccountToken != nil {
		return *woc.execWf.Spec.AutomountServiceAccountToken
	}
	return false
}

func (woc *wfOperationCtx) getMainServiceAccountName(tmpl *wfv1.Template) string {
	if tmpl.ServiceAccountName != "" {
		return tmpl.ServiceAccountName
	}
	if woc.execWf.Spec.ServiceAccountName != "" {
		return woc.execWf.Spec.ServiceAccountName
	}
	return woc.controller.Config.GetDefaultServiceAccountName()
}

func (woc *wfOperationCtx) getMainServiceAccountTokenVolumeMount() apiv1.VolumeMount {
	return apiv1.VolumeMount{
		Name:      woc.getMainServiceAccountVolumeName(),
		MountPath: common.ServiceAccountTokenMountPath,
		ReadOnly:  true,
	}
}

func (woc *wfOperationCtx) getMainServiceAccountTokenVolume(secretName string) apiv1.Volume {
	return apiv1.Volume{
		Name: woc.getMainServiceAccountVolumeName(),
		VolumeSource: apiv1.VolumeSource{
			Secret: &apiv1.SecretVolumeSource{
				SecretName: secretName,
			},
		},
	}
}

func (woc *wfOperationCtx) getMainServiceAccountVolumeName() string {
	return fmt.Sprintf("main-sa-token-%s", woc.serviceAccountNonce)
}
