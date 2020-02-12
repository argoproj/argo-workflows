package v1alpha1

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWorkflows(t *testing.T) {
	wfs := Workflows{
		{ObjectMeta: v1.ObjectMeta{Name: "3"}, Status: WorkflowStatus{FinishedAt: v1.NewTime(time.Time{}.Add(1))}},
		{ObjectMeta: v1.ObjectMeta{Name: "2"}, Status: WorkflowStatus{FinishedAt: v1.NewTime(time.Time{}.Add(0))}},
		{ObjectMeta: v1.ObjectMeta{Name: "1"}, Status: WorkflowStatus{StartedAt: v1.NewTime(time.Time{}.Add(0))}},
		{ObjectMeta: v1.ObjectMeta{Name: "0"}, Status: WorkflowStatus{StartedAt: v1.NewTime(time.Time{}.Add(1))}},
	}
	sort.Sort(wfs)
	assert.Equal(t, "0", wfs[0].Name)
	assert.Equal(t, "1", wfs[1].Name)
	assert.Equal(t, "2", wfs[2].Name)
	assert.Equal(t, "3", wfs[3].Name)
}

func TestNodes_FindByDisplayName(t *testing.T) {
	assert.Nil(t, Nodes{}.FindByDisplayName(""))
	assert.NotNil(t, Nodes{"": NodeStatus{DisplayName: "foo"}}.FindByDisplayName("foo"))
}

func TestUsage(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		assert.Equal(t, Usage{}.String(), "")
		assert.Equal(t, Usage{corev1.ResourceMemory: NewResourceUsage(1 * time.Second)}.String(), "1s*memory")
	})
	t.Run("Add", func(t *testing.T) {
		assert.Equal(t, Usage{}.Add(Usage{}).String(), "")
		assert.Equal(t, Usage{corev1.ResourceMemory: NewResourceUsage(1 * time.Second)}.Add(Usage{corev1.ResourceMemory: NewResourceUsage(1 * time.Second)}).String(), "2s*memory")
	})
}

func TestResourceUsage(t *testing.T) {
	assert.Equal(t, ResourceUsage(1), NewResourceUsage(1*time.Second))
	assert.Equal(t, "1s", NewResourceUsage(1*time.Second).String())
}

func TestNodes_GetUsage(t *testing.T) {
	assert.Equal(t, Usage{}, Nodes{}.GetUsage())
	assert.Equal(t, Usage{corev1.ResourceMemory: 3}, Nodes{
		"foo": NodeStatus{Usage: Usage{corev1.ResourceMemory: 1}},
		"bar": NodeStatus{Usage: Usage{corev1.ResourceMemory: 2}},
	}.GetUsage())
}
