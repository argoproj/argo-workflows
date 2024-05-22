package common

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
)

func MinimalCtrSC() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
		Privileged:               pointer.Bool(false),
		RunAsNonRoot:             pointer.Bool(true),
		RunAsUser:                pointer.Int64(8737),
		ReadOnlyRootFilesystem:   pointer.Bool(true),
		AllowPrivilegeEscalation: pointer.Bool(false),
		SeccompProfile:           &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
}

func MinimalPodSC() *corev1.PodSecurityContext {
	return &corev1.PodSecurityContext{
		RunAsNonRoot:   pointer.Bool(true),
		RunAsUser:      pointer.Int64(8737),
		SeccompProfile: &corev1.SeccompProfile{Type: "RuntimeDefault"},
	}
}
