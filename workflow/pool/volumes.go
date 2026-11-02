package pool

import (
	"slices"

	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

// MountVolume resolves a mount of tmpl's called name, for pods
// (addVolumeReferences in the controller) and pooled templates alike: to the
// template's own volume of that name, or else to the one of volumes, the
// workflow's, own saying which. It is nil for neither, which for a pod would
// be one of the workflow's volumeClaimTemplates.
func MountVolume(tmpl *wfv1.Template, volumes []apiv1.Volume, name string) (volume *apiv1.Volume, own bool) {
	byName := func(v apiv1.Volume) bool { return v.Name == name }
	if i := slices.IndexFunc(tmpl.Volumes, byName); i >= 0 {
		return &tmpl.Volumes[i], true
	}
	if i := slices.IndexFunc(volumes, byName); i >= 0 {
		return &volumes[i], false
	}
	return nil, false
}

// MountedWorkflowVolumes resolves tmpl's volume mounts (MountVolume). It
// returns the workflow volumes mounted, each once, in mount order, and the
// names of the mounts of neither the template's nor the workflow's
// volumes, in mount order.
func MountedWorkflowVolumes(tmpl *wfv1.Template, volumes []apiv1.Volume) (mounted []apiv1.Volume, unresolved []string) {
	seen := make(map[string]bool)
	for _, m := range tmpl.GetVolumeMounts() {
		if seen[m.Name] {
			continue
		}
		seen[m.Name] = true
		switch v, own := MountVolume(tmpl, volumes, m.Name); {
		case own:
		case v == nil:
			unresolved = append(unresolved, m.Name)
		default:
			mounted = append(mounted, *v.DeepCopy())
		}
	}
	return mounted, unresolved
}
