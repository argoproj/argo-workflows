package informer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func TestResourceVersionUnchanged(t *testing.T) {
	pod := func(rv string) *apiv1.Pod {
		return &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{ResourceVersion: rv}}
	}
	assert.True(t, ResourceVersionUnchanged(pod("1"), pod("1")))
	assert.False(t, ResourceVersionUnchanged(pod("1"), pod("2")))
	assert.False(t, ResourceVersionUnchanged("not an object", pod("1")))
	assert.False(t, ResourceVersionUnchanged(pod("1"), "not an object"))
}

func TestUntombstone(t *testing.T) {
	pod := &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-pod"}}
	assert.Same(t, pod, Untombstone(pod))
	assert.Same(t, pod, Untombstone(cache.DeletedFinalStateUnknown{Key: "ns/my-pod", Obj: pod}))
}
