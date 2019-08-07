package common

import (
	"github.com/argoproj/argo/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/serviceaccount"
)

// GetServiceAccountTokenByAccountName returns the name of the first referenced secret which is a ServiceAccountToken for the service account
func GetServiceAccountTokenByAccountName(clientset kubernetes.Interface, namespace, name string) (*corev1.Secret, error) {
	serviceAccount, err := clientset.CoreV1().ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	token, err := GetReferencedServiceAccountToken(clientset, serviceAccount)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, errors.Errorf("Error looking up service account token for %s/%s", serviceAccount.Namespace, serviceAccount.Name)
	}
	return token, nil
}

// GetReferencedServiceAccountToken returns the name of the first referenced secret which is a ServiceAccountToken for the service account
func GetReferencedServiceAccountToken(clientset kubernetes.Interface, serviceAccount *corev1.ServiceAccount) (*corev1.Secret, error) {
	if len(serviceAccount.Secrets) == 0 {
		return nil, nil
	}

	tokens, err := GetServiceAccountTokens(clientset, serviceAccount)
	if err != nil {
		return nil, err
	}

	accountTokens := map[string]*corev1.Secret{}
	for _, token := range tokens {
		accountTokens[token.Name] = token
	}
	// Prefer secrets in the order they're referenced.
	for _, obj := range serviceAccount.Secrets {
		token, ok := accountTokens[obj.Name]
		if ok {
			return token, nil
		}
	}

	return nil, nil
}

// GetServiceAccountTokens returns all ServiceAccountToken secrets for the given ServiceAccount
func GetServiceAccountTokens(clientset kubernetes.Interface, serviceAccount *corev1.ServiceAccount) ([]*corev1.Secret, error) {
	secrets, err := clientset.CoreV1().Secrets(serviceAccount.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	tokens := []*corev1.Secret{}
	for _, secret := range secrets.Items {
		if secret.Type != corev1.SecretTypeServiceAccountToken {
			continue
		}

		if serviceaccount.IsServiceAccountToken(&secret, serviceAccount) {
	        // The variable `secret` is overwritten during the loop, so need to deep copy it.
			tokens = append(tokens, secret.DeepCopy())
		}
	}
	return tokens, nil
}
