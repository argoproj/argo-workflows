package parameterization

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// ValidateExternalSecretLease bounds stale external authorization. This is a
// metadata verification window, not a promise to revoke a running process's env.
func ValidateExternalSecretLease(secret *corev1.Secret, now time.Time) error {
	const source = Group + "/external-source"
	const validity = Group + "/external-valid-until"
	if secret.Labels[source] == "" && secret.Annotations[validity] == "" {
		return nil
	}
	until, err := time.Parse(time.RFC3339Nano, secret.Annotations[validity])
	if secret.Labels[source] == "" || err != nil || !now.Before(until) {
		return fmt.Errorf("external secret version verification expired; refresh the registered backend before execution")
	}
	return nil
}
