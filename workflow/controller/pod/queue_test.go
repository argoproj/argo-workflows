package pod

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestParsePodCleanupKey(t *testing.T) {
	tests := []struct {
		name          string
		key           podCleanupKey
		wantNamespace string
		wantPodName   string
		wantAction    podCleanupAction
		wantUID       string
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

	key = newPodCleanupKeyWithUID("my-namespace", "my-pod", removeFinalizer, "pod-uid-123")
	assert.Equal(t, "my-namespace/my-pod/removeFinalizer/pod-uid-123", key)
	_, _, action, uid = parsePodCleanupKey(key)
	assert.Equal(t, removeFinalizer, action)
	assert.Equal(t, "pod-uid-123", uid)
}

func TestPodCleanupPatch(t *testing.T) {
	for _, flag := range []string{"true", "false", "unset"} {
		t.Run(flag, func(t *testing.T) {
			t.Setenv(common.EnvVarPodStatusCaptureFinalizer, flag)
			if flag == "unset" {
				require.NoError(t, os.Unsetenv(common.EnvVarPodStatusCaptureFinalizer))
			}

			for _, tt := range []struct {
				name              string
				finalizers        []string
				labelPodCompleted bool
				wantPatch         string
			}{
				{
					name:              "remove status finalizer and label completed",
					finalizers:        []string{common.FinalizerPodStatus},
					labelPodCompleted: true,
					wantPatch:         `{"metadata":{"resourceVersion":"123456","finalizers":[],"labels":{"workflows.argoproj.io/completed":"true"}}}`,
				},
				{
					name:       "remove status finalizer without labeling",
					finalizers: []string{common.FinalizerPodStatus},
					wantPatch:  `{"metadata":{"resourceVersion":"123456","finalizers":[]}}`,
				},
				{
					name:              "preserve other finalizers when labeling",
					finalizers:        []string{"example.com/retain", common.FinalizerPodStatus},
					labelPodCompleted: true,
					wantPatch:         `{"metadata":{"resourceVersion":"123456","finalizers":["example.com/retain"],"labels":{"workflows.argoproj.io/completed":"true"}}}`,
				},
				{
					name:       "preserve other finalizers without labeling",
					finalizers: []string{common.FinalizerPodStatus, "example.com/retain"},
					wantPatch:  `{"metadata":{"resourceVersion":"123456","finalizers":["example.com/retain"]}}`,
				},
				{
					name:       "other finalizer only",
					finalizers: []string{"example.com/retain"},
				},
				{
					name:              "label completed without removing other finalizers",
					finalizers:        []string{"example.com/retain"},
					labelPodCompleted: true,
					wantPatch:         `{"metadata":{"labels":{"workflows.argoproj.io/completed":"true"}}}`,
				},
				{name: "nil finalizers"},
				{name: "empty finalizers", finalizers: []string{}},
			} {
				t.Run(tt.name, func(t *testing.T) {
					pod := &apiv1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Labels:          map[string]string{common.LabelKeyCompleted: "false"},
							Finalizers:      tt.finalizers,
							ResourceVersion: "123456",
						},
					}
					original := pod.DeepCopy()
					c := &Controller{}
					patch, err := c.getPodCleanupPatch(pod, tt.labelPodCompleted)
					require.NoError(t, err)
					if tt.wantPatch == "" {
						assert.Nil(t, patch)
					} else {
						assert.JSONEq(t, tt.wantPatch, string(patch))
					}
					assert.Equal(t, original, pod, "cleanup must not mutate the informer-cached pod")
				})
			}
		})
	}
}
