package common

import (
	"github.com/argoproj/argo/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetServiceAccountTokenName returns the name of the first referenced ServiceAccountToken secret of the service account.
func GetServiceAccountTokenName(clientset kubernetes.Interface, namespace, name string) (string, error) {
	serviceAccount, err := clientset.CoreV1().ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	if len(serviceAccount.Secrets) == 0 {
		return "", errors.Errorf("Service account %s/%s does not have any token", serviceAccount.Namespace, serviceAccount.Name)
	}
	return serviceAccount.Secrets[0].Name, nil
}
