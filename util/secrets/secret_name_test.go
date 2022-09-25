package secrets

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestSecretName(t *testing.T) {
	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "sa-name"},
	}
	assert.Equal(t, "sa-name.service-account-token", SecretName(&sa))
	sa.Secrets = []corev1.ObjectReference{{Name: "existing-secret"}}
	assert.Equal(t, "existing-secret", SecretName(&sa))
}
