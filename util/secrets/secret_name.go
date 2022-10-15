package secrets

import (
	"fmt"

	"github.com/argoproj/argo-workflows/v3/workflow/common"

	corev1 "k8s.io/api/core/v1"
)

func ServiceAccountTokenName(sa *corev1.ServiceAccount) string {
	if len(sa.Secrets) > 0 {
		return sa.Secrets[0].Name
	}
	if v, ok := sa.Annotations[common.AnnotationKeyServiceAccountTokenName]; ok {
		return v
	}
	return fmt.Sprintf("%s.service-account-token", sa.Name)
}
