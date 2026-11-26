package secrets

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v4/workflow/common"

	corev1 "k8s.io/api/core/v1"
)

// TokenNameForServiceAccount returns the name of the secret container the access token for the service account
func TokenNameForServiceAccount(sa *corev1.ServiceAccount) string {
	if len(sa.Secrets) > 0 {
		return sa.Secrets[0].Name
	}
	return fallbackTokenNameForServiceAccount(sa)
}

// TokenNameForServiceAccountWithSecretGetter returns the first referenced service account token secret name.
func TokenNameForServiceAccountWithSecretGetter(ctx context.Context, sa *corev1.ServiceAccount, getSecret func(context.Context, string) (*corev1.Secret, error)) (string, error) {
	if getSecret == nil {
		return TokenNameForServiceAccount(sa), nil
	}
	for _, ref := range sa.Secrets {
		if ref.Name == "" {
			continue
		}
		secret, err := getSecret(ctx, ref.Name)
		if err != nil {
			return "", fmt.Errorf("failed to get secret %q referenced by service account %q: %w", ref.Name, sa.Name, err)
		}
		if secret.Type == corev1.SecretTypeServiceAccountToken {
			return ref.Name, nil
		}
	}
	return fallbackTokenNameForServiceAccount(sa), nil
}

func fallbackTokenNameForServiceAccount(sa *corev1.ServiceAccount) string {
	if v, ok := sa.Annotations[common.AnnotationKeyServiceAccountTokenName]; ok {
		return v
	}
	return TokenName(sa.Name)
}

func TokenName(name string) string {
	return fmt.Sprintf("%s.service-account-token", name)
}
