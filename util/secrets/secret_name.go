package secrets

import (
	"fmt"

	"github.com/argoproj/argo-workflows/v4/workflow/common"

	corev1 "k8s.io/api/core/v1"
)

// TokenNameForServiceAccount returns the name of the secret container the access token for the service account
func TokenNameForServiceAccount(sa *corev1.ServiceAccount) string {
	if len(sa.Secrets) > 0 {
		return sa.Secrets[0].Name
	}
	if v, ok := sa.Annotations[common.AnnotationKeyServiceAccountTokenName]; ok {
		return v
	}
	return TokenName(sa.Name)
}

func TokenName(name string) string {
	return fmt.Sprintf("%s.service-account-token", name)
}
