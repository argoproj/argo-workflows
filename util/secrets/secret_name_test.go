package secrets

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceAccountTokenName(t *testing.T) {
	type args struct {
		sa *corev1.ServiceAccount
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"discovery by secret (Kubernetes <v1.24)",
			args{&corev1.ServiceAccount{Secrets: []corev1.ObjectReference{{Name: "my-token"}}}},
			"my-token",
		},
		{
			"discovery by annotation (Kubernetes >=v1.24)",
			args{&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"workflows.argoproj.io/service-account-token.name": "my-token"}},
			}},
			"my-token",
		},
		{
			"discovery by name (Kubernetes >=v1.24)",
			args{&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: "my-name"},
			}},
			"my-name.service-account-token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TokenNameForServiceAccount(tt.args.sa); got != tt.want {
				t.Errorf("ServiceAccountTokenName() = %v, want %v", got, tt.want)
			}
		})
	}
}
