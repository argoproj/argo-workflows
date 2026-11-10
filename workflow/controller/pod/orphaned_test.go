package pod

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestPodOrphaned(t *testing.T) {
	informer := cache.NewSharedIndexInformer(&cache.ListWatch{}, &unstructured.Unstructured{}, 0, cache.Indexers{})
	now := metav1.Now()
	for name, deleting := range map[string]bool{"live": false, "deleting": true} {
		wf := &unstructured.Unstructured{}
		wf.SetKind(workflow.WorkflowKind)
		wf.SetNamespace("ns")
		wf.SetName(name)
		if deleting {
			wf.SetDeletionTimestamp(&now)
		}
		require.NoError(t, informer.GetIndexer().Add(wf))
	}
	c := &Controller{wfInformer: informer, log: logging.NewSlogLoggerCustom(logging.Info, logging.JSON, io.Discard)}

	ownedBy := func(kind, name string) *apiv1.Pod {
		isController := true
		return &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{
			Namespace:       "ns",
			Name:            "pod",
			OwnerReferences: []metav1.OwnerReference{{Kind: kind, Name: name, Controller: &isController}},
		}}
	}
	for _, tc := range []struct {
		name string
		pod  *apiv1.Pod
		want bool
	}{
		{name: "no owner", pod: &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "pod"}}, want: false},
		{name: "owned by something else", pod: ownedBy("Job", "live"), want: false},
		{name: "workflow is gone", pod: ownedBy(workflow.WorkflowKind, "missing"), want: true},
		{name: "workflow is live", pod: ownedBy(workflow.WorkflowKind, "live"), want: false},
		{name: "workflow is being deleted", pod: ownedBy(workflow.WorkflowKind, "deleting"), want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, c.podOrphaned(context.Background(), tc.pod))
		})
	}
}
