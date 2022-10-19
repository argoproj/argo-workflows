package secrets

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewTokenSecret creates a new secret struct.
func NewTokenSecret(name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        TokenName(name),
			Annotations: map[string]string{corev1.ServiceAccountNameKey: name},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
}
