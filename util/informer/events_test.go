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

func TestChangeHandlers(t *testing.T) {
	pod := func(rv string) *apiv1.Pod {
		return &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-pod", ResourceVersion: rv}}
	}
	var seen []any
	handlers := ChangeHandlers(func(obj any) { seen = append(seen, obj) })
	added, changed, tombstone := pod("1"), pod("2"), cache.DeletedFinalStateUnknown{Key: "ns/my-pod", Obj: pod("2")}
	handlers.OnAdd(added, false)
	handlers.OnUpdate(added, pod("1"))
	handlers.OnUpdate(added, changed)
	handlers.OnDelete(tombstone)
	assert.Equal(t, []any{added, changed, tombstone}, seen, "a replay of an unchanged object is skipped")
}

func TestUntombstone(t *testing.T) {
	pod := &apiv1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "my-pod"}}
	assert.Same(t, pod, Untombstone(pod))
	assert.Same(t, pod, Untombstone(cache.DeletedFinalStateUnknown{Key: "ns/my-pod", Obj: pod}))
}
