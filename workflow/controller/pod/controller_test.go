package pod

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	argoConfig "github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/metrics"
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

// recordingQueue records the cleanup keys queued, at once or after a delay.
type recordingQueue struct {
	workqueue.TypedRateLimitingInterface[string]
	queued []string
}

func (q *recordingQueue) AddRateLimited(key string) {
	q.queued = append(q.queued, key)
	q.TypedRateLimitingInterface.AddRateLimited(key)
}

func (q *recordingQueue) AddAfter(key string, duration time.Duration) {
	q.queued = append(q.queued, key+" after "+duration.String())
	q.TypedRateLimitingInterface.AddAfter(key, duration)
}

// newTestController returns a pod controller whose informers hold the workflow, if any, and the pods, and which
// does not run.
func newTestController(t *testing.T, wf *unstructured.Unstructured, pods ...*apiv1.Pod) (*Controller, *recordingQueue) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	wfInformer := cache.NewSharedIndexInformer(&cache.ListWatch{}, &unstructured.Unstructured{}, 0, cache.Indexers{})
	if wf != nil {
		require.NoError(t, wfInformer.GetIndexer().Add(wf))
	}
	m, _, err := metrics.CreateDefaultTestMetrics(ctx)
	require.NoError(t, err)
	c := NewController(ctx, &argoConfig.Config{}, nil, "", fake.NewClientset(), wfInformer, m, nil)
	for _, pod := range pods {
		require.NoError(t, c.kubeclientset.(*fake.Clientset).Tracker().Add(pod))
		require.NoError(t, c.podInformer.GetIndexer().Add(pod))
	}
	queue := &recordingQueue{TypedRateLimitingInterface: c.workqueue}
	c.workqueue = queue
	t.Cleanup(queue.ShutDown)
	return c, queue
}

func testWorkflow(mutate func(wf *unstructured.Unstructured)) *unstructured.Unstructured {
	wf := &unstructured.Unstructured{}
	wf.SetNamespace("test")
	wf.SetName("my-wf")
	wf.SetUID("wf-uid")
	if mutate != nil {
		mutate(wf)
	}
	return wf
}

// deletingPod returns a pod of my-wf with the status finalizer and the given deletionTimestamp, whose last
// condition transition is older than any cleanup delay.
func deletingPod(phase apiv1.PodPhase, deleted time.Time) *apiv1.Pod {
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "test",
			Name:              "my-pod",
			UID:               "pod-uid",
			Labels:            map[string]string{common.LabelKeyWorkflow: "my-wf"},
			Finalizers:        []string{common.FinalizerPodStatus},
			DeletionTimestamp: new(metav1.NewTime(deleted)),
			OwnerReferences:   []metav1.OwnerReference{*metav1.NewControllerRef(testWorkflow(nil), wfv1.WorkflowSchemaGroupVersionKind)},
		},
		Status: apiv1.PodStatus{
			Phase:      phase,
			Conditions: []apiv1.PodCondition{{Type: apiv1.PodReady, LastTransitionTime: metav1.NewTime(time.Now().Add(-10 * time.Minute))}},
		},
	}
}
