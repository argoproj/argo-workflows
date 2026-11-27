package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func TestMountedWorkflowVolumes(t *testing.T) {
	volume := func(name string) apiv1.Volume {
		return apiv1.Volume{Name: name, VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}}
	}
	tmpl := &wfv1.Template{
		Volumes: []apiv1.Volume{volume("own")},
		Container: &apiv1.Container{VolumeMounts: []apiv1.VolumeMount{
			{Name: "b", MountPath: "/b"},
			{Name: "own", MountPath: "/own"},
			{Name: "claim", MountPath: "/claim"},
			{Name: "a", MountPath: "/a"},
			{Name: "b", MountPath: "/b2"},
			{Name: "claim", MountPath: "/claim2"},
		}},
	}
	mounted, unresolved := MountedWorkflowVolumes(tmpl, []apiv1.Volume{volume("a"), volume("b"), volume("unmounted")})
	assert.Equal(t, []apiv1.Volume{volume("b"), volume("a")}, mounted, "in mount order, each once, without the template's own")
	assert.Equal(t, []string{"claim"}, unresolved)
}

func TestMountVolume(t *testing.T) {
	own := apiv1.Volume{Name: "shared", VolumeSource: apiv1.VolumeSource{EmptyDir: &apiv1.EmptyDirVolumeSource{}}}
	workflows := []apiv1.Volume{
		{Name: "shared", VolumeSource: apiv1.VolumeSource{HostPath: &apiv1.HostPathVolumeSource{Path: "/"}}},
		{Name: "wf", VolumeSource: apiv1.VolumeSource{HostPath: &apiv1.HostPathVolumeSource{Path: "/wf"}}},
	}
	tmpl := &wfv1.Template{Volumes: []apiv1.Volume{own}}

	v, isOwn := MountVolume(tmpl, workflows, "shared")
	assert.Equal(t, &own, v, "the template's own volume wins over the workflow's of the same name")
	assert.True(t, isOwn)

	v, isOwn = MountVolume(tmpl, workflows, "wf")
	assert.Equal(t, &workflows[1], v)
	assert.False(t, isOwn)

	v, isOwn = MountVolume(tmpl, workflows, "claim")
	assert.Nil(t, v)
	assert.False(t, isOwn)
}
