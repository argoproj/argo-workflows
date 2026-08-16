package signal

import (
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func podWithContainer(mountPaths ...string) *corev1.Pod {
	mounts := make([]corev1.VolumeMount, len(mountPaths))
	for i, p := range mountPaths {
		mounts[i] = corev1.VolumeMount{MountPath: p}
	}
	return &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: common.MainContainerName, VolumeMounts: mounts}},
		},
	}
}

func TestKillCommand(t *testing.T) {
	t.Run("init-less main reaches argoexec via the image volume", func(t *testing.T) {
		// init-less main mounts both the shared emptyDir and the argoexec-bin image
		// volume; the image-volume path must win — /var/run/argo/argoexec does not
		// exist in init-less mode (no init container copies it).
		pod := podWithContainer(common.VarRunArgoPath, common.ArgoExecBinMountPath)
		cmd, err := killCommand(pod, common.MainContainerName, syscall.SIGTERM)
		require.NoError(t, err)
		assert.Equal(t, []string{common.ArgoExecBinPath, "kill", "15", "1"}, cmd)
	})

	t.Run("legacy main reaches argoexec via the shared emptyDir", func(t *testing.T) {
		pod := podWithContainer(common.VarRunArgoPath)
		cmd, err := killCommand(pod, common.MainContainerName, syscall.SIGTERM)
		require.NoError(t, err)
		assert.Equal(t, []string{common.LegacyArgoExecBinPath, "kill", "15", "1"}, cmd)
	})

	t.Run("no argo mount falls back to /bin/sh kill", func(t *testing.T) {
		pod := podWithContainer()
		cmd, err := killCommand(pod, common.MainContainerName, syscall.SIGKILL)
		require.NoError(t, err)
		assert.Equal(t, []string{"/bin/sh", "-c", "kill -9 1"}, cmd)
	})

	t.Run("kill-command annotation overrides", func(t *testing.T) {
		pod := podWithContainer(common.ArgoExecBinMountPath)
		pod.Annotations = map[string]string{
			common.AnnotationKeyKillCmd(common.MainContainerName): `["/custom","kill","%d"]`,
		}
		cmd, err := killCommand(pod, common.MainContainerName, syscall.SIGTERM)
		require.NoError(t, err)
		assert.Equal(t, []string{"/custom", "kill", "15"}, cmd)
	})

	t.Run("invalid annotation errors", func(t *testing.T) {
		pod := podWithContainer()
		pod.Annotations = map[string]string{common.AnnotationKeyKillCmd(common.MainContainerName): `not-json`}
		_, err := killCommand(pod, common.MainContainerName, syscall.SIGTERM)
		require.Error(t, err)
	})
}
