package v1alpha1

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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

func TestS3Bucket_MergeInto(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		(&S3Bucket{}).MergeInto(nil)
	})
	t.Run("Endpoint", func(t *testing.T) {
		b := &S3Artifact{}
		(&S3Bucket{Endpoint: "my-endpoint"}).MergeInto(b)
		assert.Equal(t, "my-endpoint", b.Endpoint)
	})
	t.Run("Bucket", func(t *testing.T) {
		b := &S3Artifact{}
		(&S3Bucket{Bucket: "my-bucket"}).MergeInto(b)
		assert.Equal(t, "my-bucket", b.Bucket)
	})
	t.Run("Region", func(t *testing.T) {
		b := &S3Artifact{}
		(&S3Bucket{Region: "my-region"}).MergeInto(b)
		assert.Equal(t, "my-region", b.Region)
	})
	t.Run("Insecure", func(t *testing.T) {
		b := &S3Artifact{}
		(&S3Bucket{Insecure: pointer.BoolPtr(false)}).MergeInto(b)
		assert.NotNil(t, b.Insecure)
	})
	t.Run("AccessKeySecret", func(t *testing.T) {
		b := &S3Artifact{}
		assert.Empty(t, b.AccessKeySecret)
		(&S3Bucket{AccessKeySecret: corev1.SecretKeySelector{Key: "my-key"}}).MergeInto(b)
		assert.NotEmpty(t, b.AccessKeySecret)
	})
	t.Run("SecretKeySecret", func(t *testing.T) {
		b := &S3Artifact{}
		(&S3Bucket{SecretKeySecret: corev1.SecretKeySelector{Key: "my-key"}}).MergeInto(b)
		assert.NotEmpty(t, b.SecretKeySecret)
	})
	t.Run("RoleARN", func(t *testing.T) {
		b := &S3Artifact{}
		(&S3Bucket{RoleARN: "my-role-arn"}).MergeInto(b)
		assert.Equal(t, "my-role-arn", b.RoleARN)
	})
	t.Run("UseSDKCreds", func(t *testing.T) {
		b := &S3Artifact{}
		(&S3Bucket{UseSDKCreds: true}).MergeInto(b)
		assert.True(t, b.UseSDKCreds)
	})
}

func TestNodes_FindByDisplayName(t *testing.T) {
	assert.Nil(t, Nodes{}.FindByDisplayName(""))
	assert.NotNil(t, Nodes{"": NodeStatus{DisplayName: "foo"}}.FindByDisplayName("foo"))
}

func TestNodes_Any(t *testing.T) {
	assert.False(t, Nodes{"": NodeStatus{Name: "foo"}}.Any(func(node NodeStatus) bool { return node.Name == "bar" }))
	assert.True(t, Nodes{"": NodeStatus{Name: "foo"}}.Any(func(node NodeStatus) bool { return node.Name == "foo" }))
}

func TestResourcesDuration(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		assert.Equal(t, ResourcesDuration{}.String(), "")
		assert.Equal(t, ResourcesDuration{corev1.ResourceMemory: NewResourceDuration(1 * time.Second)}.String(), "1s*(100Mi memory)")
	})
	t.Run("Add", func(t *testing.T) {
		assert.Equal(t, ResourcesDuration{}.Add(ResourcesDuration{}).String(), "")
		assert.Equal(t, ResourcesDuration{corev1.ResourceMemory: NewResourceDuration(1 * time.Second)}.
			Add(ResourcesDuration{corev1.ResourceMemory: NewResourceDuration(1 * time.Second)}).
			String(), "2s*(100Mi memory)")
	})
	t.Run("CPUAndMemory", func(t *testing.T) {
		assert.Equal(t, ResourcesDuration{}.Add(ResourcesDuration{}).String(), "")
		s := ResourcesDuration{corev1.ResourceCPU: NewResourceDuration(2 * time.Second)}.
			Add(ResourcesDuration{corev1.ResourceMemory: NewResourceDuration(1 * time.Second)}).
			String()
		assert.Contains(t, s, "1s*(100Mi memory)")
		assert.Contains(t, s, "2s*(1 cpu)")
	})
}

func TestResourceDuration(t *testing.T) {
	assert.Equal(t, ResourceDuration(1), NewResourceDuration(1*time.Second))
	assert.Equal(t, "1s", NewResourceDuration(1*time.Second).String())
}

func TestNodes_GetResourcesDuration(t *testing.T) {
	assert.Equal(t, ResourcesDuration{}, Nodes{}.GetResourcesDuration())
	assert.Equal(t, ResourcesDuration{corev1.ResourceMemory: 3}, Nodes{
		"foo": NodeStatus{ResourcesDuration: ResourcesDuration{corev1.ResourceMemory: 1}},
		"bar": NodeStatus{ResourcesDuration: ResourcesDuration{corev1.ResourceMemory: 2}},
	}.GetResourcesDuration())
}

func TestWorkflowConditions_UpsertConditionMessage(t *testing.T) {
	wfCond := WorkflowConditions{WorkflowCondition{Type: WorkflowConditionCompleted, Message: "Hello"}}
	wfCond.UpsertConditionMessage(WorkflowCondition{Type: WorkflowConditionCompleted, Message: "world!"})
	assert.Equal(t, "Hello, world!", wfCond[0].Message)
}

func TestShutdownStrategy_ShouldExecute(t *testing.T) {
	assert.False(t, ShutdownStrategyTerminate.ShouldExecute(true))
	assert.False(t, ShutdownStrategyTerminate.ShouldExecute(false))
	assert.False(t, ShutdownStrategyStop.ShouldExecute(false))
	assert.True(t, ShutdownStrategyStop.ShouldExecute(true))
}
