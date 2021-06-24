package controller

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func TestContainerSetTemplate(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: pod
spec:
  entrypoint: main
  templates:
    - name: main
      volumes:
       - name: workspace
         emptyDir: { }
      containerSet:
        volumeMounts:
          - name: workspace
            mountPath: /workspace
        containers:
          - name: ctr-0
            image: argoproj/argosay:v2
`)
	cancel, controller := newController(wf)
	defer cancel()

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(context.Background())

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Len(t, woc.wf.Status.Nodes, 2)

	pod, err := getPod(woc, "pod")
	assert.NoError(t, err)

	socket := corev1.HostPathSocket
	assert.ElementsMatch(t, []corev1.Volume{
		{Name: "docker-sock", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/run/docker.sock", Type: &socket}}},
		{Name: "workspace", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}, pod.Spec.Volumes)

	assert.Empty(t, pod.Spec.InitContainers)

	assert.Len(t, pod.Spec.Containers, 2)
	for _, c := range pod.Spec.Containers {
		switch c.Name {
		case common.WaitContainerName:
			assert.ElementsMatch(t, []corev1.VolumeMount{
				{Name: "docker-sock", MountPath: "/var/run/docker.sock", ReadOnly: true},
			}, c.VolumeMounts)
		case "ctr-0":
			assert.ElementsMatch(t, []corev1.VolumeMount{
				{Name: "workspace", MountPath: "/workspace"},
			}, c.VolumeMounts)
		default:
			t.Fatalf(c.Name)
		}
	}
}

func TestContainerSetTemplateWithInputArtifacts(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: pod
spec:
  entrypoint: main
  templates:
    - name: main
      inputs:
        artifacts:
         - name: in-0
           path: /in/in-0
           raw:
             data: hi
         - name: in-1
           path: /workspace/in-1
           raw:
             data: hi
      volumes:
       - name: workspace
         emptyDir: { }
      containerSet:
        volumeMounts:
          - name: workspace
            mountPath: /workspace
        containers:
          - name: main
            image: argoproj/argosay:v2
`)
	cancel, controller := newController(wf)
	defer cancel()

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(context.Background())

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Len(t, woc.wf.Status.Nodes, 2)

	pod, err := getPod(woc, "pod")
	assert.NoError(t, err)

	socket := corev1.HostPathSocket
	assert.ElementsMatch(t, []corev1.Volume{
		{Name: "docker-sock", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/run/docker.sock", Type: &socket}}},
		{Name: "workspace", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		{Name: "input-artifacts", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}, pod.Spec.Volumes)

	if assert.Len(t, pod.Spec.InitContainers, 1) {
		c := pod.Spec.InitContainers[0]
		assert.ElementsMatch(t, []corev1.VolumeMount{
			{Name: "input-artifacts", MountPath: "/argo/inputs/artifacts"},
			{Name: "workspace", MountPath: "/mainctrfs/workspace"},
		}, c.VolumeMounts)
	}

	assert.Len(t, pod.Spec.Containers, 2)
	for _, c := range pod.Spec.Containers {
		switch c.Name {
		case common.WaitContainerName:
			assert.ElementsMatch(t, []corev1.VolumeMount{
				{Name: "docker-sock", MountPath: "/var/run/docker.sock", ReadOnly: true},
				{Name: "workspace", MountPath: "/mainctrfs/workspace"},
				{Name: "input-artifacts", MountPath: "/mainctrfs/in/in-0", SubPath: "in-0"},
			}, c.VolumeMounts)
		case "main":
			assert.ElementsMatch(t, []corev1.VolumeMount{
				{Name: "workspace", MountPath: "/workspace"},
				{Name: "input-artifacts", MountPath: "/in/in-0", SubPath: "in-0"},
			}, c.VolumeMounts)
		default:
			t.Fatalf(c.Name)
		}
	}
}

func TestContainerSetTemplateWithOutputArtifacts(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: pod
spec:
  entrypoint: main
  templates:
    - name: main
      volumes:
       - name: workspace
         emptyDir: { }
      containerSet:
        volumeMounts:
          - name: workspace
            mountPath: /workspace
        containers:
          - name: main
            image: argoproj/argosay:v2
      outputs:
        artifacts:
         - name: in-0
           path: /in/in-0
           raw:
             data: hi
         - name: in-1
           path: /workspace/in-1
           raw:
             data: hi
`)
	cancel, controller := newController(wf)
	defer cancel()

	woc := newWorkflowOperationCtx(wf, controller)
	woc.operate(context.Background())

	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.Len(t, woc.wf.Status.Nodes, 2)

	pod, err := getPod(woc, "pod")
	assert.NoError(t, err)

	socket := corev1.HostPathSocket
	assert.ElementsMatch(t, []corev1.Volume{
		{Name: "docker-sock", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/var/run/docker.sock", Type: &socket}}},
		{Name: "workspace", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}, pod.Spec.Volumes)

	assert.Len(t, pod.Spec.InitContainers, 0)

	assert.Len(t, pod.Spec.Containers, 2)
	for _, c := range pod.Spec.Containers {
		switch c.Name {
		case common.WaitContainerName:
			assert.ElementsMatch(t, []corev1.VolumeMount{
				{Name: "docker-sock", MountPath: "/var/run/docker.sock", ReadOnly: true},
				{Name: "workspace", MountPath: "/mainctrfs/workspace"},
			}, c.VolumeMounts)
		case "main":
			assert.ElementsMatch(t, []corev1.VolumeMount{
				{Name: "workspace", MountPath: "/workspace"},
			}, c.VolumeMounts)
		default:
			t.Fatalf(c.Name)
		}
	}
}
