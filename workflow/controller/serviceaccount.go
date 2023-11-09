package controller

import (
	"context"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func (woc *wfOperationCtx) getServiceAccountTokenVolume(ctx context.Context, serviceAccountName string) (*apiv1.Volume, *apiv1.VolumeMount, error) {
	secretName, err := woc.getServiceAccountTokenName(ctx, serviceAccountName)
	if err != nil {
		return nil, nil, err
	}
	// Intentionally randomize the name so that plugins cannot determine it.
	tokenVolumeName := fmt.Sprintf("kube-api-access-%s", rand.String(5))
	return &apiv1.Volume{
			Name: tokenVolumeName,
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{SecretName: secretName},
			},
		},
		&apiv1.VolumeMount{
			Name:      tokenVolumeName,
			MountPath: common.ServiceAccountTokenMountPath,
			ReadOnly:  true,
		},
		nil
}
