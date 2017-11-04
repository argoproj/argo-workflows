package common

import (
	"strings"

	wfv1 "github.com/argoproj/argo/api/workflow/v1"
	apiv1 "k8s.io/api/core/v1"
)

// FindOverlappingVolume looks an artifact path, checks if it overlaps with any
// user specified volumeMounts in the template, and returns the deepest volumeMount
// (if any).
func FindOverlappingVolume(tmpl *wfv1.Template, path string) *apiv1.VolumeMount {
	var volMnt *apiv1.VolumeMount
	deepestLen := 0
	for _, mnt := range tmpl.Container.VolumeMounts {
		if !strings.HasPrefix(path, mnt.MountPath) {
			continue
		}
		if len(mnt.MountPath) > deepestLen {
			volMnt = &mnt
			deepestLen = len(mnt.MountPath)
		}
	}
	return volMnt
}
