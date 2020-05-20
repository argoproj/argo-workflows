package rbac

import corev1 "k8s.io/api/core/v1"

type Interface interface {
	ServiceAccount(groups []string) (*corev1.LocalObjectReference, error)
}
