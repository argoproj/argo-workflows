package pod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func TestPodGCFromPod(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		want        wfv1.PodGC
	}{
		{
			name:        "no annotation",
			annotations: nil,
			want:        wfv1.PodGC{Strategy: wfv1.PodGCOnPodNone},
		},
		{
			name:        "valid annotation",
			annotations: map[string]string{common.AnnotationKeyPodGCStrategy: "OnPodCompletion/5m"},
			want:        wfv1.PodGC{Strategy: "OnPodCompletion", DeleteDelayDuration: "5m"},
		},
		{
			name:        "no slash",
			annotations: map[string]string{common.AnnotationKeyPodGCStrategy: "OnPodCompletion"},
			want:        wfv1.PodGC{Strategy: "OnPodCompletion", DeleteDelayDuration: ""},
		},
		{
			name:        "empty value",
			annotations: map[string]string{common.AnnotationKeyPodGCStrategy: ""},
			want:        wfv1.PodGC{Strategy: "", DeleteDelayDuration: ""},
		},
		{
			name:        "multiple slashes",
			annotations: map[string]string{common.AnnotationKeyPodGCStrategy: "OnPodCompletion/5m/extra"},
			want:        wfv1.PodGC{Strategy: "OnPodCompletion", DeleteDelayDuration: "5m/extra"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := &apiv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: tt.annotations,
				},
			}
			got := podGCFromPod(pod)
			assert.Equal(t, tt.want, got)
		})
	}
}
