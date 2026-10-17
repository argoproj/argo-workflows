package controller

import (
	"context"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// Projected tokens must use the agent pod's service account.
func (woc *wfOperationCtx) getServiceAccountTokenVolume(ctx context.Context, serviceAccountName string, projected bool) (*apiv1.Volume, *apiv1.VolumeMount, error) {
	var source apiv1.VolumeSource
	if projected {
		source.Projected = &apiv1.ProjectedVolumeSource{
			DefaultMode: new(int32(0o644)),
			Sources: []apiv1.VolumeProjection{
				{ServiceAccountToken: &apiv1.ServiceAccountTokenProjection{
					Path:              "token",
					ExpirationSeconds: new(int64(3600)),
				}},
				{ConfigMap: &apiv1.ConfigMapProjection{
					LocalObjectReference: apiv1.LocalObjectReference{Name: "kube-root-ca.crt"},
					Items:                []apiv1.KeyToPath{{Key: "ca.crt", Path: "ca.crt"}},
				}},
				{DownwardAPI: &apiv1.DownwardAPIProjection{
					Items: []apiv1.DownwardAPIVolumeFile{{
						Path: "namespace",
						FieldRef: &apiv1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "metadata.namespace",
						},
					}},
				}},
			},
		}
	} else {
		secretName, err := woc.getServiceAccountTokenName(ctx, serviceAccountName)
		if err != nil {
			return nil, nil, err
		}
		source.Secret = &apiv1.SecretVolumeSource{SecretName: secretName}
	}
	// Intentionally randomize the name so that plugins cannot determine it.
	tokenVolumeName := fmt.Sprintf("kube-api-access-%s", rand.String(5))
	return &apiv1.Volume{
			Name:         tokenVolumeName,
			VolumeSource: source,
		},
		&apiv1.VolumeMount{
			Name:      tokenVolumeName,
			MountPath: common.ServiceAccountTokenMountPath,
			ReadOnly:  true,
		},
		nil
}
