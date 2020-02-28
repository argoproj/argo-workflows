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

func TestNodes_Any(t *testing.T) {
	assert.False(t, Nodes{"": NodeStatus{Name: "foo"}}.Any(func(node NodeStatus) bool { return node.Name == "bar" }))
	assert.True(t, Nodes{"": NodeStatus{Name: "foo"}}.Any(func(node NodeStatus) bool { return node.Name == "foo" }))
}

func TestUsage(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		assert.Equal(t, UsageIndicator{}.String(), "")
		assert.Equal(t, UsageIndicator{corev1.ResourceMemory: NewResourceUsageIndicator(1 * time.Second)}.String(), "1s*memory")
	})
	t.Run("Add", func(t *testing.T) {
		assert.Equal(t, UsageIndicator{}.Add(UsageIndicator{}).String(), "")
		assert.Equal(t, UsageIndicator{corev1.ResourceMemory: NewResourceUsageIndicator(1 * time.Second)}.Add(UsageIndicator{corev1.ResourceMemory: NewResourceUsageIndicator(1 * time.Second)}).String(), "2s*memory")
	})
}

func TestResourceUsage(t *testing.T) {
	assert.Equal(t, ResourceUsageIndicator(1), NewResourceUsageIndicator(1*time.Second))
	assert.Equal(t, "1s", NewResourceUsageIndicator(1*time.Second).String())
}

func TestNodes_GetUsage(t *testing.T) {
	assert.Equal(t, UsageIndicator{}, Nodes{}.GetUsageIndicator())
	assert.Equal(t, UsageIndicator{corev1.ResourceMemory: 3}, Nodes{
		"foo": NodeStatus{UsageIndicator: UsageIndicator{corev1.ResourceMemory: 1}},
		"bar": NodeStatus{UsageIndicator: UsageIndicator{corev1.ResourceMemory: 2}},
	}.GetUsageIndicator())
}
