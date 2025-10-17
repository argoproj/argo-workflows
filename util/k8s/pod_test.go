package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func TestGetCurrentPodName(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	t.Run("Returns pod name from environment variable", func(t *testing.T) {
		t.Setenv(common.EnvVarPodName, "test-pod-from-env")

		client := fake.NewSimpleClientset()
		podName, err := GetCurrentPodName(ctx, client, "test-namespace", "app=test")

		require.NoError(t, err)
		assert.Equal(t, "test-pod-from-env", podName)
	})

	t.Run("Falls back to Kubernetes client when env var not set", func(t *testing.T) {
		// Ensure env var is not set
		t.Setenv(common.EnvVarPodName, "")

		// Create a fake pod
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-from-k8s",
				Namespace: "test-namespace",
				Labels: map[string]string{
					"app": "test",
				},
			},
			Status: v1.PodStatus{
				Phase: v1.PodRunning,
			},
		}

		client := fake.NewSimpleClientset(pod)
		podName, err := GetCurrentPodName(ctx, client, "test-namespace", "app=test")

		require.NoError(t, err)
		assert.Equal(t, "test-pod-from-k8s", podName)
	})

	t.Run("Returns first pod when no running pods found", func(t *testing.T) {
		t.Setenv(common.EnvVarPodName, "")

		// Create a fake pod that's not running
		pod := &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod-pending",
				Namespace: "test-namespace",
				Labels: map[string]string{
					"app": "test",
				},
			},
			Status: v1.PodStatus{
				Phase: v1.PodPending,
			},
		}

		client := fake.NewSimpleClientset(pod)
		podName, err := GetCurrentPodName(ctx, client, "test-namespace", "app=test")

		require.NoError(t, err)
		assert.Equal(t, "test-pod-pending", podName)
	})

	t.Run("Returns error when no pods found", func(t *testing.T) {
		t.Setenv(common.EnvVarPodName, "")

		client := fake.NewSimpleClientset()
		_, err := GetCurrentPodName(ctx, client, "test-namespace", "app=nonexistent")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no pods found with selector")
	})
}
