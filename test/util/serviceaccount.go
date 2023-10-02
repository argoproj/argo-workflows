package util

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/secrets"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CreateServiceAccountWithToken creates a service account with a given name with a service account token.
// Need to use this function to simulate the actual behavior of Kubernetes API server with the fake client.
func CreateServiceAccountWithToken(ctx context.Context, clientset kubernetes.Interface, namespace, name string) (*corev1.ServiceAccount, error) {
	sa, err := clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: name}}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secrets.NewTokenSecret(name),
		metav1.CreateOptions{})
	return sa, err
}
