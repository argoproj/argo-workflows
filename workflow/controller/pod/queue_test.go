package pod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/workflow/common"
)

func TestParsePodCleanupKey(t *testing.T) {
	tests := []struct {
		name              string
		key               podCleanupKey
		wantNamespace     string
		wantPodName       string
		wantAction        podCleanupAction
		wantUID           string
	}{
		{
			name:          "standard key without UID",
			key:           "default/my-pod/deletePod",
			wantNamespace: "default",
			wantPodName:   "my-pod",
			wantAction:    deletePod,
			wantUID:       "",
		},
		{
			name:          "key with UID",
			key:           "default/my-pod/deletePodByUID/abc-123-def",
			wantNamespace: "default",
			wantPodName:   "my-pod",
			wantAction:    deletePodByUID,
			wantUID:       "abc-123-def",
		},
		{
			name:          "invalid key - too few parts",
			key:           "default/my-pod",
			wantNamespace: "",
			wantPodName:   "",
			wantAction:    "",
			wantUID:       "",
		},
		{
			name:          "invalid key - too many parts",
			key:           "default/my-pod/deletePod/uid/extra",
			wantNamespace: "",
			wantPodName:   "",
			wantAction:    "",
			wantUID:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace, podName, action, uid := parsePodCleanupKey(tt.key)
			assert.Equal(t, tt.wantNamespace, namespace)
			assert.Equal(t, tt.wantPodName, podName)
			assert.Equal(t, tt.wantAction, action)
			assert.Equal(t, tt.wantUID, uid)
		})
	}
}

func TestNewPodCleanupKeyWithUID(t *testing.T) {
	key := newPodCleanupKeyWithUID("my-namespace", "my-pod", deletePodByUID, "pod-uid-123")
	assert.Equal(t, "my-namespace/my-pod/deletePodByUID/pod-uid-123", key)

	// Verify round-trip
	namespace, podName, action, uid := parsePodCleanupKey(key)
	assert.Equal(t, "my-namespace", namespace)
	assert.Equal(t, "my-pod", podName)
	assert.Equal(t, deletePodByUID, action)
	assert.Equal(t, "pod-uid-123", uid)
}

func TestPodCleanupPatch(t *testing.T) {
	c := &Controller{}

	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          map[string]string{common.LabelKeyCompleted: "false"},
			Finalizers:      []string{common.FinalizerPodStatus},
			ResourceVersion: "123456",
		},
	}

	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "true")

	// pod finalizer enabled, patch label
	patch, err := c.getPodCleanupPatch(pod, true)
	require.NoError(t, err)
	expected := `{"metadata":{"resourceVersion":"123456","finalizers":[],"labels":{"workflows.argoproj.io/completed":"true"}}}`
	assert.JSONEq(t, expected, string(patch))

	// pod finalizer enabled, do not patch label
	patch, err = c.getPodCleanupPatch(pod, false)
	require.NoError(t, err)
	expected = `{"metadata":{"resourceVersion":"123456","finalizers":[]}}`
	assert.JSONEq(t, expected, string(patch))

	// pod finalizer enabled, do not patch label, nil/empty finalizers
	podWithNilFinalizers := &apiv1.Pod{}
	patch, err = c.getPodCleanupPatch(podWithNilFinalizers, false)
	require.NoError(t, err)
	assert.Nil(t, patch)

	t.Setenv(common.EnvVarPodStatusCaptureFinalizer, "false")

	// pod finalizer disabled, patch both
	patch, err = c.getPodCleanupPatch(pod, true)
	require.NoError(t, err)
	expected = `{"metadata":{"labels":{"workflows.argoproj.io/completed":"true"}}}`
	assert.JSONEq(t, expected, string(patch))

	// pod finalizer disabled, do not patch label
	patch, err = c.getPodCleanupPatch(pod, false)
	require.NoError(t, err)
	assert.Nil(t, patch)
}
