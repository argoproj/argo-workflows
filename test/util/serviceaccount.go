package util

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

// CreateServiceAccountWithToken creates a service account with a given name with a service account token.
// Need to use this function to simulate the actual behavior of Kubernetes API server with the fake client.
func CreateServiceAccountWithToken(ctx context.Context, dy dynamic.Interface, namespace, name, tokenName string) error {
	un, err := util.ServiceAccountToUnstructured(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name}})
	un, err = dy.Resource(common.ServiceAccountGVR).Namespace(namespace).Create(ctx, un, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: tokenName,
			Annotations: map[string]string{
				corev1.ServiceAccountNameKey: un.GetName(),
				corev1.ServiceAccountUIDKey:  string(un.GetUID()),
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}

	un1, err := util.SecretToUnstructured(secret)
	if err != nil {
		return err
	}
	token, err := dy.Resource(common.SecretsGVR).Namespace(namespace).Create(ctx, un1, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	sa, err := util.ServiceAccountFromUnstructured(un)
	if err != nil {
		return err
	}
	sa.Secrets = []corev1.ObjectReference{{Name: token.GetName()}}
	un, err = util.ServiceAccountToUnstructured(sa)
	_, err = dy.Resource(common.ServiceAccountGVR).Namespace(namespace).Update(ctx, un, metav1.UpdateOptions{})
	return err
}
