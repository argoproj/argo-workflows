package common

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func MinimalCtrSC() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		Privileged:               ptr.To(false),
		RunAsNonRoot:             ptr.To(true),
		RunAsUser:                ptr.To(int64(8737)),
		ReadOnlyRootFilesystem:   ptr.To(true),
		AllowPrivilegeEscalation: ptr.To(false),
		SeccompProfile:           &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
}

func MinimalPodSC() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		RunAsNonRoot:   ptr.To(true),
		RunAsUser:      ptr.To(int64(8737)),
		SeccompProfile: &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
}
