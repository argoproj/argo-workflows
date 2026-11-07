package secrets

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestTokenNameForServiceAccountWithSecretGetter(t *testing.T) {
	ctx := context.Background()
	secrets := map[string]*corev1.Secret{
		"image-pull-secret": {
			ObjectMeta: metav1.ObjectMeta{Name: "image-pull-secret"},
			Type:       corev1.SecretTypeDockerConfigJson,
		},
		"my-token": {
			ObjectMeta: metav1.ObjectMeta{Name: "my-token"},
			Type:       corev1.SecretTypeServiceAccountToken,
		},
	}
	getSecret := func(_ context.Context, name string) (*corev1.Secret, error) {
		secret, ok := secrets[name]
		if !ok {
			return nil, errors.New("not found")
		}
		return secret, nil
	}

	t.Run("selects first service account token secret", func(t *testing.T) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "my-sa"},
			Secrets: []corev1.ObjectReference{
				{Name: "image-pull-secret"},
				{Name: "my-token"},
			},
		}
		got, err := TokenNameForServiceAccountWithSecretGetter(ctx, sa, getSecret)
		require.NoError(t, err)
		assert.Equal(t, "my-token", got)
	})

	t.Run("falls back to annotation when no token secret is referenced", func(t *testing.T) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "my-sa",
				Annotations: map[string]string{"workflows.argoproj.io/service-account-token.name": "annotated-token"},
			},
			Secrets: []corev1.ObjectReference{{Name: "image-pull-secret"}},
		}
		got, err := TokenNameForServiceAccountWithSecretGetter(ctx, sa, getSecret)
		require.NoError(t, err)
		assert.Equal(t, "annotated-token", got)
	})

	t.Run("falls back to generated name when no token secret is referenced", func(t *testing.T) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "my-sa"},
			Secrets:    []corev1.ObjectReference{{Name: "image-pull-secret"}},
		}
		got, err := TokenNameForServiceAccountWithSecretGetter(ctx, sa, getSecret)
		require.NoError(t, err)
		assert.Equal(t, "my-sa.service-account-token", got)
	})

	t.Run("ignores empty secret references", func(t *testing.T) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "my-sa"},
			Secrets:    []corev1.ObjectReference{{}},
		}
		got, err := TokenNameForServiceAccountWithSecretGetter(ctx, sa, getSecret)
		require.NoError(t, err)
		assert.Equal(t, "my-sa.service-account-token", got)
	})

	t.Run("returns get errors", func(t *testing.T) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "my-sa"},
			Secrets:    []corev1.ObjectReference{{Name: "missing-secret"}},
		}
		_, err := TokenNameForServiceAccountWithSecretGetter(ctx, sa, getSecret)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing-secret")
	})
}
