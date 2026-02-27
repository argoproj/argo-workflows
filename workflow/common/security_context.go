package common

import (
	corev1 "k8s.io/api/core/v1"
)

func MinimalCtrSC() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		Privileged:               new(false),
		RunAsNonRoot:             new(true),
		RunAsUser:                new(int64(8737)),
		ReadOnlyRootFilesystem:   new(true),
		AllowPrivilegeEscalation: new(false),
		SeccompProfile:           &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
}

func MinimalPodSC() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		RunAsNonRoot:   new(true),
		RunAsUser:      new(int64(8737)),
		SeccompProfile: &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
}
