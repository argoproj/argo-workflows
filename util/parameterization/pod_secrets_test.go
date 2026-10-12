package parameterization

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
)

func leaseTestSecret(valid bool) *corev1.Secret {
	expires := time.Now().Add(time.Hour)
	if !valid {
		expires = time.Now().Add(-time.Hour)
	}
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{Group + "/external-source": "synthetic-source"}, Annotations: map[string]string{Group + "/external-valid-until": expires.Format(time.RFC3339Nano)}}}
}

func TestPodExternalSecretLeaseReferences(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{
		InitContainers: []corev1.Container{{Env: []corev1.EnvVar{{ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: "a"}}}}}}},
		Containers:     []corev1.Container{{EnvFrom: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "b"}}}, {SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "a"}}}}}},
		Volumes: []corev1.Volume{
			{VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "c"}}},
			{VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: []corev1.VolumeProjection{{Secret: &corev1.SecretProjection{LocalObjectReference: corev1.LocalObjectReference{Name: "d"}}}}}}},
		},
		ImagePullSecrets: []corev1.LocalObjectReference{{Name: "e"}},
	}}
	before := pod.DeepCopy()
	var reads []string
	get := func(_ context.Context, name string) (*corev1.Secret, error) {
		reads = append(reads, name)
		if name == "c" {
			return &corev1.Secret{}, nil // Ordinary Kubernetes Secrets retain native semantics.
		}
		return leaseTestSecret(true), nil
	}
	require.NoError(t, ValidatePodExternalSecretLeases(t.Context(), pod, get))
	require.Equal(t, []string{"a", "b", "c", "d", "e"}, reads)
	require.Equal(t, before, pod)
}

func TestPodExternalSecretLeaseRequiredAndOptional(t *testing.T) {
	for _, test := range []struct {
		name                            string
		optional, alsoRequired, expired bool
		wantError                       bool
	}{
		{name: "optional absent", optional: true},
		{name: "required absent", wantError: true},
		{name: "mixed reference remains required", optional: true, alsoRequired: true, wantError: true},
		{name: "optional present expired", optional: true, expired: true, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			pod := &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{{EnvFrom: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: "version"}, Optional: ptr.To(test.optional)}}}}}}}
			if test.alsoRequired {
				pod.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "version"}}
			}
			get := func(context.Context, string) (*corev1.Secret, error) {
				if test.expired {
					return leaseTestSecret(false), nil
				}
				return nil, apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, "version")
			}
			err := ValidatePodExternalSecretLeases(t.Context(), pod, get)
			if test.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestPodExternalSecretLeaseErrorsAndBounds(t *testing.T) {
	pod := &corev1.Pod{Spec: corev1.PodSpec{ImagePullSecrets: []corev1.LocalObjectReference{{Name: "version"}}}}
	get := func(context.Context, string) (*corev1.Secret, error) {
		return nil, fmt.Errorf("synthetic-provider-secret-must-not-leak")
	}
	err := ValidatePodExternalSecretLeases(t.Context(), pod, get)
	require.EqualError(t, err, "workflow credential metadata is unavailable at Pod dispatch")
	get = func(context.Context, string) (*corev1.Secret, error) { return leaseTestSecret(false), nil }
	require.ErrorContains(t, ValidatePodExternalSecretLeases(t.Context(), pod, get), "verification expired")
	for i := range 64 {
		pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: fmt.Sprintf("version-%d", i)})
	}
	reads := 0
	get = func(context.Context, string) (*corev1.Secret, error) { reads++; return nil, nil }
	require.ErrorContains(t, ValidatePodExternalSecretLeases(t.Context(), pod, get), "64 Secret reference")
	require.Zero(t, reads)
}
