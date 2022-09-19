package secrets

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func SecretName(serviceAccount *corev1.ServiceAccount) string {
	secretName := fmt.Sprintf("%s.service-account-token", serviceAccount.Name)
	if len(serviceAccount.Secrets) > 0 {
		secretName = serviceAccount.Secrets[0].Name
	}
	return secretName
}
