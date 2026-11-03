package controller

import (
	"context"
	"fmt"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

// getAgentTokenVolume mounts the agent's token. Since Kubernetes 1.24 no token secret is created for a
// ServiceAccount, so unless it explicitly references one, a projected token is used. The pod keeps
// automountServiceAccountToken=false; the projection is an explicit volume that is only mounted into the agent.
func (woc *wfOperationCtx) getAgentTokenVolume(ctx context.Context, serviceAccountName string) (*apiv1.Volume, *apiv1.VolumeMount, error) {
	if serviceAccountName == "" {
		serviceAccountName = "default"
	}
	account, err := woc.controller.kubeclientset.CoreV1().ServiceAccounts(woc.wf.Namespace).Get(ctx, serviceAccountName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}
	if len(account.Secrets) > 0 || account.Annotations[common.AnnotationKeyServiceAccountTokenName] != "" {
		return woc.getServiceAccountTokenVolume(ctx, serviceAccountName)
	}
	tokenVolumeName := fmt.Sprintf("kube-api-access-%s", rand.String(5))
	expiration := int64(3607) // kubelet default
	return &apiv1.Volume{
			Name: tokenVolumeName,
			VolumeSource: apiv1.VolumeSource{
				Projected: &apiv1.ProjectedVolumeSource{
					Sources: []apiv1.VolumeProjection{
						{ServiceAccountToken: &apiv1.ServiceAccountTokenProjection{Path: "token", ExpirationSeconds: &expiration}},
						{ConfigMap: &apiv1.ConfigMapProjection{
							LocalObjectReference: apiv1.LocalObjectReference{Name: "kube-root-ca.crt"},
							Items:                []apiv1.KeyToPath{{Key: "ca.crt", Path: "ca.crt"}},
						}},
						{DownwardAPI: &apiv1.DownwardAPIProjection{
							Items: []apiv1.DownwardAPIVolumeFile{{Path: "namespace", FieldRef: &apiv1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.namespace"}}},
						}},
					},
				},
			},
		},
		&apiv1.VolumeMount{Name: tokenVolumeName, MountPath: common.ServiceAccountTokenMountPath, ReadOnly: true},
		nil
}

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
