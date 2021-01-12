package common

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/argoproj/argo/errors"
)

// GetServiceAccountTokenName returns the name of the first referenced ServiceAccountToken secret of the service account.
func GetServiceAccountTokenName(ctx context.Context, clientset dynamic.Interface, namespace, name string) (string, error) {
	un, err := clientset.Resource(schema.GroupVersionResource{Version: "v1", Resource: "serviceaccounts"}).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	serviceAccount := v1.ServiceAccount{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, serviceAccount)
	if err != nil {
		return "", err
	}
	if len(serviceAccount.Secrets) == 0 {
		return "", errors.Errorf("", "Service account %s/%s does not have any token", serviceAccount.GetNamespace(), serviceAccount.GetName())
	}
	return serviceAccount.Secrets[0].Name, nil
}
