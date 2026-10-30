package controller

import (
	"context"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

const (
	readOnlyFileMode int32 = 0o444
	oneHour          int64 = 3600
)

// Projected tokens must use the agent pod's service account.
func (woc *wfOperationCtx) getServiceAccountTokenVolume(ctx context.Context, serviceAccountName string, projected bool) (*apiv1.Volume, *apiv1.VolumeMount, error) {
	var source apiv1.VolumeSource
	if projected {
		source.Projected = woc.getProjectedServiceAccountTokenVolumeSource()
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

func (woc *wfOperationCtx) getProjectedServiceAccountTokenVolumeSource() *apiv1.ProjectedVolumeSource {
	serviceAccountTokenProjection := &apiv1.ServiceAccountTokenProjection{
		// Kubelet requests and rotates this token automatically.
		Path:              "token",
		ExpirationSeconds: new(oneHour),
	}
	kubernetesAPICAProjection := &apiv1.ConfigMapProjection{
		// Provide the CA bundle for verifying the Kubernetes API server's TLS certificate.
		LocalObjectReference: apiv1.LocalObjectReference{Name: "kube-root-ca.crt"},
		Items:                []apiv1.KeyToPath{{Key: "ca.crt", Path: "ca.crt"}},
	}
	podNamespaceProjection := &apiv1.DownwardAPIProjection{
		Items: []apiv1.DownwardAPIVolumeFile{{
			// Project the namespace explicitly since we no longer mount the service account Secret containing it.
			Path: "namespace",
			FieldRef: &apiv1.ObjectFieldSelector{
				APIVersion: "v1",
				FieldPath:  "metadata.namespace",
			},
		}},
	}
	return &apiv1.ProjectedVolumeSource{
		// Allow plugins running as non-root to read the projected files.
		DefaultMode: new(readOnlyFileMode),
		Sources: []apiv1.VolumeProjection{
			{ServiceAccountToken: serviceAccountTokenProjection},
			{ConfigMap: kubernetesAPICAProjection},
			{DownwardAPI: podNamespaceProjection},
		},
	}
}
