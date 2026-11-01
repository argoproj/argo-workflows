package parameterization

import (
	"context"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// ValidatePodExternalSecretLeases checks the concrete Pod at native dispatch.
// It never resolves a latest alias, changes a Secret reference or logs values.
// It cannot revoke environment variables already injected by the kubelet.
func ValidatePodExternalSecretLeases(ctx context.Context, pod *corev1.Pod, get func(context.Context, string) (*corev1.Secret, error)) error {
	refs := map[string]bool{}
	add := func(name string, optional *bool) {
		value := optional != nil && *optional
		if previous, found := refs[name]; found {
			value = value && previous
		}
		refs[name] = value
	}
	containers := append([]corev1.Container{}, pod.Spec.InitContainers...)
	containers = append(containers, pod.Spec.Containers...)
	for _, c := range containers {
		for _, e := range c.Env {
			if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
				ref := e.ValueFrom.SecretKeyRef
				add(ref.Name, ref.Optional)
			}
		}
		for _, e := range c.EnvFrom {
			if e.SecretRef != nil {
				add(e.SecretRef.Name, e.SecretRef.Optional)
			}
		}
	}
	for _, volume := range pod.Spec.Volumes {
		if volume.Secret != nil {
			add(volume.Secret.SecretName, volume.Secret.Optional)
		}
		if volume.Projected != nil {
			for _, source := range volume.Projected.Sources {
				if source.Secret != nil {
					add(source.Secret.Name, source.Secret.Optional)
				}
			}
		}
	}
	for _, ref := range pod.Spec.ImagePullSecrets {
		add(ref.Name, nil)
	}
	if len(refs) > 64 {
		return fmt.Errorf("parameterized Pod exceeds the 64 Secret reference verification limit")
	}
	names := make([]string, 0, len(refs))
	for name := range refs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		secret, err := get(ctx, name)
		if apierrors.IsNotFound(err) && refs[name] {
			continue
		}
		if err != nil || secret == nil {
			return fmt.Errorf("workflow credential metadata is unavailable at Pod dispatch")
		}
		if err = ValidateExternalSecretLease(secret, time.Now()); err != nil {
			return err
		}
	}
	return nil
}
